package p9

import "errors"

// Mount mounts fs to the target path.
func (fs *FileSystem) Mount(target string) error { return errors.New("not implemented") }

// Unmount removes the mount from the target path.
func (fs *FileSystem) Unmount(target string) error { return errors.New("not implemented") }
