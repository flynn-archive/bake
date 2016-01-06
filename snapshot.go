package bake

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/flynn/bake/internal"
	"github.com/gogo/protobuf/proto"
)

// ErrSnapshotTargetNotFound is returned when operating on a target that doesn't exist.
var ErrSnapshotTargetNotFound = errors.New("snapshot target not found")

// Snapshot represents the state of the build system.
// This includes the targets, their list of file dependencies, and file states.
type Snapshot struct {
	path string // path to snapshot data
	root string // path to project root
}

// NewSnapshot returns a new instance of Snapshot.
func NewSnapshot(path, root string) *Snapshot {
	return &Snapshot{
		path: path,
		root: root,
	}
}

// Path returns the path that the snapshot was initialized with.
func (ss *Snapshot) Path() string { return ss.path }

// Root returns the project root that the snapshot was initialized with.
func (ss *Snapshot) Root() string { return ss.root }

// AddTarget adds a target to the snapshot.
//
// If the target already exists then it is merged with the existing record.
// The file dependencies of target are checked for changes and updated if needed.
func (ss *Snapshot) AddTarget(t *Target, inputs []string) error {
	// Create and stat input files.
	files, err := newFileSnapshots(ss.root, inputs)
	if err != nil {
		return err
	}

	// Add target with current input file state.
	ts := &targetSnapshot{
		name:   t.Name,
		hash:   hashTarget(t),
		inputs: files,
	}

	// Write to file.
	if err := ss.writeTarget(ts); err != nil {
		return err
	}

	return nil
}

// IsTargetDirty returns true if a target has changed or its file inputs have changed.
func (ss *Snapshot) IsTargetDirty(t *Target) (bool, error) {
	// Read the target from file.
	ts, err := ss.readTarget(t.Name)
	if err == ErrSnapshotTargetNotFound {
		return true, nil
	} else if err != nil {
		return false, err
	}

	// Find snapshot target and compare hash values.
	if ts.hash != hashTarget(t) {
		return true, nil
	}

	// Check if any input files or directories have changed.
	dirty, err := fileSnapshots(ts.inputs).isDirty(ss.root)
	if err != nil {
		return false, err
	}

	return dirty, nil
}

// readTarget reads a target snapshot from within the snapshot and unmarshals it.
func (ss *Snapshot) readTarget(name string) (*targetSnapshot, error) {
	// Create target filename relative to snapshot path.
	path := filepath.Join(ss.path, name)

	// Read from file.
	buf, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, ErrSnapshotTargetNotFound
	} else if err != nil {
		return nil, err
	}

	// Unmarshal to snapshot.
	var pb internal.TargetSnapshot
	if err := proto.Unmarshal(buf, &pb); err != nil {
		return nil, err
	}

	return decodeTargetSnapshot(&pb), nil
}

// writeTarget marshals a target snapshot to bytes and writes it within the snapshot path.
func (ss *Snapshot) writeTarget(t *targetSnapshot) error {
	// Marshal to bytes.
	buf, err := proto.Marshal(encodeTargetSnapshot(t))
	if err != nil {
		return err
	}

	// Create target filename relative to snapshot path.
	path := filepath.Join(ss.path, t.name)

	// Make parent directories.
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return err
	}

	// Write to file.
	if err := ioutil.WriteFile(path, buf, 0666); err != nil {
		return err
	}

	return nil
}

// targetSnapshot represents the state of a target.
type targetSnapshot struct {
	name   string
	hash   string
	inputs []*fileSnapshot
}

// encodeTargetSnapshot encodes a snapshot target into a protobuf object.
func encodeTargetSnapshot(t *targetSnapshot) *internal.TargetSnapshot {
	return &internal.TargetSnapshot{
		Name:   proto.String(t.name),
		Hash:   proto.String(t.hash),
		Inputs: encodeFileSnapshots(t.inputs),
	}
}

// decodeTargetSnapshot decodes a snapshot target from a protobuf object.
func decodeTargetSnapshot(pb *internal.TargetSnapshot) *targetSnapshot {
	return &targetSnapshot{
		name:   pb.GetName(),
		hash:   pb.GetHash(),
		inputs: decodeFileSnapshots(pb.GetInputs()),
	}
}

// fileSnapshot represents the state of a file dependency for a target.
type fileSnapshot struct {
	name    string
	hash    string
	content string
}

// newFileSnapshot returns a new instance of fileSnapshot for a filename.
func newFileSnapshot(path, name string) (*fileSnapshot, error) {
	hash, err := hashFileInfo(filepath.Join(path, name))
	if err != nil {
		return nil, err
	}
	content, err := hashFileContent(filepath.Join(path, name))
	if err != nil {
		return nil, err
	}
	return &fileSnapshot{name: name, hash: hash, content: content}, nil
}

// isDirty returns true if the hash of the file has changed or the file was deleted.
func (f *fileSnapshot) isDirty(path string) (bool, error) {
	// Check for differences in file info first.
	if h, err := hashFileInfo(filepath.Join(path, f.name)); os.IsNotExist(err) {
		return true, nil
	} else if err != nil {
		return false, err
	} else if f.hash != h {
		return true, nil
	}

	// If info is the same then compare a hash of the contents.
	if h, err := hashFileContent(filepath.Join(path, f.name)); err != nil {
		return false, err
	} else if f.content != h {
		return true, nil
	}

	return false, nil
}

