package p9

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/user"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/flynn/bake"
	"github.com/rminnich/go9p"
)

// Type represents the type name for this file system.
const Type = "9p"

// DefaultAddr is the default address to listen on for the file system.
const DefaultAddr = "127.0.0.1:0"

func init() {
	bake.RegisterFileSystem(Type, func(opt bake.FileSystemOptions) (bake.FileSystem, error) {
		fs := NewFileSystem(opt.Path)
		fs.MountPath = opt.MountPath
		return fs, nil
	})
}

// FileSystem represents a 9p file system.
type FileSystem struct {
	mu  sync.Mutex
	srv go9p.Srv
	ln  net.Listener

	path string // Directory to serve

	// Copies of the root path.
	// These are separated to allow concurrent tracking of file system access.
	roots      map[string]*FileSystemRoot
	nextRootID int

	closing chan struct{}
	wg      sync.WaitGroup

	Addr string

	// Directory to mount to.
	MountPath string
}

// NewFileSystem returns a new instance of FileSystem.
func NewFileSystem(path string) *FileSystem {
	return &FileSystem{
		srv: go9p.Srv{
			Id:         "bakefs",
			Dotu:       true,
			Debuglevel: 0,
		},

		path:    path,
		roots:   make(map[string]*FileSystemRoot),
		closing: make(chan struct{}),

		Addr: DefaultAddr,
	}
}

func (fs *FileSystem) Open() error {
	// Listen to bind address.
	ln, err := net.Listen("tcp", fs.Addr)
	if err != nil {
		return err
	}
	fs.ln = ln

	// Begin serving connections on the listener.
	fs.wg.Add(1)
	go func() {
		defer fs.wg.Done()
		if err := fs.serve(); err != nil && isTemporary(err) {
			log.Println("serve error: %s", err)
		}
	}()

	// Attach filesystem to 9p server.
	// This only panics if fs doesn't implement go9p.SrvReqOps.
	if !fs.srv.Start((*fileSystem)(fs)) {
		panic("could not start file system")
	}

	// Mount to mount path.
	if fs.MountPath != "" {
		if err := fs.mount(); err != nil {
			return err
		}
	}

	return nil
}

// Close closes the file system. Returns after listener has returned.
func (fs *FileSystem) Close() error {
	if fs.MountPath != "" {
		fs.unmount()
	}

	if fs.ln != nil {
		fs.ln.Close()
	}

	// Notify goroutines of closing and wait.
	close(fs.closing)
	fs.wg.Wait()

	return nil
}

// Path returns the path being served.
func (fs *FileSystem) Path() string { return fs.path }

// Listener returns the underlying listener. Available after Open().
func (fs *FileSystem) Listener() net.Listener { return fs.ln }

// serve accepts and handles connections from the listener.
func (fs *FileSystem) serve() error {
	for {
		c, err := fs.ln.Accept()
		if err != nil {
			return err
		}
		fs.srv.NewConn(c)
	}
}

// CreateRoot returns a new copy of the root path of the file system.
func (fs *FileSystem) CreateRoot() bake.FileSystemRoot {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Generate string id to use for root.
	id := fmt.Sprintf("%04x", fs.nextRootID)
	fs.nextRootID++

	// Create root and add it to map.
	root := NewFileSystemRoot(id, path.Join(fs.MountPath, id))
	fs.roots[id] = root

	return root
}

// Root returns a root by id.
func (fs *FileSystem) Root(id string) *FileSystemRoot {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.roots[id]
}

// Ensure fileSystem implements go9p.SrvReqOps.
var _ go9p.SrvReqOps = (*fileSystem)(nil)

// fileSystem is a wrapper type for FileSystem that implements go9p callbacks.
type fileSystem FileSystem

// addToReadset adds s to the readset.
func (fs *fileSystem) addToReadset(rootID, filename string) {
	root := (*FileSystem)(fs).Root(rootID)
	if root == nil {
		return
	}
	root.AddToReadset(strings.TrimPrefix(filename, fs.path))
}

// addToWriteset adds s to the writeset.
func (fs *fileSystem) addToWriteset(rootID, filename string) {
	root := (*FileSystem)(fs).Root(rootID)
	if root == nil {
		return
	}
	root.AddToWriteset(strings.TrimPrefix(filename, fs.path))
}

// split splits s into the root name and remaining filepath.
func split(s string) (root, filename string) {
	a := strings.SplitN(s, "/", 3)
	if len(a) < 3 {
		return "", filename
	}
	return a[1], "/" + a[2]
}

