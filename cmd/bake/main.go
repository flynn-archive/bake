package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	// "github.com/davecgh/go-spew/spew"
	"github.com/flynn/bake"
	_ "github.com/flynn/bake/filesystem"
)

const (
	// DefaultRoot is the default project path to begin parsing from.
	DefaultRoot = "."

	// DefaultFileSystem is the type of filesystem used to mount and track changes.
	DefaultFileSystem = "9p"

	// SnapshotFile is the directory a snapshot is store in within the data directory.
	// It's used to avoid issues with overlapping project paths.
	SnapshotFile = "__SNAPSHOT__"
)

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
	fs *bake.FileSystem

	// List of targets to build.
	Targets []string // target name

	// Forces all listed targets to be rebuilt when true.
	Force bool

	// Directory to start parsing from.
	Root string

	// Directory to store snapshot data.
	DataDir string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// NewMain returns a new instance of Main.
func NewMain() *Main {
	return &Main{
		Root: DefaultRoot,

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// ParseFlags parses the command line flags into fields on the program.
func (m *Main) ParseFlags(args []string) error {
	fs := flag.NewFlagSet("bake", flag.ContinueOnError)
	fs.SetOutput(m.Stderr)
	fs.BoolVar(&m.Force, "f", false, "force rebuild")
	fs.StringVar(&m.Root, "root", DefaultRoot, "project root")
	fs.StringVar(&m.DataDir, "data", "", "data directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Retrieve targets from arg list.
	m.Targets = fs.Args()

	// If no data directory is specified then use ~/.bake
	if m.DataDir == "" {
		u, err := user.Current()
		if err != nil {
			return errors.New("data directory not specified and current user unknown")
		} else if u.HomeDir == "" {
			return errors.New("data directory must be specified if no home directory exists")
		}
		m.DataDir = filepath.Join(u.HomeDir, ".bake")
	}

	return nil
}

// Run executes the program.
func (m *Main) Run() error {
	// Validate arguments.
	if m.Root == "" {
		return errors.New("project root required")
	} else if m.DataDir == "" {
		return errors.New("data directory required")
	}

	// Ensure root is an absolute path.
	root, err := filepath.Abs(m.Root)
	if err != nil {
		return fmt.Errorf("abs path: %s", err)
	}
	m.Root = root

	// Initialize snapshot.
	ss := bake.NewSnapshot(filepath.Join(m.DataDir, m.Root, SnapshotFile), m.Root)

	// Parse build rules.
	parser := bake.NewParser()
	if err := parser.ParseDir(m.Root); err != nil {
		return err
	}
	pkg := parser.Package

	// If no targets are specified then build all targets.
	if len(m.Targets) == 0 {
		m.Targets = pkg.TargetNames()
	}

	// Create planner. Only use snapshot if not force building.
	p := bake.NewPlanner(pkg)
	if !m.Force {
		p.Snapshot = ss
	}

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

	// Create mount directory.
	mountPath, err := ioutil.TempDir("", "bake-")
	if err != nil {
		return err
	}
	defer os.Remove(mountPath)

	// Create file system.
	fs, err := m.openFileSystem(mountPath)
	if err != nil {
		return err
	}
	defer m.closeFileSystem(fs)

	// Execute the build.
	if err := m.build(build, fs, ss); err != nil {
		return err
	}

	return nil
}

// openFileSystem initializes and mounts a file system to a temporary directory.
func (m *Main) openFileSystem(mountPath string) (bake.FileSystem, error) {
	// Create file system.
	fs, err := bake.NewFileSystem(DefaultFileSystem, bake.FileSystemOptions{
		Path:      m.Root,
		MountPath: mountPath,
	})
	if err != nil {
		return nil, fmt.Errorf("new file system: %s", err)
	}

	// Open file system.
	if err := fs.Open(); err != nil {
		return nil, fmt.Errorf("open file system: %s", err)
	}

	return fs, nil
}

// closeFileSystem shuts down and unmounts the file system.
// Also removes underlying mount directory.
func (m *Main) closeFileSystem(fs bake.FileSystem) error {
	if err := fs.Close(); err != nil {
		return err
	}
	return nil
}

// build executes a build against a file system.
func (m *Main) build(build *bake.Build, fs bake.FileSystem, ss *bake.Snapshot) error {
	// Execute build.
	b := bake.NewBuilder()
	b.FileSystem = fs
	b.Snapshot = ss
	b.Output = m.Stderr
	b.Build(build)

	if err := build.RootErr(); err != nil {
		return err
	}

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
