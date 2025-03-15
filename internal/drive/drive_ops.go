package drive

import (
	"fmt"
	"io"

	"google.golang.org/api/drive/v3"

)

// DriveService struct holds the Drive client
type DriveService struct {
	client *drive.Service
}

// NewDriveService initializes a DriveService
func NewDriveService(client *drive.Service) *DriveService {
	return &DriveService{client: client}
}

// UploadFile uploads a file to Google Drive
func (d *DriveService) UploadFile(filename string, file io.Reader) (*drive.File, error) {
	fileMetadata := &drive.File{Name: filename}
	driveFile, err := d.client.Files.Create(fileMetadata).Media(file).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to upload file: %v", err)
	}
	return driveFile, nil
}

// DownloadFile downloads a file from Google Drive
func (d *DriveService) DownloadFile(fileID string) ([]byte, error) {
	resp, err := d.client.Files.Get(fileID).Download()
	if err != nil {
		return nil, fmt.Errorf("unable to download file: %v", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading file data: %v", err)
	}

	return data, nil
}