// encodeFileSnapshot encodes a snapshot file into a protobuf object.
func encodeFileSnapshot(f *fileSnapshot) *internal.FileSnapshot {
	return &internal.FileSnapshot{
		Name:    proto.String(f.name),
		Hash:    proto.String(f.hash),
		Content: proto.String(f.content),
	}
}

// decodeFileSnapshot decodes a snapshot file from a protobuf object.
func decodeFileSnapshot(pb *internal.FileSnapshot) *fileSnapshot {
	return &fileSnapshot{
		name:    pb.GetName(),
		hash:    pb.GetHash(),
		content: pb.GetContent(),
	}
}

// fileSnapshots represents a list of snapshot files.
type fileSnapshots []*fileSnapshot

// newFileSnapshots returns a slice of stat'd snapshot files.
func newFileSnapshots(path string, names []string) ([]*fileSnapshot, error) {
	// Sort filenames for consistency.
	sort.Strings(names)

	// Build list of snapshot files with current stats.
	// Ignore files that have been deleted. They are likely temporary files.
	a := make([]*fileSnapshot, 0, len(names))
	for _, name := range names {
		f, err := newFileSnapshot(path, name)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		a = append(a, f)
	}
	return a, nil
}

// isDirty returns true after finding the first dirty file.
func (a fileSnapshots) isDirty(path string) (bool, error) {
	for _, f := range a {
		v, err := f.isDirty(path)
		if err != nil {
			return false, err
		} else if v {
			return true, nil
		}
	}
	return false, nil
}

// encodeFileSnapshots encodes a slice of snapshot files into a protobuf object.
func encodeFileSnapshots(a []*fileSnapshot) []*internal.FileSnapshot {
	pb := make([]*internal.FileSnapshot, len(a))
	for i := range a {
		pb[i] = encodeFileSnapshot(a[i])
	}
	return pb
}

// decodeFileSnapshots decodes a slice of snapshot files from a protobuf object.
func decodeFileSnapshots(pb []*internal.FileSnapshot) []*fileSnapshot {
	a := make([]*fileSnapshot, len(pb))
	for i := range pb {
		a[i] = decodeFileSnapshot(pb[i])
	}
	return a
}

// hashTarget returns a hash for a target based on its commands and dependencies.
func hashTarget(t *Target) string {
	h := sha256.New()
	writeStrings(h, t.Dependencies)

	for _, c := range t.Commands {
		switch c := c.(type) {
		case *ExecCommand:
			h.Write([]byte("exec"))
			writeStrings(h, c.Args)
		case *ShellCommand:
			h.Write([]byte("shell"))
			h.Write([]byte(c.Source))
		default:
			panic("unreachable")
		}
	}

	return fmt.Sprintf("%64x", h.Sum(nil))
}

// hashFileInfo generates a hash for a file or directory info.
// Regular files are hashed using their mode, mtime & size.
// Directories are hashed using its mode and a list of the directory's files.
func hashFileInfo(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	// If it's a regular file then hash the mode, mtime, & size.
	if !fi.IsDir() {
		h := sha256.New()
		h.Write(u32tob(uint32(fi.Mode())))
		h.Write(u64tob(uint64(fi.ModTime().UnixNano())))
		h.Write(u64tob(uint64(fi.Size())))
		return fmt.Sprintf("%64x", h.Sum(nil)), nil
	}

	// Otherwise hash the files within the directory.
	return hashDirInfo(path)
}

func hashDirInfo(path string) (string, error) {
	// Open directory for reading.
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Retrieve file info.
	fi, err := f.Stat()
	if err != nil {
		return "", err
	}

	// Read and sort all filenames from within the directory.
	names, err := f.Readdirnames(0)
	if err != nil {
		return "", err
	}
	sort.Strings(names)

	// Generate a hash a null-delimited list of names.
	h := sha256.New()
	h.Write(u32tob(uint32(fi.Mode())))
	writeStrings(h, names)
	return fmt.Sprintf("%64x", h.Sum(nil)), nil
}

// hashContent generates a hash for a file's contents.
// Directories always return a hash of zero.
func hashFileContent(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return "", err
	}

	// If it's a regular file then hash the mtime.
	if fi.IsDir() {
		return "", nil
	}

	// Generate hash from file contents.
	h := sha256.New()
	if _, err := io.CopyN(h, f, fi.Size()); err != nil {
		return "", err
	}
	return fmt.Sprintf("%64x", h.Sum(nil)), nil
}

// writeStrings writes a null terminated strings to h.
func writeStrings(w io.Writer, a []string) {
	for _, s := range a {
		w.Write([]byte(s))
		w.Write([]byte{0})
	}
}

func u32tob(v uint32) []byte { b := make([]byte, 4); binary.BigEndian.PutUint32(b, v); return b }
func u64tob(v uint64) []byte { b := make([]byte, 8); binary.BigEndian.PutUint64(b, v); return b }
