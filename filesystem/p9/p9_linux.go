package p9

import (
	"fmt"
	"net"
	"syscall"
)

// Mount mounts fs to the target path.
func (fs *FileSystem) Mount(target string) error {
	addr := fs.Listener().Addr().(*net.TCPAddr)
	println(addr.String(), "/", target, "/", "9p", "/", fmt.Sprintf("trans=tcp,port=%d", addr.Port))
	return syscall.Mount(addr.IP.String(), target, "9p", 0, fmt.Sprintf("trans=tcp,port=%d", addr.Port))
}

// Unmount removes the mount from the target path.
func (fs *FileSystem) Unmount(target string) error {
	return syscall.Unmount(target, 0)
}
