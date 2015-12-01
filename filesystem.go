package bake

import (
	"errors"
)

// ErrUnregisteredFileSystem is returned when a file system has not been registered.
var ErrUnregisteredFileSystem = errors.New("unregistered file system")

// FileSystem represents a way to mount the build filesystem and track reads & writes.
type FileSystem interface {
	Open() error
	Close() error

	Mount(target string) error
	Unmount(target string) error

	Readset() map[string]struct{}
	Writeset() map[string]struct{}
	Reset()
}

// lookup of file system constructors by type.
var newFileSystemFns = make(map[string]NewFileSystemFunc)

// NewFileSystem returns a new instance of FileSystem for a given type.
func NewFileSystem(typ string, opt FileSystemOptions) (FileSystem, error) {
	fn := newFileSystemFns[typ]
	if fn == nil {
		return nil, ErrUnregisteredFileSystem
	}
	return fn(opt)
}

// NewFileSystemFunc is a function for creating a new file system.
type NewFileSystemFunc func(FileSystemOptions) (FileSystem, error)

// RegisterFileSystem registers a filesystem constructor by type.
func RegisterFileSystem(typ string, fn NewFileSystemFunc) {
	if _, ok := newFileSystemFns[typ]; ok {
		panic("file system already registered: " + typ)
	}
	newFileSystemFns[typ] = fn
}

// FileSystemOptions represents a list of options passed to a file sytem on creation.
type FileSystemOptions struct {
	Root string
}

// nopFileSystem is a file system that does nothing.
type nopFileSystem struct{}

func (*nopFileSystem) Open() error  { return nil }
func (*nopFileSystem) Close() error { return nil }

func (*nopFileSystem) Mount(string) error   { return nil }
func (*nopFileSystem) Unmount(string) error { return nil }

func (*nopFileSystem) Readset() map[string]struct{}  { return nil }
func (*nopFileSystem) Writeset() map[string]struct{} { return nil }
func (*nopFileSystem) Reset()                        {}
