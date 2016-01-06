package bake

import (
	"errors"
)

// ErrUnregisteredFileSystem is returned when a file system has not been registered.
var ErrUnregisteredFileSystem = errors.New("unregistered file system")

// FileSystem represents a way to mount the build filesystem.
type FileSystem interface {
	Open() error
	Close() error

	// The underlying path being served.
	Path() string

	// Creates a new root path for the file system where changes can be tracked.
	CreateRoot() FileSystemRoot
}

// FileSystemRoot represents a copy of the file system root.
// It can be used for tracking reads & writes.
type FileSystemRoot interface {
	Path() string
	Readset() map[string]struct{}
	Writeset() map[string]struct{}
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
	// Underlying path to serve
	Path string

	// Directory to mount to.
	MountPath string
}

// nopFileSystem is a file system that does nothing.
type nopFileSystem struct{}

func (*nopFileSystem) Open() error                { return nil }
func (*nopFileSystem) Close() error               { return nil }
func (*nopFileSystem) Path() string               { return "" }
func (*nopFileSystem) CreateRoot() FileSystemRoot { return &nopFileSystemRoot{} }

// nopFileSystemRoot is a file system root that does nothing.
type nopFileSystemRoot struct{}

func (*nopFileSystemRoot) Path() string                  { return "" }
func (*nopFileSystemRoot) Readset() map[string]struct{}  { return nil }
func (*nopFileSystemRoot) Writeset() map[string]struct{} { return nil }
