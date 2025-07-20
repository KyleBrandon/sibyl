// Package gcp contains the MCP server for loading reMarkable PDFs from GCP
package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/KyleBrandon/sibyl/pkg/dto"
	"github.com/KyleBrandon/sibyl/pkg/utils"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type GCPServer struct {
	ctx          context.Context
	McpServer    *mcp.Server
	driveService *drive.Service
	folderID     string
}

type SearchDriveFilesRequest struct {
	Query string `json:"query"`
}

type ReadDriveFileRequest struct {
	FileID string `json:"file_id"`
}

func NewGCPServer(ctx context.Context, credentialsPath, notesFolderID string) (*GCPServer, error) {
	server := &GCPServer{}

	// Initialize Google Drive service
	driveService, err := drive.NewService(
		ctx,
		option.WithCredentialsFile(credentialsPath),
		option.WithScopes(drive.DriveScope))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	serverOptions := mcp.ServerOptions{
		InitializedHandler:      server.handleInitialized,
		RootsListChangedHandler: server.handleRootsListChanged,
	}

	server.ctx = ctx
	server.driveService = driveService
	server.folderID = notesFolderID
	server.McpServer = mcp.NewServer("gcp-server", "v1.0.0", &serverOptions)
	server.addTools()

	return server, nil
}

func (gs *GCPServer) handleInitialized(ctx context.Context, session *mcp.ServerSession, params *mcp.InitializedParams) {
	slog.Info("Initialized", "params", params)
}

// handleRootsListChanged will receive a "root changed" event from the client and update the note server to use the new root folder
func (gs *GCPServer) handleRootsListChanged(ctx context.Context, session *mcp.ServerSession, params *mcp.RootsListChangedParams) {
	result, err := session.ListRoots(ctx, &mcp.ListRootsParams{})
	if err != nil {
		slog.Error("Failed to get the roots", "error", err)
		return
	}

	if len(result.Roots) != 1 {
		slog.Error("We only support a single root at this time")
		return
	}

	gs.setVaultFolder(result.Roots[0].URI)
}

func (gs *GCPServer) addTools() {
	// Add search files tool
	searchFilesTool := mcp.NewServerTool(
		"search_drive_files",
		"Search for files in Google Drive by name or query",
		gs.handleSearchFiles,
		mcp.Input(
			mcp.Property("query", mcp.Description("Search query (file name or search terms)"), mcp.Required(true)),
			mcp.Property("folder_id", mcp.Description("The Google Drive folder ID to search in"), mcp.Required(false)),
			mcp.Property("max_results", mcp.Description("Maximum number of results to return (default: 10)"), mcp.Required(false)),
		),
	)

	// // Add read file tool
	readFileTool := mcp.NewServerTool(
		"read_drive_file",
		"Read content from Google Drive file by file ID",
		gs.handleReadFile,
		mcp.Input(
			mcp.Property("file_id", mcp.Description("Google Drive file ID"), mcp.Required(true)),
		),
	)

	gs.McpServer.AddTools(searchFilesTool, readFileTool)
}

func (gs *GCPServer) handleSearchFiles(ctx context.Context, session *mcp.ServerSession, request *mcp.CallToolParamsFor[SearchDriveFilesRequest]) (*mcp.CallToolResultFor[any], error) {
	// Build search query for Google Drive
	query := fmt.Sprintf("name contains '%s' and trashed=false and '%s' in parents", request.Arguments.Query, gs.folderID)

	// Search for files
	files, err := gs.driveService.Files.List().
		Q(query).
		// PageSize(int64(maxResults)).
		Spaces("drive").
		Fields("files(id, name, parents, createdTime, modifiedTime, size, webViewLink)").
		Do()
	if err != nil {
		return &mcp.CallToolResultFor[any]{
			IsError: true,
			Content: []*mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Error searching files: %v", err)),
			},
		}, nil
	}

	// Format results
	results := make([]dto.DriveFileResult, 0)
	for _, file := range files.Files {
		results = append(results, dto.DriveFileResult{
			ID:           file.Id,
			Name:         file.Name,
			MimeType:     file.MimeType,
			Size:         file.Size,
			ModifiedTime: file.ModifiedTime,
			WebViewLink:  file.WebViewLink,
		})
	}

	resultJSON, _ := json.MarshalIndent(results, "", "  ")

	return &mcp.CallToolResultFor[any]{
		Content: []*mcp.Content{
			mcp.NewTextContent(string(resultJSON)),
		},
	}, nil
}

