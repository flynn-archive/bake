package main_test

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"testing"

	main "github.com/flynn/bake/cmd/bake"
)

func TestMain_ParseFlags_Target(t *testing.T) {
	m := NewMain()
	if err := m.ParseFlags([]string{"foo:bar"}); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(m.Targets, []string{"foo:bar"}) {
		t.Fatalf("unexpected targets: %+v", m.Targets)
	}
}

// Main represents a test wrapper for main.Main.
type Main struct {
	*main.Main
	Stdin  bytes.Buffer
	Stdout bytes.Buffer
	Stderr bytes.Buffer
}

// NewMain returns a new instance of Main.
func NewMain() *Main {
	m := &Main{Main: main.NewMain()}

	// Redirect streams to buffers.
	m.Main.Stdin = &m.Stdin
	m.Main.Stdout = &m.Stdout
	m.Main.Stderr = &m.Stderr

	// Pipe to standard streams, if verbose is specified.
	if testing.Verbose() {
		m.Main.Stdout = io.MultiWriter(os.Stdout, m.Main.Stdout)
		m.Main.Stderr = io.MultiWriter(os.Stderr, m.Main.Stderr)
	}

	return m
}
