package cmd

import (
	"GDrive/internal/drive"
	"GDrive/internal/fs"
	"fmt"
	"log"

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

	// Mount Google Drive
	fs.Mount("/mnt/gdrive")

	
}
