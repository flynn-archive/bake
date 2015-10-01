package bake_test

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/flynn/bake"
)

// Ensure the builder can build a target in a project with no dependencies.
func TestBuilder_Build_NoDependencies(t *testing.T) {
	// Create builder and mock exec call.
	b := NewBuilder()
	b.Importer.ImportFn = func(name string) (*bake.Package, error) {
		if name != "myproj" {
			t.Fatalf("unexpected name: %s", name)
		}
		return &bake.Package{
			Name: "myproj",
			Targets: []*bake.Target{
				{
					Command: "go build -o bin/foo .",
					Outputs: []string{"bin/foo"},
				},
			},
		}, nil
	}
	b.Execer.ExecFn = func(cmd string) bake.Command {
		if cmd != "go build -o bin/foo ." {
			t.Fatalf("unexpected command: %s", cmd)
		}
		return &Command{stdout: "OK"}
	}

	// Build target and verify output.
	build := b.Build("myproj#bin/foo")
	if !reflect.DeepEqual(build, &bake.Build{
		Label:   bake.Label{Package: "myproj", Target: "bin/foo"},
		Outputs: []string{"bin/foo"},
	}) {
		t.Fatalf("unexpected build: %s", spew.Sdump(build))
	}
}

// Ensure the builder can build a target in a project with with nested dependencies.
func TestBuilder_Build_WithDependencies(t *testing.T) {
	// Create builder and mock exec call.
	b := NewBuilder()
	b.Importer.ImportFn = func(name string) (*bake.Package, error) {
		switch name {
		case "A":
			return &bake.Package{
				Name: "A",
				Targets: []*bake.Target{
					{
						Command: "make",
						Inputs:  []string{"B#site.css"},
						Outputs: []string{"main.exe"},
					},
				},
			}, nil
		case "B":
			return &bake.Package{
				Name: "B",
				Targets: []*bake.Target{
					{
						Command: "lessc site.less site.css",
						Outputs: []string{"site.css"},
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected import name: %q", name)
			return nil, nil
		}
	}
	b.Execer.ExecFn = func(cmd string) bake.Command { return &Command{} }

	// Build target and verify output.
	build := b.Build("A#main.exe")
	if !reflect.DeepEqual(build, &bake.Build{
		Label:   bake.Label{Package: "A", Target: "main.exe"},
		Outputs: []string{"main.exe"},
		Dependencies: []*bake.Build{
			{
				Label:   bake.Label{Package: "B", Target: "site.css"},
				Outputs: []string{"site.css"},
			},
		},
	}) {
		t.Fatalf("unexpected build: %s", spew.Sdump(build))
	}
}

// Builder represents a mockable test wrapper for bake.Builder.
type Builder struct {
	*bake.Builder
	Importer Importer
	Execer   Execer
}

// NewBuilder returns a new instance of Builder.
func NewBuilder() *Builder {
	f, _ := ioutil.TempFile("", "bake-")
	f.Close()
	os.RemoveAll(f.Name())

	b := &Builder{Builder: bake.NewBuilder(f.Name())}
	b.Builder.Importer = &b.Importer
	b.Builder.Execer = &b.Execer
	return b
}

// Close stops the builder and removes the underlying data.
func (b *Builder) Close() error {
	os.RemoveAll(b.Path())
	return nil
}
