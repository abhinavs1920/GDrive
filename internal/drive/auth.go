package drive

import (
	"context"

	"google.golang.org/api/drive/v3"
)

// AuthenticateGoogleDrive initializes Google Drive API client
func AuthenticateGoogleDrive() (*drive.Service, error) {
	ctx := context.Background()

}
