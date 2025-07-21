package notes

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/KyleBrandon/sibyl/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"
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

func (ns *NotesServer) NewWriteNoteTool() {
	tool := mcp.NewTool(
		"write_note",
		mcp.WithDescription("Write the contents to the note"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Path to the note file to write the contents to"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Text content to write to the note file"),
		),
	)

	ns.McpServer.AddTool(tool, mcp.NewTypedToolHandler(ns.WriteNote))
}

// WriteNote writes content to a note
func (ns *NotesServer) WriteNote(ctx context.Context, req mcp.CallToolRequest, params WriteNoteRequest) (*mcp.CallToolResult, error) {
	path := params.Path
	content := params.Content

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
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Successfully wrote to note: %s", relativePath)),
		},
	}, nil
}

func (ns *NotesServer) NewAppendNoteTool() {
	tool := mcp.NewTool(
		"append_note",
		mcp.WithDescription("Append the contents to the specified note file"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Path to the note file to append the contents to"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Text content to write to the note file"),
		),
	)

	ns.McpServer.AddTool(tool, mcp.NewTypedToolHandler(ns.AppendNote))
}

// AppendNote writes content to the end of an existing note
func (ns *NotesServer) AppendNote(ctx context.Context, req mcp.CallToolRequest, params AppendNoteRequest) (*mcp.CallToolResult, error) {
	path := params.Path

	content := params.Content

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
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Successfully wrote to note: %s", relativePath)),
		},
	}, nil
}

func (ns *NotesServer) NewCreateFolderTool() {
	tool := mcp.NewTool(
		"create_folder",
		mcp.WithDescription("Create a new folder"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Path to the folder to create"),
		),
	)

	ns.McpServer.AddTool(tool, mcp.NewTypedToolHandler(ns.CreateFolder))
}

// CreateFolder creates a new folder
func (ns *NotesServer) CreateFolder(ctx context.Context, req mcp.CallToolRequest, params CreateFolderRequest) (*mcp.CallToolResult, error) {
	path := params.Path

	fullPath, err := utils.ValidatePath(ns.vaultDir, path)
	if err != nil {
		return nil, err
	}

	if err := utils.MkdirAll(fullPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	relativePath, _ := filepath.Rel(ns.vaultDir, fullPath)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Successfully created folder: %s", relativePath)),
		},
	}, nil
}