// Attach creates a file handle on the file system.
func (fs *fileSystem) Attach(req *go9p.SrvReq) {
	rootID, filename := split(req.Tc.Aname)
	aux := &Aux{
		rootID: rootID,
		path:   path.Join(fs.path, filename),
	}
	req.Fid.Aux = aux

	if err := aux.stat(); err != nil {
		req.RespondError(err)
		return
	}

	req.RespondRattach(aux.qid())
}

func (fs *fileSystem) Flush(req *go9p.SrvReq) {}

func (fs *fileSystem) Walk(req *go9p.SrvReq) {
	// Stat the file.
	aux := req.Fid.Aux.(*Aux)
	if err := aux.stat(); err != nil {
		req.RespondError(err)
		return
	}

	// Create a new file handle, if necessary.
	if req.Newfid.Aux == nil {
		req.Newfid.Aux = &Aux{rootID: aux.rootID}
	}

	nfid := req.Newfid.Aux.(*Aux)
	wqids := make([]go9p.Qid, 0, len(req.Tc.Wname))
	newPath := aux.path

	// Append actual files after the root.
	for i, name := range req.Tc.Wname {
		// If we have no root then the first segment is the root ID.
		if nfid.rootID == "" {
			nfid.rootID = name
			if root := (*FileSystem)(fs).Root(nfid.rootID); root == nil {
				req.RespondError(go9p.Enoent)
				return
			}
			wqids = append(wqids, *newRootQid(nfid.rootID))
			continue
		}

		// Otherwise we're already walking a root so continue to traverse the files.
		p := newPath + "/" + name
		st, err := os.Lstat(p)
		if err != nil {
			if i == 0 {
				req.RespondError(go9p.Enoent)
				return
			}
			break
		}

		wqids = append(wqids, *newQid(st))
		newPath = p
	}

	nfid.path = newPath
	req.RespondRwalk(wqids)
}

// Open opens a local file.
func (fs *fileSystem) Open(req *go9p.SrvReq) {
	// Stat file handle.
	aux := req.Fid.Aux.(*Aux)
	if err := aux.stat(); err != nil {
		req.RespondError(err)
		return
	}

	// Open local file handle.
	file, err := os.OpenFile(aux.path, omode2uflags(req.Tc.Mode), 0)
	if err != nil {
		req.RespondError(toError(err))
		return
	}
	aux.file = file

	// Add to appropriate set.
	if req.Tc.Mode&go9p.OREAD != 0 {
		fs.addToReadset(aux.rootID, aux.path)
	} else if req.Tc.Mode&go9p.OWRITE != 0 {
		fs.addToWriteset(aux.rootID, aux.path)
	} else if req.Tc.Mode&go9p.ORDWR != 0 {
		fs.addToReadset(aux.rootID, aux.path)
		fs.addToWriteset(aux.rootID, aux.path)
	}

	req.RespondRopen(aux.qid(), 0)
}

// Create creates a new file.
func (fs *fileSystem) Create(req *go9p.SrvReq) {
	aux := req.Fid.Aux.(*Aux)
	if err := aux.stat(); err != nil {
		req.RespondError(err)
		return
	}

	path := aux.path + "/" + req.Tc.Name

	var file *os.File
	var err error
	switch {
	case req.Tc.Perm&go9p.DMDIR != 0:
		err = os.Mkdir(path, os.FileMode(req.Tc.Perm&0777))

	case req.Tc.Perm&go9p.DMSYMLINK != 0:
		err = os.Symlink(req.Tc.Ext, path)

	case req.Tc.Perm&go9p.DMLINK != 0:
		n, err := strconv.ParseUint(req.Tc.Ext, 10, 0)
		if err != nil {
			break
		}

		ofid := req.Conn.FidGet(uint32(n))
		if ofid == nil {
			req.RespondError(go9p.Eunknownfid)
			return
		}

		err = os.Link(ofid.Aux.(*Aux).path, path)
		ofid.DecRef()

	case req.Tc.Perm&go9p.DMNAMEDPIPE != 0:
	case req.Tc.Perm&go9p.DMDEVICE != 0:
		req.RespondError(&go9p.Error{"not implemented", go9p.EIO})
		return

	default:
		var mode uint32 = req.Tc.Perm & 0777
		if req.Conn.Dotu {
			if req.Tc.Perm&go9p.DMSETUID > 0 {
				mode |= syscall.S_ISUID
			}
			if req.Tc.Perm&go9p.DMSETGID > 0 {
				mode |= syscall.S_ISGID
			}
		}
		file, err = os.OpenFile(path, omode2uflags(req.Tc.Mode)|os.O_CREATE, os.FileMode(mode))
	}

	if file == nil && err == nil {
		file, err = os.OpenFile(path, omode2uflags(req.Tc.Mode), 0)
	}

	if err != nil {
		req.RespondError(toError(err))
		return
	}

	aux.path = path
	aux.file = file

	// Save file to writeset.
	fs.addToWriteset(aux.rootID, path)

	if err := aux.stat(); err != nil {
		req.RespondError(err)
		return
	}

	req.RespondRcreate(aux.qid(), 0)
}

