package main

import (
	"GDrive/internal/drive"
	"GDrive/internal/fs"
	"io"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

func main() {
	// Setup logging
	logFile, err := os.OpenFile("gdrive.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open log file")
	}
	defer logFile.Close()
	
	// Log to both file and console
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("=== Starting GDriveDisk ===")
	log.Printf("OS: %s, Arch: %s", runtime.GOOS, runtime.GOARCH)

	// Set mount point to a drive letter (make sure it's not in use)
	mountPoint := "X:" // Try X: or any other available drive letter

	// Authenticate Google Drive
	log.Println("Authenticating with Google Drive...")
	client, err := drive.AuthenticateGoogleDrive()
	if err != nil {
		log.Fatalf("Failed to authenticate Google Drive: %v", err)
	}

	// Initialize Drive Service
	driveService := drive.NewDriveService(client)

	// Test uploading a file if it exists
	if _, err := os.Stat("test.txt"); err == nil {
		log.Println("Found test.txt, attempting to upload...")
		testFile, err := os.Open("test.txt")
		if err != nil {
			log.Printf("Warning: Failed to open test file: %v", err)
		} else {
			defer testFile.Close()
			uploadedFile, err := driveService.UploadFile("test.txt", testFile)
			if err != nil {
				log.Printf("Warning: File upload failed: %v", err)
			} else {
				log.Printf("Test file uploaded successfully! ID: %s\n", uploadedFile.Id)
			}
		}
	}

	// Mount the FUSE filesystem
	log.Printf("Mounting GDrive at %s...", mountPoint)
	host, err := fs.Mount(mountPoint, driveService)
	if err != nil {
		log.Printf("Failed to mount filesystem: %v", err)
		log.Println("This could be due to:")
		log.Println("1. Another instance is already running")
		log.Println("2. Previous mount wasn't properly unmounted")
		log.Println("3. WinFsp is not properly installed")
		log.Println("4. Insufficient permissions")
		os.Exit(1)
	}

	log.Println("Successfully mounted GDrive at", mountPoint)
	log.Println("Press Ctrl+C to unmount and exit")

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("\nShutting down...")

	// Unmount before exiting
	if host != nil {
		host.Unmount()
		// Clean up the mount point if it's a directory
		if !strings.HasSuffix(mountPoint, ":") {
			os.Remove(mountPoint)
		}
	}

	log.Println("GDrive has been unmounted")
}
