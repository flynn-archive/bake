package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	// "github.com/davecgh/go-spew/spew"
	"github.com/flynn/bake"
)

// DefaultPath is the default path to begin parsing from.
const DefaultPath = "."

func main() {
	m := NewMain()
	if err := m.ParseFlags(os.Args[1:]); err != nil {
		fmt.Fprintln(m.Stderr, err)
		os.Exit(1)
	}

	if err := m.Run(); err != nil {
		fmt.Fprintln(m.Stderr, err)
		os.Exit(1)
	}
}

// Main represents the main program execution.
type Main struct {
	// List of targets to build.
	Targets []string // target name

	// Directory to start parsing from.
	Path string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// NewMain returns a new instance of Main.
func NewMain() *Main {
	return &Main{
		Path: DefaultPath,

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// Run executes the program.
func (m *Main) Run() error {
	// Save working directory to restore when program is done.
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(wd)

	// Change to new directory
	if err := os.Chdir(m.Path); err != nil {
		return err
	}

	// Parse build rules.
	parser := bake.NewParser()
	if err := parser.ParseDir(m.Path); err != nil {
		return err
	}
	pkg := parser.Package

	// Create planner.
	p := bake.NewPlanner(pkg)

	// Create build plan.
	build, err := p.Plan(m.Targets)
	if err != nil {
		return err
	} else if build == nil {
		fmt.Fprintln(m.Stderr, "nothing to build, exiting")
		return nil
	}
	defer build.Close()

	// Recursively attach all readers to stdout/stderr.
	m.pipeReaders(build, make(map[*bake.Build]struct{}))

	// Execute build.
	b := bake.NewBuilder()
	b.Build(build)

	if err := build.RootErr(); err != nil {
		return err
	}

	return nil
}

// ParseFlags parses the command line flags into fields on the program.
func (m *Main) ParseFlags(args []string) error {
	fs := flag.NewFlagSet("bake", flag.ContinueOnError)
	fs.SetOutput(m.Stderr)
	fs.StringVar(&m.Path, "C", DefaultPath, "working directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Retrieve targets from arg list.
	if fs.NArg() == 0 {
		return errors.New("target required")
	}
	m.Targets = fs.Args()

	return nil
}

// pipeReaders creates goroutines for all readers to copy to stderr & stdout.
func (m *Main) pipeReaders(build *bake.Build, set map[*bake.Build]struct{}) {
	// Ignore if the build has already been attached.
	if _, ok := set[build]; ok {
		return
	}
	set[build] = struct{}{}

	// NOTE: goroutines are closed automatically when build is closed.
	go io.Copy(m.Stdout, build.Stdout())
	go io.Copy(m.Stderr, build.Stderr())

	// Recursively pipe dependencies.
	for _, subbuild := range build.Dependencies() {
		m.pipeReaders(subbuild, set)
	}
}