// Read reads data from a file handle.
func (fs *fileSystem) Read(req *go9p.SrvReq) {
	aux := req.Fid.Aux.(*Aux)
	if err := aux.stat(); err != nil {
		req.RespondError(err)
		return
	}

	// Add to readset.
	fs.addToReadset(aux.rootID, aux.path)

	go9p.InitRread(req.Rc, req.Tc.Count)
	if aux.st.IsDir() {
		fs.readDir(req)
		return
	}
	fs.readFile(req)
}

func (fs *fileSystem) readDir(req *go9p.SrvReq) {
	aux := req.Fid.Aux.(*Aux)

	var n int
	if req.Tc.Offset == 0 {
		// If we got here, it was open. Can't really seek
		// in most cases, just close and reopen it.
		aux.file.Close()

		file, err := os.OpenFile(aux.path, omode2uflags(req.Fid.Omode), 0)
		if err != nil {
			req.RespondError(toError(err))
			return
		}
		aux.file = file

		dirs, e := aux.file.Readdir(-1)
		if e != nil {
			req.RespondError(toError(e))
			return
		}
		aux.dirs = dirs

		aux.dirents = nil
		aux.direntends = nil
		for _, dir := range aux.dirs {
			path := aux.path + "/" + dir.Name()
			st, _ := new9pDir(path, dir, req.Conn.Dotu, req.Conn.Srv.Upool)
			if st == nil {
				continue
			}

			b := go9p.PackDir(st, req.Conn.Dotu)
			aux.dirents = append(aux.dirents, b...)
			n += len(b)
			aux.direntends = append(aux.direntends, n)
		}
	}

	switch {
	case req.Tc.Offset > uint64(len(aux.dirents)):
		n = 0
	case len(aux.dirents[req.Tc.Offset:]) > int(req.Tc.Count):
		n = int(req.Tc.Count)
	default:
		n = len(aux.dirents[req.Tc.Offset:])
	}

	copy(req.Rc.Data, aux.dirents[req.Tc.Offset:int(req.Tc.Offset)+n])

	go9p.SetRreadCount(req.Rc, uint32(n))
	req.Respond()
}

func (fs *fileSystem) readFile(req *go9p.SrvReq) {
	aux := req.Fid.Aux.(*Aux)

	n, err := aux.file.ReadAt(req.Rc.Data, int64(req.Tc.Offset))
	if err != nil && err != io.EOF {
		req.RespondError(toError(err))
		return
	}

	go9p.SetRreadCount(req.Rc, uint32(n))
	req.Respond()
}

// Write writes data to a file.
func (fs *fileSystem) Write(req *go9p.SrvReq) {
	aux := req.Fid.Aux.(*Aux)
	if err := aux.stat(); err != nil {
		req.RespondError(err)
		return
	}

	// Add to writeset.
	fs.addToWriteset(aux.rootID, aux.path)

	n, err := aux.file.WriteAt(req.Tc.Data, int64(req.Tc.Offset))
	if err != nil {
		req.RespondError(toError(err))
		return
	}

	req.RespondRwrite(uint32(n))
}

func (fs *fileSystem) Clunk(req *go9p.SrvReq) { req.RespondRclunk() }

func (fs *fileSystem) Remove(req *go9p.SrvReq) {
	aux := req.Fid.Aux.(*Aux)
	if err := aux.stat(); err != nil {
		req.RespondError(err)
		return
	}

	// Add to writeset.
	fs.addToWriteset(aux.rootID, aux.path)

	if err := os.Remove(aux.path); err != nil {
		req.RespondError(toError(err))
		return
	}

	req.RespondRremove()
}

