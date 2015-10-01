package bake_test

import (
	"io"
	"strings"

	"github.com/flynn/bake"
)

// Execer represents a mock of bake.Execer.
type Execer struct {
	ExecFn func(cmd string) bake.Command
}

func (e *Execer) Exec(cmd string) bake.Command { return e.ExecFn(cmd) }

// Command represents a mock bake.Command.
type Command struct {
	stdout string
	stderr string
	err    error

	inputs  []string
	outputs []string
}

func (c *Command) Stdout() io.Reader { return strings.NewReader(c.stdout) }
func (c *Command) Stderr() io.Reader { return strings.NewReader(c.stderr) }
func (c *Command) Wait() error       { return c.err }

func (c *Command) Inputs() []string  { return c.inputs }
func (c *Command) Outputs() []string { return c.outputs }
