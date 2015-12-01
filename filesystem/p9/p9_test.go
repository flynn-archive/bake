package p9_test

import (
	"bytes"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/flynn/bake/filesystem/p9"
	"github.com/rminnich/go9p"
)

// Ensure that a file can be read and tracked.
func TestFileSystem_Read(t *testing.T) {
	fs := OpenFileSystem()
	defer fs.Close()
	c := MustMountFS(fs)
	defer c.Unmount()

	// Generate fake file.
	fs.MustWriteFile("foo/bar", []byte{0, 1, 2, 3}, 0666)

	// Open and read contents of file through 9p.
	f, err := c.FOpen("/foo/bar", go9p.OREAD)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	buf := make([]byte, 4)
	if n, err := f.Read(buf); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(buf, []byte{0, 1, 2, 3}) {
		t.Fatalf("unexpected bytes: %x (n=%d)", buf, n)
	}

	// Verify readset.
	if rs := fs.ReadsetSlice(); !reflect.DeepEqual(rs, []string{"/foo/bar"}) {
		t.Fatalf("unexpected readset: %#v", rs)
	}
}

// Ensure that a file can be written and tracked.
func TestFileSystem_Write(t *testing.T) {
	fs := OpenFileSystem()
	defer fs.Close()
	c := MustMountFS(fs)
	defer c.Unmount()

	// Create directory through 9p.
	if f, err := c.FCreate("/foo", 0777|go9p.DMDIR, go9p.OREAD); err != nil {
		t.Fatal(err)
	} else if err = f.Close(); err != nil {
		t.Fatal(err)
	}

	// Open file through 9p.
	f, err := c.FCreate("/foo/bar", 0777, go9p.OWRITE)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Write file through 9p.
	if _, err := f.Write([]byte{0, 1, 2, 3}); err != nil {
		t.Fatal(err)
	}

	// Read local file.
	if buf, err := ioutil.ReadFile(filepath.Join(fs.Root, "foo", "bar")); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(buf, []byte{0, 1, 2, 3}) {
		t.Fatalf("unexpected bytes: %x", buf)
	}

	// Verify writeset.
	if ws := fs.WritesetSlice(); !reflect.DeepEqual(ws, []string{"/foo", "/foo/bar"}) {
		t.Fatalf("unexpected writeset: %#v", ws)
	}
}

// Ensure that a file can be deleted and tracked.
func TestFileSystem_Remove(t *testing.T) {
	fs := OpenFileSystem()
	defer fs.Close()
	c := MustMountFS(fs)
	defer c.Unmount()

	// Generate fake file.
	fs.MustWriteFile("foo/bar", []byte{0, 1, 2, 3}, 0666)

	// Remove file through 9p.
	if err := c.FRemove("/foo/bar"); err != nil {
		t.Fatal(err)
	}

	// Verify writeset.
	if ws := fs.WritesetSlice(); !reflect.DeepEqual(ws, []string{"/foo/bar"}) {
		t.Fatalf("unexpected writeset: %#v", ws)
	}
}

// FileSystem represents a test wrapper for p9.FileSystem.
type FileSystem struct {
	*p9.FileSystem
	ln net.Listener
}

// NewFileSystem returns a new instance of FileSystem.
func NewFileSystem() *FileSystem {
	path, err := ioutil.TempDir("", "p9-")
	if err != nil {
		panic(err)
	}

	fs := &FileSystem{FileSystem: p9.NewFileSystem()}
	fs.Root = path
	fs.Addr = ":0"
	return fs
}

// OpenFileSystem returns an open FileSystem on a random port. Panic on error.
func OpenFileSystem() *FileSystem {
	fs := NewFileSystem()
	if err := fs.Open(); err != nil {
		panic(err)
	}
	return fs
}

// Close closes the file system and removes the underlying temp directory.
func (fs *FileSystem) Close() error {
	err := fs.FileSystem.Close()
	os.RemoveAll(fs.Root)
	return err
}

func (fs *FileSystem) MustWriteFile(filename string, data []byte, perm os.FileMode) {
	path := filepath.Join(fs.Root, filename)

	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(path, data, perm); err != nil {
		panic(err)
	}
}

// MustMountFS mounts a client to fs. Panic on error.
func MustMountFS(fs *FileSystem) *go9p.Clnt {
	root := go9p.OsUsers.Uid2User(0)
	clnt, err := go9p.Mount("tcp", fs.Listener().Addr().String(), "/", 8192, root)
	if err != nil {
		panic(err)
	}
	// clnt.Debuglevel = 1
	return clnt
}
