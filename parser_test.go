package bake_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/flynn/bake"
)

// Ensure a dependency-less target can be parsed.
func TestParser_Parse_NoDependencies(t *testing.T) {
	path := MustTempDir()
	defer MustRemoveAll(path)

	MustWriteFile(filepath.Join(path, "Bakefile.lua"), []byte(`
target("bin/flynn-host", function()
	exec "run1.sh"
	exec "run2.sh"
end)
`))

	// Parse directory.
	p := bake.NewParser()
	if err := p.ParseDir(path); err != nil {
		t.Fatal(err)
	}

	// Parse directory and retrieve target from package.
	target := p.Package.Target("bin/flynn-host")
	if target == nil {
		t.Fatal("expected target")
	} else if len(target.Dependencies) != 0 {
		t.Fatalf("unexpected dependencies: %v", target.Dependencies)
	} else if !reflect.DeepEqual(target.Commands, []bake.Command{
		&bake.ExecCommand{Args: []string{"run1.sh"}},
		&bake.ExecCommand{Args: []string{"run2.sh"}},
	}) {
		t.Fatalf("unexpected commands: %s", spew.Sdump(target.Commands))
	}
}

// Ensure a target can be parsed with dependencies.
func TestParser_Parse_Dependencies(t *testing.T) {
	path := MustTempDir()
	defer MustRemoveAll(path)

	MustWriteFile(filepath.Join(path, "Bakefile.lua"), []byte(`
target("bin/flynn-host", depends("A", "B"), function() end)
`))

	// Parse directory.
	p := bake.NewParser()
	if err := p.ParseDir(path); err != nil {
		t.Fatal(err)
	}

	// Retrieve target from package.
	target := p.Package.Target("bin/flynn-host")
	if target == nil {
		t.Fatal("expected target")
	} else if !reflect.DeepEqual(target.Dependencies, []string{"A", "B"}) {
		t.Fatalf("unexpected denpendencies: %v", target.Dependencies)
	}
}

// MustTempDir returns a path to a temporary directory. Panic on error.
func MustTempDir() string {
	path, err := ioutil.TempDir("", "bake-")
	if err != nil {
		panic(err)
	}
	return path
}

// MustRemoveAll recursively deletes a path. Panic on error.
func MustRemoveAll(path string) {
	if err := os.RemoveAll(path); err != nil {
		panic(err)
	}
}

// MustWriteFile writes data to filename. Panic on error.
func MustWriteFile(filename string, data []byte) {
	if err := os.MkdirAll(filepath.Dir(filename), 0777); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(filename, data, 0666); err != nil {
		panic(err)
	}
}
