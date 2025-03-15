package main

import (
	"GDrive/internal/cache"
	"GDrive/internal/drive"
	"GDrive/internal/fs"
	"fmt"
	"log"
	"os"

)

func main() {
	fmt.Println("Starting GDriveDisk...")

	// Authenticate Google Drive
	client, err := drive.AuthenticateGoogleDrive()
	if err != nil {
		log.Fatalf("Failed to authenticate Google Drive: %v", err)
	}

	// Initialize Drive Service
	driveService := drive.NewDriveService(client)

	// Test Uploading a File
	testFile, err := os.Open("test.txt") // Ensure test.txt exists in the project root
	if err != nil {
		log.Fatalf("Failed to open test file: %v", err)
	}
	defer testFile.Close()

	uploadedFile, err := driveService.UploadFile("test.txt", testFile)
	if err != nil {
		log.Fatalf("File upload failed: %v", err)
	}

	fmt.Printf("File uploaded successfully! ID: %s\n", uploadedFile.Id)

	// Mount the FUSE filesystem
	fs.Mount("/mnt/gdrivedisk")

	// Initialize Cache
	redisCache := cache.NewRedisCache("localhost:6379")
	redisCache.SetCache("test", "hello", 0)

}
