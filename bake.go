package bake

import (
	"bytes"
	"io"
	"os"
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

// Target represents a buildable rule.
type Target struct {
	Name     string // e.g. "test"
	Commands []Command
	Inputs   []string // dependencies
	Outputs  []string // declared outputs
}

// Command represents an executable command.
type Command interface{}

// ExecCommand represents a command that is executed against the OS's exec().
type ExecCommand struct {
	Text string
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
