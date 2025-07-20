package dto

type DriveFileResult struct {
	ID           string `json:"id,omitempty"`
	FolderID     string `json:"folder_id,omitempty"`
	Name         string `json:"name,omitempty"`
	MimeType     string `json:"mimeType,omitempty"`
	Size         int64  `json:"size,omitempty"`
	ModifiedTime string `json:"modifiedTime,omitempty"`
	WebViewLink  string `json:"webViewLink,omitempty"`
}
