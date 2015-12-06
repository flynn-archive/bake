package p9

import (
	"fmt"
	"net"
	"syscall"
)

// mount mounts fs to the mount path.
func (fs *FileSystem) mount() error {
	addr := fs.Listener().Addr().(*net.TCPAddr)
	println(addr.String(), "/", fs.MountPath, "/", "9p", "/", fmt.Sprintf("trans=tcp,port=%d", addr.Port))
	return syscall.Mount(addr.IP.String(), fs.MountPath, "9p", 0, fmt.Sprintf("trans=tcp,port=%d", addr.Port))
}

// unmount removes the mount from the mount path.
func (fs *FileSystem) unmount() error {
	return syscall.Unmount(fs.MountPath, 0)
}
