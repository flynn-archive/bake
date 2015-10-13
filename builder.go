package bake

import (
	"errors"
)

var (
	// ErrDependency is set on a build if a dependent build has an error.
	ErrDependency = errors.New("dependency error")

	// ErrCanceled is set on a build if the build was canceled early.
	ErrCanceled = errors.New("build canceled")
)

// Builder processes targets to produce an output Build.
type Builder struct {
}

// NewBuilder returns a new instance of Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build recursively executes build steps in parallel.
func (b *Builder) Build(build *Build) {
	closing := make(chan struct{})

	// Build top-level build.
	b.build(build, closing)

	// If an error occurred then send a cancel signal to all nested builds.
	if build.err != nil {
		close(closing)
	}
}

func (b *Builder) build(build *Build, closing <-chan struct{}) {
	ch := make(chan *Build)

	// Build dependencies in order.
	for _, subbuild := range build.Dependencies() {
		go func(subbuild *Build) {
			b.build(subbuild, closing)
			subbuild.Wait()

			ch <- subbuild
		}(subbuild)
	}

	// Check all dependencies for errors.
	for range build.Dependencies() {
		select {
		case subbuild := <-ch:
			if err := subbuild.Err(); err != nil {
				build.Done(ErrDependency)
				return
			}
		case <-closing:
			build.Done(ErrCanceled)
			return
		}
	}

	// Execute build after dependencies are finished.
	// cmd := b.Execer.Exec(build.Command)

	// Mark build as finished with no error.
	build.Done(nil)
}
