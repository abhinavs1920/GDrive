package fs

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	p "path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/winfsp/cgofuse/fuse"
	gdrive "GDrive/internal/drive"
	googleDrive "google.golang.org/api/drive/v3"
	"sync"
)

// GDriveFS struct represents our virtual filesystem
type GDriveFS struct {
	fuse.FileSystemBase
	Drive *gdrive.DriveService
	quotaTotal uint64
	quotaUsed  uint64
	lastQuota  time.Time
	mu         sync.RWMutex
	index      map[string]*googleDrive.File
	fileCache  map[string][]byte
	handles    map[uint64]*os.File
	tempNames  map[uint64]string
	handleCtr  uint64
}

// Read handles file reading (read-only)
func (fs *GDriveFS) Read(path string, buff []byte, offset int64, fh uint64) int {
    cleaned := strings.TrimPrefix(path, "/")
    fs.mu.RLock()
    data, ok := fs.fileCache[cleaned]
    fs.mu.RUnlock()
    if !ok {
        fs.mu.RLock()
        file, ok2 := fs.index[cleaned]
        fs.mu.RUnlock()
        if !ok2 {
            return -fuse.ENOENT
        }

        content, err := fs.Drive.DownloadFile(file)
        if err != nil {
            log.Printf("Download error for %s: %v", cleaned, err)
            return -fuse.EIO
        }
        data = content
        fs.mu.Lock()
        fs.fileCache[cleaned] = data
        fs.mu.Unlock()
    }
    if offset >= int64(len(data)) {
        return 0
    }
    n := copy(buff, data[offset:])
    return n
}

// Write writes to a temp file mapped to the handle
// Write writes to a temp file mapped to the handle
func (fs *GDriveFS) Write(path string, buff []byte, offset int64, fh uint64) int {
    fs.mu.RLock()
    f, ok := fs.handles[fh]
    fs.mu.RUnlock()
    if !ok {
        return -fuse.EBADF
    }
    n, err := f.WriteAt(buff, offset)
    if err != nil {
        log.Printf("write error: %v", err)
        return -fuse.EIO
    }
    return n
}

// Getattr gets file or directory attributes
func (fs *GDriveFS) Getattr(path string, stat *fuse.Stat_t, fh uint64) int {
    if path == "/" || path == "" {
        stat.Mode = fuse.S_IFDIR | 0755
        stat.Nlink = 2
        return 0
    }
    cleaned := strings.TrimPrefix(path, "/")
    fs.mu.RLock()
    file, ok := fs.index[cleaned]
    fs.mu.RUnlock()
    if !ok {
        return -fuse.ENOENT
    }
    if file.MimeType == "application/vnd.google-apps.folder" {
        stat.Mode = fuse.S_IFDIR | 0755
        stat.Nlink = 2
    } else {
        stat.Mode = fuse.S_IFREG | 0644
        stat.Size = file.Size
        stat.Nlink = 1
    }
    return 0
}

// Readdir lists entries in a directory (currently root only)
func (fs *GDriveFS) Readdir(path string, fill func(name string, stat *fuse.Stat_t, ofst int64) bool, offset int64, fh uint64) int {
    cleaned := strings.TrimPrefix(path, "/")
    prefix := ""
    if cleaned != "" {
        // verify dir exists
        fs.mu.RLock()
        if _, ok := fs.index[cleaned]; !ok {
            fs.mu.RUnlock()
            return -fuse.ENOENT
        }
        fs.mu.RUnlock()
        prefix = cleaned + "/"
    }
    fill(".", nil, 0)
    fill("..", nil, 0)
    fs.mu.RLock()
    for p := range fs.index {
        if !strings.HasPrefix(p, prefix) {
            continue
        }
        rest := strings.TrimPrefix(p, prefix)
        if rest == "" || strings.Contains(rest, "/") {
            continue
        }
        fill(rest, nil, 0)
    }
    fs.mu.RUnlock()
    return 0
}

// Statfs provides filesystem statistics using actual quota if available
func (fs *GDriveFS) Statfs(path string, stat *fuse.Statfs_t) int {
    const blockSize = 4096
    stat.Bsize = blockSize
    stat.Frsize = blockSize

    fs.mu.RLock()
    expired := time.Since(fs.lastQuota) > 5*time.Minute
    total := fs.quotaTotal
    used := fs.quotaUsed
    fs.mu.RUnlock()

    if expired && fs.Drive != nil {
        fs.refreshQuota()
        fs.mu.RLock()
        total = fs.quotaTotal
        used = fs.quotaUsed
        fs.mu.RUnlock()
    }

    // if API failed, default 1TiB
    if total == 0 {
        total = 1 << 40
    }
    if used > total {
        used = 0
    }
    freeBytes := total - used

    stat.Blocks = total / blockSize
    stat.Bfree = freeBytes / blockSize
    stat.Bavail = stat.Bfree

    stat.Files = 1 << 20
    stat.Ffree = 1 << 20
    stat.Favail = 1 << 20
    return 0
}