func (fs *fileSystem) Stat(req *go9p.SrvReq) {
	aux := req.Fid.Aux.(*Aux)
	if err := aux.stat(); err != nil {
		req.RespondError(err)
		return
	}

	st, err := new9pDir(aux.path, aux.st, req.Conn.Dotu, req.Conn.Srv.Upool)
	if st == nil {
		req.RespondError(err)
		return
	}

	req.RespondRstat(st)
}

// Wstat updates file info.
func (fs *fileSystem) Wstat(req *go9p.SrvReq) {
	aux := req.Fid.Aux.(*Aux)
	if err := aux.stat(); err != nil {
		req.RespondError(err)
		return
	}

	// Add to writeset.
	fs.addToWriteset(aux.rootID, aux.path)

	dir := &req.Tc.Dir
	if dir.Mode != 0xFFFFFFFF {
		mode := dir.Mode & 0777
		if req.Conn.Dotu {
			if dir.Mode&go9p.DMSETUID > 0 {
				mode |= syscall.S_ISUID
			}
			if dir.Mode&go9p.DMSETGID > 0 {
				mode |= syscall.S_ISGID
			}
		}

		err := os.Chmod(aux.path, os.FileMode(mode))
		if err != nil {
			req.RespondError(toError(err))
			return
		}
	}

	uid, gid := go9p.NOUID, go9p.NOUID
	if req.Conn.Dotu {
		uid = dir.Uidnum
		gid = dir.Gidnum
	}

	// Try to find local uid, gid by name.
	if (dir.Uid != "" || dir.Gid != "") && !req.Conn.Dotu {
		var err error
		uid, err = lookupUser(dir.Uid)
		if err != nil {
			req.RespondError(err)
			return
		}

		gid, err = lookupGroup(dir.Gid)
		if err != nil {
			req.RespondError(err)
			return
		}
	}

	if uid != go9p.NOUID || gid != go9p.NOUID {
		err := os.Chown(aux.path, int(uid), int(gid))
		if err != nil {
			req.RespondError(toError(err))
			return
		}
	}

	if dir.Name != "" {
		// if first char is / it is relative to root, else relative to cwd.
		var destpath string
		if dir.Name[0] == '/' {
			destpath = path.Join(fs.path, dir.Name)
			fmt.Printf("/ results in %s\n", destpath)
		} else {
			auxdir, _ := path.Split(aux.path)
			destpath = path.Join(auxdir, dir.Name)
			fmt.Printf("rel  results in %s\n", destpath)
		}
		err := syscall.Rename(aux.path, destpath)
		fmt.Printf("rename %s to %s gets %v\n", aux.path, destpath, err)
		if err != nil {
			req.RespondError(toError(err))
			return
		}
		aux.path = destpath
	}

	// Set file size, if specified.
	if dir.Length != 0xFFFFFFFFFFFFFFFF {
		if err := os.Truncate(aux.path, int64(dir.Length)); err != nil {
			req.RespondError(toError(err))
			return
		}
	}

	// If either mtime or atime need to be changed, then we must change both.
	if dir.Mtime != ^uint32(0) || dir.Atime != ^uint32(0) {
		mtime := time.Unix(int64(dir.Mtime), 0)
		atime := time.Unix(int64(dir.Atime), 0)

		mtimeChanged := (dir.Mtime == ^uint32(0))
		atimeChanged := (dir.Atime == ^uint32(0))
		if mtimeChanged || atimeChanged {
			st, err := os.Stat(aux.path)
			if err != nil {
				req.RespondError(toError(err))
				return
			} else if mtimeChanged {
				mtime = st.ModTime()
			}
		}
		if err := os.Chtimes(aux.path, atime, mtime); err != nil {
			req.RespondError(toError(err))
			return
		}
	}

	req.RespondRwstat()
}

func (fs *fileSystem) FidDestroy(fid *go9p.SrvFid) {
	if fid.Aux == nil {
		return
	} else if aux := fid.Aux.(*Aux); aux.file != nil {
		aux.file.Close()
	}
}

// func (fs *fileSystem) ConnOpened(conn *go9p.Conn) { println("conn open") }
// func (fs *fileSystem) ConnClosed(conn *go9p.Conn) { println("conn closed") }

// FileSystemRoot represents root path of the file system.
// This is used for tracking read and write access.
type FileSystemRoot struct {
	mu   sync.Mutex
	id   string
	path string

	readset  map[string]struct{}
	writeset map[string]struct{}
}

