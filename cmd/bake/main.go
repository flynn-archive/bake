package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/flynn/bake"
)

func main() {
	m := NewMain()
	if err := m.Run(os.Args[1:]...); err != nil {
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
	// Parse command line flags.
	opt, err := m.ParseFlags(args)
	if err != nil {
		return err
	}

	// Use present working directory for default project.
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %s", err)
	}

	// Generate work directory, if necessary.
	if opt.Work == "" {
		path, err := ioutil.TempDir("", "bake-")
		if err != nil {
			return fmt.Errorf("error generating work dir: %s", err)
		}
		opt.Work = path
	}

	// TODO: Parse package file.
	pkg := &bake.Package{}

	// TODO: Determine changeset.

	// Create planner.
	p := bake.NewPlanner(pkg)

	// Create build plan.
	build, err := p.Plan(opt.Targets, nil)
	if err != nil {
		return err
	} else if build == nil {
		fmt.Fprintln(m.Stderr, "nothing to build, exiting")
		return nil
	}

	// Execute build.
	b := bake.NewBuilder()
	if err := b.Build(build); err != nil {
		return err
	}

	return nil
}

// ParseFlags parses the command line flags and returns a set of options.
func (m *Main) ParseFlags(args []string) (Options, error) {
	var opt Options
	fs := flag.NewFlagSet("bake", flag.ContinueOnError)
	fs.SetOutput(m.Stderr)
	fs.StringVar(&opt.Work, "work", "", "temporary work directory")
	if err := fs.Parse(args); err != nil {
		return opt, err
	}

	// Retrieve targets from arg list.
	if fs.NArg() == 0 {
		return opt, errors.New("target required")
	}
	opt.Targets = fs.Args()

	return opt, nil
}

// Options represents a set of options passed to the program.
type Options struct {
	Targets []string // target name
	Pwd     string   // default project path
	Work    string   // temporary work directory
}
