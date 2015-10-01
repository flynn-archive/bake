package bake

import (
	"io"
)

// Execer represents an object that can execute commands.
type Execer interface {
	Exec(cmd string) Command
}

// Command represents command execution. It is returned by the Execer.
// These can be local commands or can be executed remotely.
type Command interface {
	// Output streams from command.
	Stdout() io.Reader
	Stderr() io.Reader

	// Returns once the command has finished executing.
	// A nil error is returned if the command was successful.
	Wait() error

	// Files read from and written to.
	// These are not available until after Wait() has returned.
	Inputs() []string
	Outputs() []string
}
