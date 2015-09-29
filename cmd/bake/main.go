package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	m := NewMain()
	if err := m.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(m.Stderr, err)
		os.Exit(1)
	}
}

// Main represents the main program execution.
type Main struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// NewMain returns a new instance of Main.
func NewMain() *Main {
	return &Main{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// Run executes the program.
func (m *Main) Run(args ...string) error {
	panic("FIXME: parse flags")
	panic("FIXME: delegate to action")
}
