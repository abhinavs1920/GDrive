package fs

import (
	"fmt"

	"github.com/winfsp/cgofuse/fuse"

)

// GDriveFS struct represents our virtual filesystem
type GDriveFS struct {
	fuse.FileSystemBase
}

// Read handles file reading
func (fs *GDriveFS) Read(path string, buff []byte, offset int64, fh uint64) int {
	fmt.Println("Reading file:", path)
	return 0
}

// Write handles file writing
func (fs *GDriveFS) Write(path string, buff []byte, offset int64, fh uint64) int {
	fmt.Println("Writing to file:", path)
	return len(buff)
}

// Mount initializes FUSE
func Mount(mountPoint string) {
	fs := &GDriveFS{}
	host := fuse.NewFileSystemHost(fs)
	host.Mount(mountPoint, nil)
}
