package bake_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flynn/bake"
)

// Ensures that a target is marked as dirty if its dependencies change.
func TestSnapshot_IsTargetDirty_Dependencies(t *testing.T) {
	ss := NewSnapshot()
	defer ss.Close()

	target := &bake.Target{
		Name:         "T",
		Dependencies: []string{"X", "Y"},
	}

	// Add target.
	if err := ss.AddTarget(target, nil); err != nil {
		t.Fatal(err)
	}

	// Update dependencies and verify it's dirty.
	target.Dependencies = []string{"X"}
	if dirty, err := ss.IsTargetDirty(target); err != nil {
		t.Fatal(err)
	} else if !dirty {
		t.Fatal("expected dirty")
	}
}

// Ensures that a target is marked as dirty if its commands change.
func TestSnapshot_IsTargetDirty_Commands(t *testing.T) {
	ss := NewSnapshot()
	defer ss.Close()

	target := &bake.Target{
		Name: "T",
		Commands: []bake.Command{
			&bake.ExecCommand{Args: []string{"run.sh"}},
		},
	}

	// Add target.
	if err := ss.AddTarget(target, nil); err != nil {
		t.Fatal(err)
	}

	// Update command and verify it's dirty.
	target.Commands[0].(*bake.ExecCommand).Args[0] = "build.sh"
	if dirty, err := ss.IsTargetDirty(target); err != nil {
		t.Fatal(err)
	} else if !dirty {
		t.Fatal("expected dirty")
	}
}

// Ensures that a target is marked as dirty if its files change.
func TestSnapshot_IsTargetDirty_Files(t *testing.T) {
	t.Parallel()

	ss := NewSnapshot()
	defer ss.Close()

	// Initialize input files to a value.
	MustWriteFile(filepath.Join(ss.Root(), "a"), []byte("0"))
	MustWriteFile(filepath.Join(ss.Root(), "b"), []byte("1"))

	// Add target with input files.
	if err := ss.AddTarget(&bake.Target{Name: "T"}, []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}

	// Verify target is not dirty immediately after adding.
	if dirty, err := ss.IsTargetDirty(&bake.Target{Name: "T"}); err != nil {
		t.Fatal(err)
	} else if dirty {
		t.Fatal("expected not dirty")
	}

	// Wait for a second because of mtime resolution.
	time.Sleep(1 * time.Second)

	// Update one of the input files.
	MustWriteFile(filepath.Join(ss.Root(), "b"), []byte("2"))

	// Update command and verify it's dirty.
	if dirty, err := ss.IsTargetDirty(&bake.Target{Name: "T"}); err != nil {
		t.Fatal(err)
	} else if !dirty {
		t.Fatal("expected dirty")
	}
}

// Ensures that a target is marked as dirty if files are added to input directories.
func TestSnapshot_IsTargetDirty_Dirs(t *testing.T) {
	t.Parallel()

	ss := NewSnapshot()
	defer ss.Close()

	// Initialize input files to a value.
	MustWriteFile(filepath.Join(ss.Root(), "a/b"), []byte("0"))

	// Add target with input files.
	if err := ss.AddTarget(&bake.Target{Name: "T"}, []string{"a"}); err != nil {
		t.Fatal(err)
	}

	// Wait for a second because of mtime resolution.
	time.Sleep(1 * time.Second)

	// Update one of the input files.
	MustWriteFile(filepath.Join(ss.Root(), "a/c"), []byte("0"))

	// Update command and verify it's dirty.
	if dirty, err := ss.IsTargetDirty(&bake.Target{Name: "T"}); err != nil {
		t.Fatal(err)
	} else if !dirty {
		t.Fatal("expected dirty")
	}
}

// Snapshot represents a test wrapper for bake.Snapshot.
type Snapshot struct {
	*bake.Snapshot
}

// NewSnapshot returns a new instance of Snapshot backed by a temporary path.
func NewSnapshot() *Snapshot {
	path, err := ioutil.TempDir("", "bake-snapshot-path-")
	if err != nil {
		panic(err)
	}
	root, err := ioutil.TempDir("", "bake-snapshot-root-")
	if err != nil {
		panic(err)
	}
	return &Snapshot{Snapshot: bake.NewSnapshot(path, root)}
}

// Close removes underlying temporary path for the snapshot.
func (ss *Snapshot) Close() error {
	os.RemoveAll(ss.Path())
	os.RemoveAll(ss.Root())
	return nil
}