func (fs *GDriveFS) Create(path string, flags int, mode uint32) (int, uint64) {
    cleaned := strings.TrimPrefix(path, "/")
    tmpFile, err := os.CreateTemp("", "gdfs-*")
    if err != nil {
        log.Printf("temp file create error: %v", err)
        return -fuse.EIO, 0
    }
    fs.mu.Lock()
    fs.handleCtr++
    fh := fs.handleCtr
    fs.handles[fh] = tmpFile
    fs.tempNames[fh] = cleaned
    fs.mu.Unlock()
    return 0, fh
}

func (fs *GDriveFS) Open(path string, flags int) (int, uint64) {
    if path == "/" {
        return 0, 0
    }
    cleaned := strings.TrimPrefix(path, "/")
    fs.mu.RLock()
    _, ok := fs.index[cleaned]
    fs.mu.RUnlock()
    if !ok {
        return -fuse.ENOENT, 0
    }
    return 0, 0
}

// Release is called when file handle is closed; we upload the file if it was newly created
// Release is called when file handle is closed; upload the temp file to Drive
func (fs *GDriveFS) Release(path string, fh uint64) int {
    fs.mu.Lock()
    f, ok := fs.handles[fh]
    name := fs.tempNames[fh]
    delete(fs.handles, fh)
    delete(fs.tempNames, fh)
    fs.mu.Unlock()
    if !ok {
        return 0
    }
    // get size before close for Explorer
    statInfo, _ := f.Stat()
    fileSize := uint64(0)
    if statInfo != nil {
        fileSize = uint64(statInfo.Size())
    }
    // temporarily expose in-memory size for Explorer
    fs.mu.Lock()
    fs.index[name] = &googleDrive.File{Name: p.Base(name), Size: int64(fileSize)}
    fs.mu.Unlock()
    f.Close()
    // Upload
    tmpReader, err := os.Open(f.Name())
    if err != nil {
        log.Printf("open temp for upload err: %v", err)
        return 0
    }
    defer tmpReader.Close()
    // Determine target parent folder ID based on path of the new file
    baseName := p.Base(name)
    parentID := "root"
    parentPath := p.Dir(name)
    if parentPath != "." && parentPath != "" {
        fs.mu.RLock()
        if pFile, ok := fs.index[parentPath]; ok {
            parentID = pFile.Id
        }
        fs.mu.RUnlock()
    }
    
    _, err = fs.Drive.UploadFileToFolder(baseName, parentID, tmpReader)
    if err != nil {
        log.Printf("upload failed: %v", err)
    } else {
        log.Printf("uploaded %s to Drive", name)
        // refresh index for subsequent reads
        if err := fs.buildIndex(); err != nil {
            log.Printf("index refresh err: %v", err)
        }
    }
    os.Remove(f.Name())
    return 0
}

// Truncate resizes a file (needed by Windows before writes)
func (fs *GDriveFS) Truncate(path string, size int64, fh uint64) int {
    fs.mu.RLock()
    f, ok := fs.handles[fh]
    fs.mu.RUnlock()
    if ok {
        if err := f.Truncate(size); err != nil {
            return -fuse.EIO
        }
        return 0
    }
    // no handle; ignore
    return 0
}

// Flush ensures data is written to disk for a handle
// Rename moves/renames a file or directory locally (Drive change postponed)
func (fs *GDriveFS) Rename(oldpath, newpath string) int {
    oldclean := strings.TrimPrefix(oldpath, "/")
    newclean := strings.TrimPrefix(newpath, "/")
    fs.mu.Lock()
    f, ok := fs.index[oldclean]
    if ok {
        fs.index[newclean] = f
        delete(fs.index, oldclean)
    }
    fs.mu.Unlock()
    // TODO: call Drive API Files.Update to rename/move in background
    return 0
}

// No-op implementations required by Windows
func (fs *GDriveFS) Chmod(path string, mode uint32) int           { return 0 }
func (fs *GDriveFS) Chown(path string, uid, gid uint32) int       { return 0 }
func (fs *GDriveFS) Utimens(path string, tmsp []fuse.Timespec) int { return 0 }

func (fs *GDriveFS) Flush(path string, fh uint64) int {
    fs.mu.RLock()
    f, ok := fs.handles[fh]
    fs.mu.RUnlock()
    if ok {
        f.Sync()
    }
    return 0
}

