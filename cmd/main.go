package cmd

import (
	"GDrive/internal/drive"
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

}
