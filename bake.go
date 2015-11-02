package bake

import (
	"bytes"
	"io"
	"os"
	"path"
	"regexp"
	"sync"
)

//go:generate protoc --gogo_out=. internal/internal.proto

// Package represents a collection of targets.
type Package struct {
	Name    string
	Targets []*Target
}

// Target returns a target by name or by output.
func (p *Package) Target(name string) *Target {
	for _, t := range p.Targets {
		if t.Name == name {
			return t
		}

		for _, output := range t.Outputs {
			if output == name {
				return t
			}
		}
	}

	return nil
}

// MatchTargets returns a list of targets matching a glob pattern.
// Matches pattern against target name or outputs.
func (p *Package) MatchTargets(pattern string) ([]*Target, error) {
	var a []*Target
	for _, t := range p.Targets {
		if matched, err := MatchTarget(pattern, t); err != nil {
			return nil, err
		} else if matched {
			a = append(a, t)
		}
	}
	return a, nil
}

// Target represents a buildable rule.
type Target struct {
	// Unique identifier for the target within the package.
	Name string

	// Indicates that the target does not produce a file.
	// For example, "test" or "clean".
	Phony bool

	// Text shown to users instead of commands.
	Title string

	// The working directory that commands are run from.
	WorkDir string

	// The commands to execute to build the target.
	Commands []Command

	// Depedent target names.
	Inputs []string

	// Files to be retained after build.
	// Any files written that are not declared here are assumed to be temporary files.
	Outputs []string
}

// MatchTarget returns true if t's name or outputs match pattern.
func MatchTarget(pattern string, t *Target) (matched bool, err error) {
	if matched, err = path.Match(pattern, t.Name); matched || err != nil {
		return
	}

	for _, output := range t.Outputs {
		if matched, err = path.Match(pattern, output); matched || err != nil {
			return
		}
	}

	return
}

// Command represents an executable command.
type Command interface{}

// ExecCommand represents a command that is executed against the OS's exec().
type ExecCommand struct {
	Args []string
}

// File represents a physical file or directory in a package.
type File struct {
	Name     string
	Size     int64
	Mode     os.FileMode
	ModTime  int64
	Children map[string]File
}

// Label represents a reference to a project and target.
// An empty package represents the default package. An empty target represents
// the default target.
type Label struct {
	Package string
	Target  string
}

// String returns the string representation of l.
func (l Label) String() string {
	var buf bytes.Buffer
	if l.Package != "" {
		buf.WriteString(l.Package)
		buf.WriteByte(':')
	}
	buf.WriteString(l.Target)
	return buf.String()
}

var labelRegexp = regexp.MustCompile(`^(?:(\S*):)?(\S*)$`)

// ParseLabel parses a URI string into a label.
func ParseLabel(s string) Label {
	a := labelRegexp.FindStringSubmatch(s)
	return Label{Package: a[1], Target: a[2]}
}

// Build represents a build step.
type Build struct {
	mu sync.Mutex

	target *Target

	// Output streams, available during execution.
	stdout struct {
		reader *io.PipeReader
		writer *io.PipeWriter
	}
	stderr struct {
		reader *io.PipeReader
		writer *io.PipeWriter
	}

	err  error
	done chan struct{}

	dependencies []*Build
}

// newBuild creates a new build.
func newBuild(target *Target) *Build {
	b := &Build{
		target: target,
		done:   make(chan struct{}),
	}

	// Set up pipes for streaming command output.
	b.stdout.reader, b.stdout.writer = io.Pipe()
	b.stderr.reader, b.stderr.writer = io.Pipe()

	return b
}

// Close recursively closes the build's readers and its dependencies readers.
func (b *Build) Close() error {
	b.stdout.reader.Close()
	b.stderr.reader.Close()

	for _, subbuild := range b.dependencies {
		subbuild.Close()
	}

	return nil
}

// Name returns the target's name. Returns a blank string if a top-level build.
func (b *Build) Name() string {
	if b.target == nil {
		return ""
	}
	return b.target.Name
}

// Target returns the build target.
func (b *Build) Target() *Target { return b.target }

// Dependencies returns a list of builds that b depends on.
func (b *Build) Dependencies() []*Build { return b.dependencies }

// Wait blocks until the build has finished.
func (b *Build) Wait() { <-b.done }

// Err returns the error that occurred on the build, if any.
func (b *Build) Err() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.err
}

// RootErr recursively searches the build tree and finds the build error.
func (b *Build) RootErr() error {
	if err := b.Err(); err != nil && err != ErrDependency && err != ErrCanceled {
		return err
	}

	for _, subbuid := range b.dependencies {
		if err := subbuid.RootErr(); err != nil {
			return err
		}
	}

	return nil
}

// Done marks the build as complete and sets the error, if any.
// Calling this method twice on a build will cause it to panic.
func (b *Build) Done(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.err = err
	close(b.done)
}

// Stdout returns the standard output stream.
func (b *Build) Stdout() io.ReadCloser {
	return b.stdout.reader
}

// Stderr returns the standard error stream.
func (b *Build) Stderr() io.ReadCloser {
	return b.stderr.reader
}

// Builds represents a
type Builds []*Build

// dedupe returns a unique set of builds in a.
func (a Builds) dedupe() Builds {
	set := make(map[*Build]struct{})

	other := make([]*Build, 0, len(a))
	for _, b := range a {
		if _, ok := set[b]; ok {
			continue
		}

		other = append(other, b)
		set[b] = struct{}{}
	}
	return other
}