// NewFileSystemRoot returns a new filesystem root identified by id.
func NewFileSystemRoot(id, path string) *FileSystemRoot {
	return &FileSystemRoot{
		id:       id,
		path:     path,
		readset:  make(map[string]struct{}),
		writeset: make(map[string]struct{}),
	}
}

// ID returns the identifier for this root.
func (r *FileSystemRoot) ID() string { return r.id }

// Path returns the path this root is served from.
func (r *FileSystemRoot) Path() string { return r.path }

// Readset returns a set of files that have been read from the file system.
func (r *FileSystemRoot) Readset() map[string]struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	return copySet(r.readset)
}

// ReadsetSlice returns a slice of files that have been read from the file system.
func (r *FileSystemRoot) ReadsetSlice() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	a := make([]string, 0, len(r.readset))
	for k := range r.readset {
		a = append(a, k)
	}
	sort.Strings(a)
	return a
}

// AddToReadset adds s to the root's readset.
func (r *FileSystemRoot) AddToReadset(s string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.readset[s] = struct{}{}
}

// Writeset returns a set of files that have been written to the file system.
func (r *FileSystemRoot) Writeset() map[string]struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	return copySet(r.writeset)
}

// WritesetSlice returns a slice of files that have been written from the file system.
func (r *FileSystemRoot) WritesetSlice() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	a := make([]string, 0, len(r.writeset))
	for k := range r.writeset {
		a = append(a, k)
	}
	sort.Strings(a)
	return a
}

// AddToWriteset adds s to the root's writeset.
func (r *FileSystemRoot) AddToWriteset(s string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writeset[s] = struct{}{}
}

// Aux represents auxillary data for 9p file handles.
type Aux struct {
	rootID     string
	path       string
	file       *os.File
	dirs       []os.FileInfo
	direntends []int
	dirents    []byte
	diroffset  uint64
	st         os.FileInfo
}

// qid returns an qid identifier for aux.
func (aux Aux) qid() *go9p.Qid { return newQid(aux.st) }

func (aux *Aux) stat() *go9p.Error {
	st, err := os.Lstat(aux.path)
	if err != nil {
		return toError(err)
	}
	aux.st = st

	return nil
}

func new9pDir(path string, fi os.FileInfo, dotu bool, upool go9p.Users) (*go9p.Dir, error) {
	// Retrieve underlying stat_t implementation.
	stat, _ := fi.Sys().(*syscall.Stat_t)
	if stat == nil {
		return nil, &os.PathError{"stat_t unavailable", path, nil}
	}

	// Construct 9p directory.
	var dir go9p.Dir
	dir.Qid = *newQid(fi)
	dir.Mode = mode(fi.Mode(), dotu)
	dir.Mtime = uint32(fi.ModTime().Unix())
	dir.Length = uint64(fi.Size())
	dir.Name = path[strings.LastIndex(path, "/")+1:]

	// Use simple user/group lookups for non-dotu calls.
	if !dotu {
		dir.Uid = strconv.Itoa(int(stat.Uid))
		if u, err := user.LookupId(dir.Uid); err == nil {
			dir.Uid = u.Username
		}

		dir.Gid = strconv.Itoa(int(stat.Gid))
		if g, err := user.LookupId(dir.Gid); err == nil {
			dir.Gid = g.Username
		}
		return &dir, nil
	}

	// Lookup user from pool.
	u := upool.Uid2User(int(stat.Uid))
	if dir.Uid = u.Name(); dir.Uid == "" {
		dir.Uid = "none"
	}
	dir.Uidnum = uint32(u.Id())

	// Lookup group from pool.
	g := upool.Gid2Group(int(stat.Gid))
	if dir.Gid = g.Name(); dir.Gid == "" {
		dir.Gid = "none"
	}
	dir.Gidnum = uint32(g.Id())

	dir.Muid = "none"
	dir.Muidnum = go9p.NOUID

	// Determine extension by type.
	dir.Ext = ""
	if fi.Mode()&os.ModeSymlink != 0 {
		ext, err := os.Readlink(path)
		if err != nil {
			dir.Ext = ""
		}
		dir.Ext = ext
	} else if isBlock(fi) {
		dir.Ext = fmt.Sprintf("b %d %d", stat.Rdev>>24, stat.Rdev&0xFFFFFF)
	} else if isChar(fi) {
		dir.Ext = fmt.Sprintf("c %d %d", stat.Rdev>>24, stat.Rdev&0xFFFFFF)
	}

	return &dir, nil
}

