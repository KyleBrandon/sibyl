package notes

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/KyleBrandon/sibyl/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"
)

func (ns *NotesServer) NewWriteNoteTool() mcp.Tool {
	return mcp.NewTool("write_note",
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
}

// WriteNote writes content to a note
func (s *NotesServer) WriteNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return nil, err
	}

	content, err := req.RequireString("content")
	if err != nil {
		return nil, err
	}

	fullPath, err := utils.ValidatePath(s.vaultDir, path)
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

	relativePath, _ := filepath.Rel(s.vaultDir, fullPath)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Successfully wrote to note: %s", relativePath)),
		},
	}, nil
}

func (ns *NotesServer) NewAppendNoteTool() mcp.Tool {
	return mcp.NewTool("append_note",
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
}

// AppendNote writes content to the end of an existing note
func (s *NotesServer) AppendNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return nil, err
	}

	content, err := req.RequireString("content")
	if err != nil {
		return nil, err
	}

	fullPath, err := utils.ValidatePath(s.vaultDir, path)
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

	relativePath, _ := filepath.Rel(s.vaultDir, fullPath)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Successfully wrote to note: %s", relativePath)),
		},
	}, nil
}

func (ns *NotesServer) NewCreateFolderTool() mcp.Tool {
	return mcp.NewTool("create_folder",
		mcp.WithDescription("Create a new folder"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Path to the folder to create"),
		),
	)
}

// CreateFolder creates a new folder
func (s *NotesServer) CreateFolder(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return nil, err
	}

	fullPath, err := utils.ValidatePath(s.vaultDir, path)
	if err != nil {
		return nil, err
	}

	if err := utils.MkdirAll(fullPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	relativePath, _ := filepath.Rel(s.vaultDir, fullPath)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Successfully created folder: %s", relativePath)),
		},
	}, nil
}
