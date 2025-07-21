package notes

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/KyleBrandon/sibyl/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"
)

type ReadNoteRequest struct {
	Path string `json:"path,omitempty" mcp:"Path to the note file to read the contents from"`
}

func (ns *NotesServer) NewReadNoteTool() {
	tool := mcp.NewTool(
		"read_note",
		mcp.WithDescription("Read the contents of the note from the file location"),
		mcp.WithString("path", mcp.Description("Path to the note file"), mcp.Required()),
	)

	ns.McpServer.AddTool(tool, mcp.NewTypedToolHandler(ns.ReadNote))
	return
}

// ReadNote reads the contents of a note
func (ns *NotesServer) ReadNote(ctx context.Context, req mcp.CallToolRequest, params ReadNoteRequest) (*mcp.CallToolResult, error) {
	path := params.Path

	fullPath, err := utils.ValidatePath(ns.vaultDir, path)
	if err != nil {
		slog.Error("Failed to validate path", "path", path, "error", err)
		return nil, err
	}

	// Check if file exists and is a file
	// TODO: validate file?
	info, err := utils.Stat(fullPath)
	if err != nil {
		slog.Error("file not found", "fullPath", fullPath, "error", err)
		return nil, fmt.Errorf("file not found: %s", path)
	}

	if info.IsDir() {
		slog.Error("path is a directory not a file", "fullPath", fullPath, "error", err)
		return nil, fmt.Errorf("path is a directory, not a file: %s", path)
	}

	// Read file contents
	content, err := utils.ReadFile(fullPath)
	if err != nil {
		slog.Error("failed toread file", "fullPath", fullPath, "error", err)
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(content)),
		},
	}, nil
}
