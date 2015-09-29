package bake

import (
	"errors"
	"fmt"
)

// ErrDependency is set on a build if a dependent build has an error.
var ErrDependency = errors.New("dependency error")

// Builder processes targets to produce an output Build.
type Builder struct {
	path    string          // working path
	targets map[Label]Build // completed builds by label

	// Imports packages by name.
	// Package caching and remote retrieval are implemented in the importer.
	Importer Importer

	// Executes commands and returns output.
	Execer Execer
}

// NewBuilder returns a new instance of Builder.
func NewBuilder(path string) *Builder {
	return &Builder{
		path: path,
	}
}

// Path returns the path that the builder was initialized with.
func (b *Builder) Path() string { return b.path }

// Build builds the target and outputs a build.
func (b *Builder) Build(label string) *Build {
	return b.buildTarget(label)
}

func (b *Builder) buildTarget(name string) (build *Build) {
	build = &Build{}

	// Parse label.
	label, err := ParseLabel(name)
	if err != nil {
		build.Err = fmt.Errorf("parse label: %s", err)
		return
	}
	build.Label = label

	// Import the package if it hasn't been loaded yet.
	pkg, err := b.Importer.Import(label.Package)
	if err != nil {
		build.Err = fmt.Errorf("import package: %s", err)
		return
	}

	// Lookup target by name.
	t := pkg.Target(label.Target)
	if t == nil {
		build.Err = fmt.Errorf("target not found: %s", label.Target)
		return
	}

	// Build dependencies first.
	for _, input := range t.Inputs {
		subbuild := b.buildTarget(input)
		build.Dependencies = append(build.Dependencies, subbuild)

		// Exit if dependency errors out.
		if subbuild.Err != nil {
			build.Err = ErrDependency
			return
		}
	}

	// Build target.
	cmd := b.Execer.Exec(t.Command)
	// TODO: Attach stdout, stderr.
	if err := cmd.Wait(); err != nil {
		build.Err = err
		return
	}

	// Copy outputs to build.
	build.Outputs = make([]string, len(t.Outputs))
	copy(build.Outputs, t.Outputs)

	// TODO: Remove intermediate files.

	return
}

// Build represents a build state for a target.
type Build struct {
	Label Label

	// Root path of the build.
	Path string

	// Relative paths to files output by the build.
	Outputs []string

	// Set if an error occurred during the build process.
	Err error

	// Builds that this build depends on.
	Dependencies []*Build
}