func (gs *GCPServer) handleReadFile(ctx context.Context, session *mcp.ServerSession, request *mcp.CallToolParamsFor[ReadDriveFileRequest]) (*mcp.CallToolResultFor[any], error) {
	fileID := request.Arguments.FileID

	// Get file metadata first
	file, err := gs.driveService.Files.Get(fileID).Do()
	if err != nil {
		return &mcp.CallToolResultFor[any]{
			IsError: true,
			Content: []*mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Error getting file metadata: %v", err)),
			},
		}, nil
	}

	// Handle different file types
	var fileContents []byte
	var downloadErr error

	if strings.HasPrefix(file.MimeType, "application/vnd.google-apps.") {
		// Google Workspace files need to be exported
		fileContents, downloadErr = gs.exportGoogleWorkspaceFile(fileID, file.MimeType)
	} else {
		// Regular files can be downloaded directly
		fileContents, downloadErr = gs.downloadRegularFile(fileID)
	}

	if downloadErr != nil {
		return &mcp.CallToolResultFor[any]{
			IsError: true,
			Content: []*mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Error reading file content: %v", downloadErr)),
			},
		}, nil
	}

	textContent := mcp.NewTextContent(fmt.Sprintf("File resource '%s'", file.Name))
	resource := mcp.NewBlobResourceContents(file.WebContentLink, file.MimeType, fileContents)
	resContent := mcp.NewResourceContent(resource)

	return &mcp.CallToolResultFor[any]{
		Content: []*mcp.Content{
			textContent,
			resContent,
		},
	}, nil
}

func (gs *GCPServer) exportGoogleWorkspaceFile(fileID, mimeType string) ([]byte, error) {
	var exportMimeType string

	// Map Google Workspace MIME types to exportable formats
	switch mimeType {
	case "application/vnd.google-apps.document":
		exportMimeType = "text/plain"
	case "application/vnd.google-apps.spreadsheet":
		exportMimeType = "text/csv"
	case "application/vnd.google-apps.presentation":
		exportMimeType = "text/plain"
	default:
		return nil, fmt.Errorf("unsupported Google Workspace file type: %s", mimeType)
	}

	resp, err := gs.driveService.Files.Export(fileID, exportMimeType).Download()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the exported content into a byte slice
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func (gs *GCPServer) downloadRegularFile(fileID string) ([]byte, error) {
	resp, err := gs.driveService.Files.Get(fileID).Download()
	if err != nil {
		slog.Error("Failed to get file from GCP", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Failed to read file body", "error", err)
		return nil, err
	}

	return content, nil
}

func (gs *GCPServer) setVaultFolder(vaultDir string) {
	// NOTE: Currently MCP Roots only support "file://" URIs.  Here we mimic this format for Google Drive:
	// 	"file://subFolder" -> "/subFolder"
	path, err := utils.FileURIToPath(vaultDir)
	if err != nil {
		gs.folderID = "root"
	} else {
		folderID, err := gs.GetFolderIDForPath(path)
		if err != nil {
			slog.Error("Failed to get the root folder ID", "error", err)
			return
		}

		gs.folderID = folderID
	}
}

// GetFolderIDForPath walks a slash-separated path under rootFolderID
// and returns the Drive folder ID of the final segment.
func (gs *GCPServer) GetFolderIDForPath(path string) (string, error) {
	q := "mimeType = 'application/vnd.google-apps.folder' and trashed = false"
	resp, err := gs.driveService.Files.List().
		Q(q).
		Fields("files(id, name, parents)").
		SupportsAllDrives(true).
		Do()
	if err != nil {
		return "", fmt.Errorf("error listing folders: %w", err)
	}

	// build a map of the current folder structure
	folderMap := make(map[string][]*drive.File)
	for _, f := range resp.Files {
		parents := f.Parents
		if len(parents) == 0 {
			parents = []string{"root"}
		}
		for _, parentID := range parents {
			folderMap[parentID] = append(folderMap[parentID], f)
		}
		slog.Info("Folder", "name", f.Name, "id", f.Id, "parents", parents)
	}

	// given a 'path', find the folderID by walking the parent map
	parts := strings.Split(strings.Trim(path, "/"), "/")
	parentID := "root"
	if len(parts) == 1 && parts[0] == "" {
		return parentID, nil
	}

	for _, name := range parts {
		slog.Info("findID", "name", name, "parentID", parentID)
		children, ok := folderMap[parentID]
		if !ok {
			return "", fmt.Errorf("no folders found under parent %s", parentID)
		}
		found := false
		for _, f := range children {
			if f.Name == name {
				parentID = f.Id
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("folder %q not found under %s", name, parentID)
		}
	}
	return parentID, nil
}
