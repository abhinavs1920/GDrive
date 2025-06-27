package drive

import (
    "fmt"
    "io"
    "net/http"

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

// UploadFile uploads a file to Drive root
func (d *DriveService) UploadFile(filename string, file io.Reader) (*googleDrive.File, error) {
    return d.UploadFileToFolder(filename, "root", file)
}

// UploadFileToFolder uploads a file to the given parent folderID ("root" for MyDrive root)
func (d *DriveService) UploadFileToFolder(filename, parentID string, file io.Reader) (*googleDrive.File, error) {
    fileMetadata := &googleDrive.File{Name: filename, Parents: []string{parentID}}
    driveFile, err := d.client.Files.Create(fileMetadata).Media(file).Do()
    if err != nil {
        return nil, fmt.Errorf("unable to upload file: %v", err)
    }
    return driveFile, nil
}

// DownloadFile downloads or exports a file from Google Drive depending on its type.
// For native Google docs it chooses a sensible Office format.
func (d *DriveService) DownloadFile(file *googleDrive.File) ([]byte, error) {
    var resp *http.Response
    var err error
    switch file.MimeType {
    case "application/vnd.google-apps.spreadsheet":
        resp, err = d.client.Files.Export(file.Id, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet").Download()
    case "application/vnd.google-apps.document":
        resp, err = d.client.Files.Export(file.Id, "application/vnd.openxmlformats-officedocument.wordprocessingml.document").Download()
    case "application/vnd.google-apps.presentation":
        resp, err = d.client.Files.Export(file.Id, "application/vnd.openxmlformats-officedocument.presentationml.presentation").Download()
    default:
        resp, err = d.client.Files.Get(file.Id).Download()
    }
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

// DownloadFileLegacy kept for compatibility with older callers.
func (d *DriveService) DownloadFileLegacy(fileID string) ([]byte, error) {
    resp, err := d.client.Files.Get(fileID).Download()
    if err != nil {
        return nil, fmt.Errorf("unable to download file: %v", err)
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}


// Deprecated: use DownloadFileByID or DownloadFileLegacy; kept for backward compat

func (d *DriveService) DownloadFileByID(fileID string) ([]byte, error) {
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
