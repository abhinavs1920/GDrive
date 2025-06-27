package drive

import (
	"fmt"
	"io"

	googleDrive "google.golang.org/api/drive/v3"
)

// DriveService struct holds the Drive client
type DriveService struct {
	client *googleDrive.Service
}

// NewDriveService initializes a DriveService
func NewDriveService(client *googleDrive.Service) *DriveService {
	return &DriveService{client: client}
}

// UploadFile uploads a file to Google Drive
func (d *DriveService) UploadFile(filename string, file io.Reader) (*googleDrive.File, error) {
	fileMetadata := &googleDrive.File{Name: filename}
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

// ListFilesInFolder lists files in given folderID ("root" for My Drive root) limited to 1000.
func (d *DriveService) ListFilesInFolder(folderID string) ([]*googleDrive.File, error) {
    var files []*googleDrive.File
    pageTok := ""
    for {
        req := d.client.Files.List().Q(fmt.Sprintf("'%s' in parents and trashed=false", folderID)).Fields("nextPageToken, files(id,name,mimeType,size,parents)").PageSize(1000)
        if pageTok != "" {
            req = req.PageToken(pageTok)
        }
        resp, err := req.Do()
        if err != nil {
            return nil, fmt.Errorf("failed to list files: %v", err)
        }
        files = append(files, resp.Files...)
        if resp.NextPageToken == "" {
            break
        }
        pageTok = resp.NextPageToken
    }
    return files, nil
}

// ListAllFiles retrieves all non-trashed files in the drive with parents information.
func (d *DriveService) ListAllFiles() ([]*googleDrive.File, error) {
    var files []*googleDrive.File
    pageTok := ""
    for {
        req := d.client.Files.List().Q("trashed=false").Fields("nextPageToken, files(id,name,mimeType,size,parents)").PageSize(1000)
        if pageTok != "" {
            req = req.PageToken(pageTok)
        }
        resp, err := req.Do()
        if err != nil {
            return nil, fmt.Errorf("failed to list files: %v", err)
        }
        files = append(files, resp.Files...)
        if resp.NextPageToken == "" {
            break
        }
        pageTok = resp.NextPageToken
    }
    return files, nil
}

// GetQuota returns total and used storage bytes.
// total == 0 means unlimited.
func (d *DriveService) GetQuota() (total uint64, used uint64, err error) {
    about, err := d.client.About.Get().Fields("storageQuota").Do()
    if err != nil {
        return 0, 0, fmt.Errorf("failed to get Drive quota: %v", err)
    }
    if about.StorageQuota == nil {
        return 0, 0, fmt.Errorf("storageQuota not available")
    }
    total = uint64(about.StorageQuota.Limit)
    used = uint64(about.StorageQuota.Usage)
    return
}
