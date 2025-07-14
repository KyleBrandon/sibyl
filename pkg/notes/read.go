package notes

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/KyleBrandon/sibyl/pkg/utils"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ReadNoteRequest struct {
	Path string `json:"path,omitempty" mcp:"Path to the note file to read the contents from"`
}

func (ns *NotesServer) NewReadNoteTool() *mcp.ServerTool {
	return mcp.NewServerTool(
		"read_note",
		"Read the contents of the note from the file location",
		ns.ReadNote,
		mcp.Input(
			mcp.Property("path", mcp.Description("Path to the note file"), mcp.Required(true)),
		),
	)
}

// ReadNote reads the contents of a note
func (ns *NotesServer) ReadNote(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[ReadNoteRequest]) (*mcp.CallToolResultFor[any], error) {
	path := req.Arguments.Path

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

	return &mcp.CallToolResultFor[any]{
		Content: []*mcp.Content{
			mcp.NewTextContent(string(content)),
		},
	}, nil
}
