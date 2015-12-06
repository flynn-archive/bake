package p9

import "errors"

// Mount mounts fs to the target path.
func (fs *FileSystem) mount() error { return errors.New("not implemented") }

// Unmount removes the mount from the target path.
func (fs *FileSystem) unmount() error { return errors.New("not implemented") }