// buildIndex fetches root folder listing and builds path index
func (fs *GDriveFS) buildIndex() error {
    if fs.Drive == nil {
        return fmt.Errorf("Drive service not set")
    }
    files, err := fs.Drive.ListAllFiles()
    if err != nil {
        return err
    }
    fs.mu.Lock()
    // Build maps
    fs.index = make(map[string]*googleDrive.File)
    fs.fileCache = make(map[string][]byte)
    idToFile := make(map[string]*googleDrive.File)
    parentsMap := make(map[string][]string) // childID -> parents
    for _, f := range files {
        idToFile[f.Id] = f
        if len(f.Parents) > 0 {
            parentsMap[f.Id] = f.Parents
        } else {
            // orphan, treat as root child
            parentsMap[f.Id] = []string{"root"}
        }
    }
    // Build path for each
    pathCache := map[string]string{"root": ""}
    var resolvePath func(id string) string
    resolvePath = func(id string) string {
        if p, ok := pathCache[id]; ok {
            return p
        }
        prnts := parentsMap[id]
        if len(prnts) == 0 {
            return "" // orphan
        }
        parentPath := resolvePath(prnts[0]) // use first parent for now
        if parentPath == "" {
            pathCache[id] = idToFile[id].Name
        } else {
            pathCache[id] = p.Join(parentPath, idToFile[id].Name)
        }
        return pathCache[id]
    }
    for id := range idToFile {
        p := resolvePath(id)
        if p == "" {
            continue
        }
        fs.index[p] = idToFile[id]
    }
    fs.mu.Unlock()
    return nil
}

// refreshQuota updates quota information from Drive API
func (fs *GDriveFS) refreshQuota() {
    if fs.Drive == nil {
        return
    }
    total, used, err := fs.Drive.GetQuota()
    if err != nil {
        log.Printf("Failed to refresh Drive quota: %v", err)
        return
    }
    fs.mu.Lock()
    fs.quotaTotal = total
    fs.quotaUsed = used
    fs.lastQuota = time.Now()
    fs.mu.Unlock()
}

// cleanupMountPoint attempts to remove the mount point directory
func cleanupMountPoint(mountPoint string) {
	if _, err := os.Stat(mountPoint); os.IsNotExist(err) {
		return // Directory doesn't exist, nothing to clean up
	}

	// On Windows, we need to wait a bit before cleanup
	if runtime.GOOS == "windows" {
		time.Sleep(1 * time.Second) // Increased wait time for Windows

		// Try to remove any lingering handles using Windows commands
		cmd := exec.Command("cmd", "/c", "rmdir", "/s", "/q", mountPoint)
		if err := cmd.Run(); err == nil {
			log.Printf("Cleaned up mount point: %s", mountPoint)
			return
		}
	}

	// Fallback to standard removal
	if entries, _ := os.ReadDir(mountPoint); len(entries) == 0 {
		if err := os.Remove(mountPoint); err != nil {
			log.Printf("Warning: Failed to remove mount point: %v", err)
		} else {
			log.Printf("Removed mount point: %s", mountPoint)
		}
	} else {
		log.Printf("Mount point not empty, skipping removal: %s", mountPoint)
	}
}

// Mount initializes and mounts the FUSE filesystem and returns the host for unmounting
func Mount(mountPoint string, drv *gdrive.DriveService) (*fuse.FileSystemHost, error) {
	// For drive letters, skip the absolute path conversion
	if !strings.HasSuffix(mountPoint, ":") {
		// Convert to absolute path for directory mounts
		var err error
		mountPoint, err = filepath.Abs(mountPoint)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %v", err)
		}

		log.Printf("Preparing to mount at directory: %s", mountPoint)

		// Clean up any existing mount point
		log.Println("Cleaning up any existing mount point...")
		cleanupMountPoint(mountPoint)

		// Create mount point directory with 0700 permissions (more secure)
		log.Println("Creating mount point...")
		if err := os.MkdirAll(mountPoint, 0700); err != nil {
			return nil, fmt.Errorf("failed to create mount point: %v", err)
		}
	} else {
		log.Printf("Preparing to mount at drive: %s", mountPoint)
	}

	log.Printf("Mounting GDriveFS at %s", mountPoint)

	// Initialize filesystem
	fs := &GDriveFS{Drive: drv, handles: make(map[uint64]*os.File), tempNames: make(map[uint64]string)}
    fs.refreshQuota()
    if err := fs.buildIndex(); err != nil {
        log.Printf("Failed to build index: %v", err)
    }
	
	// Create FUSE host
	host := fuse.NewFileSystemHost(fs)
	
	// Set mount options - match memfs defaults
	options := []string{
		"-o", "debug",                  // Enable debug output
		"-o", "umask=0",                 // Full permissions
		"-o", "uid=-1",                  // Current user
		"-o", "gid=-1",                  // Current group
		"-o", "FileInfoTimeout=0",
		"-o", "VolumeInfoTimeout=0",
		"-o", "VolumeSerialNumber=0",
		"-o", "FileSystemName=GDriveFS", // Name shown in Windows Explorer
		"-o", "volname=GDrive",
	}

	// Try to unmount first in case of previous unclean shutdown
	host.Unmount()

	// Mount the filesystem
	if !host.Mount(mountPoint, options) {
		// If mount fails, clean up the mount point
		cleanupMountPoint(mountPoint)
		return nil, fmt.Errorf("mount failed - is the mount point in use?")
	}

	log.Println("Filesystem mounted successfully")
	return host, nil
}
