package notes

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/KyleBrandon/sibyl/pkg/utils"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type WriteNoteRequest struct {
	Path    string `json:"path,omitempty" mcp:"Path to the note file to write the contents to"`
	Content string `json:"content,omitempty" mcp:"Text content to write to the note file"`
}

type AppendNoteRequest struct {
	Path    string `json:"path,omitempty" mcp:"Path to the note file to append the contents to"`
	Content string `json:"content,omitempty" mcp:"Text content to append to the end of the note file"`
}

type CreateFolderRequest struct {
	Path string `json:"path,omitempty" mcp:"Path to the note file to append the contents to"`
}

func (ns *NotesServer) NewWriteNoteTool() *mcp.ServerTool {
	return mcp.NewServerTool(
		"write_note",
		"Write the contents to the note",
		ns.WriteNote,
		mcp.Input(
			mcp.Property("path",
				mcp.Required(true),
				mcp.Description("Path to the note file to write the contents to"),
			),
			mcp.Property("content",
				mcp.Required(true),
				mcp.Description("Text content to write to the note file"),
			),
		),
	)
}

// WriteNote writes content to a note
func (ns *NotesServer) WriteNote(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[WriteNoteRequest]) (*mcp.CallToolResultFor[any], error) {
	path := req.Arguments.Path
	content := req.Arguments.Content

	fullPath, err := utils.ValidatePath(ns.vaultDir, path)
	if err != nil {
		return nil, err
	}

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	// Permissions:
	// 	Owner=rwx
	// 	Group=rx
	// 	Other=rx
	if err := utils.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	// Permissions:
	// 	Owner=rw
	// 	Group=r
	// 	Other=r
	if err := utils.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	relativePath, _ := filepath.Rel(ns.vaultDir, fullPath)
	return &mcp.CallToolResultFor[any]{
		Content: []*mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Successfully wrote to note: %s", relativePath)),
		},
	}, nil
}

func (ns *NotesServer) NewAppendNoteTool() *mcp.ServerTool {
	return mcp.NewServerTool(
		"append_note",
		"Append the contents to the specified note file",
		ns.AppendNote,
		mcp.Input(
			mcp.Property("path",
				mcp.Required(true),
				mcp.Description("Path to the note file to append the contents to"),
			),
			mcp.Property("content",
				mcp.Required(true),
				mcp.Description("Text content to write to the note file"),
			),
		))
}

// AppendNote writes content to the end of an existing note
func (ns *NotesServer) AppendNote(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[AppendNoteRequest]) (*mcp.CallToolResultFor[any], error) {
	path := req.Arguments.Path

	content := req.Arguments.Content

	fullPath, err := utils.ValidatePath(ns.vaultDir, path)
	if err != nil {
		return nil, err
	}

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	// Permissions:
	// 	Owner=rwx
	// 	Group=rx
	// 	Other=rx
	if err := utils.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	// Permissions:
	// 	Owner=rw
	// 	Group=r
	// 	Other=r
	if err := utils.AppendFile(fullPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	relativePath, _ := filepath.Rel(ns.vaultDir, fullPath)
	return &mcp.CallToolResultFor[any]{
		Content: []*mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Successfully wrote to note: %s", relativePath)),
		},
	}, nil
}

func (ns *NotesServer) NewCreateFolderTool() *mcp.ServerTool {
	return mcp.NewServerTool(
		"create_folder",
		"Create a new folder",
		ns.CreateFolder,
		mcp.Input(
			mcp.Property("path",
				mcp.Required(true),
				mcp.Description("Path to the folder to create"),
			),
		),
	)
}

// CreateFolder creates a new folder
func (ns *NotesServer) CreateFolder(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[CreateFolderRequest]) (*mcp.CallToolResultFor[any], error) {
	path := req.Arguments.Path

	fullPath, err := utils.ValidatePath(ns.vaultDir, path)
	if err != nil {
		return nil, err
	}

	if err := utils.MkdirAll(fullPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	relativePath, _ := filepath.Rel(ns.vaultDir, fullPath)
	return &mcp.CallToolResultFor[any]{
		Content: []*mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Successfully created folder: %s", relativePath)),
		},
	}, nil
}