func isBlock(d os.FileInfo) bool {
	stat := d.Sys().(*syscall.Stat_t)
	return (stat.Mode & syscall.S_IFMT) == syscall.S_IFBLK
}

func isChar(d os.FileInfo) bool {
	stat := d.Sys().(*syscall.Stat_t)
	return (stat.Mode & syscall.S_IFMT) == syscall.S_IFCHR
}

func omode2uflags(mode uint8) int {
	var ret int
	switch mode & 3 {
	case go9p.OREAD:
		ret = os.O_RDONLY
	case go9p.ORDWR:
		ret = os.O_RDWR
	case go9p.OWRITE:
		ret = os.O_WRONLY
	case go9p.OEXEC:
		ret = os.O_RDONLY
	}

	if mode&go9p.OTRUNC != 0 {
		ret |= os.O_TRUNC
	}

	return ret
}

// newQid creates a new qid from a file info object.
func newQid(fi os.FileInfo) *go9p.Qid {
	// Create qid type.
	typ := uint8(0)
	if fi.IsDir() {
		typ |= go9p.QTDIR
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		typ |= go9p.QTSYMLINK
	}

	return &go9p.Qid{
		Path:    fi.Sys().(*syscall.Stat_t).Ino,
		Version: uint32(fi.ModTime().UnixNano() / 1000000),
		Type:    typ,
	}
}

// newRootQid creates a new qid from a root id.
func newRootQid(id string) *go9p.Qid {
	// Parse identifier into number.
	num, err := strconv.ParseUint(id, 16, 64)
	if err != nil {
		panic(err)
	}

	return &go9p.Qid{
		Path:    rootQidFlag & num,
		Version: 1,
		Type:    go9p.QTDIR,
	}
}

// rootQidFlag is a flag on a qid ids for root paths.
const rootQidFlag = uint64(1 << 63)

// mode converts an os.FileMode to a go9p mode.
func mode(m os.FileMode, dotu bool) uint32 {
	other := uint32(m & 0777)
	if m.IsDir() {
		other |= go9p.DMDIR
	}

	if !dotu {
		return other
	}

	if m&os.ModeSymlink != 0 {
		other |= go9p.DMSYMLINK
	}
	if m&os.ModeSocket != 0 {
		other |= go9p.DMSOCKET
	}
	if m&os.ModeNamedPipe != 0 {
		other |= go9p.DMNAMEDPIPE
	}
	if m&os.ModeDevice != 0 {
		other |= go9p.DMDEVICE
	}
	if m&os.ModeSetuid != 0 {
		other |= go9p.DMSETUID
	}
	if m&os.ModeSetgid != 0 {
		other |= go9p.DMSETGID
	}

	return other
}

func lookupUser(uid string) (uint32, *go9p.Error) {
	if uid == "" {
		return go9p.NOUID, nil
	}
	usr, err := user.Lookup(uid)
	if err != nil {
		return go9p.NOUID, toError(err)
	}

	u, err := strconv.Atoi(usr.Uid)
	if err != nil {
		return go9p.NOUID, toError(err)
	}

	return uint32(u), nil
}

func lookupGroup(uid string) (uint32, *go9p.Error) {
	if uid == "" {
		return go9p.NOUID, nil
	}
	usr, err := user.Lookup(uid)
	if err != nil {
		return go9p.NOUID, toError(err)
	}

	u, err := strconv.Atoi(usr.Gid)
	if err != nil {
		return go9p.NOUID, toError(err)
	}

	return uint32(u), nil
}

func toError(err error) *go9p.Error {
	ecode := uint32(go9p.EIO)
	if e, ok := err.(syscall.Errno); ok {
		ecode = uint32(e)
	}
	return &go9p.Error{Err: err.Error(), Errornum: ecode}
}

// isTemporary returns true if the error has an IsTemporary function that returns true.
func isTemporary(err error) bool {
	if err, ok := err.(interface {
		IsTemporary() bool
	}); ok && err.IsTemporary() {
		return true
	}
	return false
}

// copySet returns a copy of m.
func copySet(m map[string]struct{}) map[string]struct{} {
	other := make(map[string]struct{}, len(m))
	for k, v := range m {
		other[k] = v
	}
	return other
}
