package bake_test

import (
	"github.com/flynn/bake"
)

// Importer represents a mock of bake.Importer.
type Importer struct {
	ImportFn func(name string) (*bake.Package, error)
}

func (i *Importer) Import(name string) (*bake.Package, error) { return i.ImportFn(name) }
