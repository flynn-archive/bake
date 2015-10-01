package bake

// Importer imports a package, possibly from a remote location.
type Importer interface {
	Import(name string) (*Package, error)
}
