package notes

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/KyleBrandon/sibyl/pkg/dto"
	"github.com/KyleBrandon/sibyl/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"
)

type ListNotesRequest struct {
	Path      string `json:"path,omitempty" mcp:"Directory path (optional, defaults to vault root)"`
	Recursive bool   `json:"recursive,omitempty" mcp:"Whether to list recursively"`
}

type ListFoldersRequest struct {
	Path      string `json:"path,omitempty" mcp:"Directory path (optional, defaults to vault root)"`
	Recursive bool   `json:"recursive,omitempty" mcp:"Whether to list recursively"`
}

func (ns *NotesServer) NewListNotesTool() {
	tool := mcp.NewTool(
		"list_notes",
		mcp.WithDescription("List notes in a directory"),
		mcp.WithString("path", mcp.Description("Directory path (option, defaults to vault root)"), mcp.Required()),
		mcp.WithBoolean("recursive", mcp.Description("Whether to list recursively")),
	)
	ns.McpServer.AddTool(tool, mcp.NewTypedToolHandler(ns.ListNotes))
}

// ListNotes lists notes in a directory
func (ns *NotesServer) ListNotes(ctx context.Context, req mcp.CallToolRequest, params ListNotesRequest) (*mcp.CallToolResult, error) {
	path := params.Path
	if path == "" {
		path = ns.vaultDir
	}
	recursive := params.Recursive

	fullPath, err := utils.ValidatePath(ns.vaultDir, path)
	if err != nil {
		return nil, err
	}

	var notes []dto.NoteMetadata

	if recursive {
		err = utils.WalkDir(fullPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Only include markdown files for notes
			if !info.IsDir() && (strings.HasSuffix(strings.ToLower(info.Name()), ".md") || strings.HasSuffix(strings.ToLower(info.Name()), ".markdown")) {
				relativePath, _ := filepath.Rel(ns.vaultDir, path)
				notes = append(notes, dto.NoteMetadata{
					Name:     info.Name(),
					Path:     relativePath,
					Size:     info.Size(),
					Modified: info.ModTime(),
					IsDir:    false,
				})
			}
			return nil
		})
	} else {
		entries, err := utils.ReadDir(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			// Only include markdown files for notes
			if !info.IsDir() && (strings.HasSuffix(strings.ToLower(info.Name()), ".md") || strings.HasSuffix(strings.ToLower(info.Name()), ".markdown")) {
				entryPath := filepath.Join(fullPath, info.Name())
				relativePath, _ := filepath.Rel(ns.vaultDir, entryPath)
				notes = append(notes, dto.NoteMetadata{
					Name:     info.Name(),
					Path:     relativePath,
					Size:     info.Size(),
					Modified: info.ModTime(),
					IsDir:    false,
				})
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}

	result, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notes list: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(result)),
		},
	}, nil
}

func (ns *NotesServer) NewListFoldersTool() {
	tool := mcp.NewTool(
		"list_folders",
		mcp.WithDescription("List the folders at the given path"),
		mcp.WithString("path", mcp.Description("Directory path (option, defaults to vault root)"), mcp.Required()),
		mcp.WithBoolean("recursive", mcp.Description("Whether to return all sub folders")),
	)

	ns.McpServer.AddTool(tool, mcp.NewTypedToolHandler(ns.ListFolders))
}

// ListFolders gets a list of folders at the 'path' location.
func (ns *NotesServer) ListFolders(ctx context.Context, req mcp.CallToolRequest, params ListFoldersRequest) (*mcp.CallToolResult, error) {
	path := params.Path
	recursive := params.Recursive

	fullPath, err := utils.ValidatePath(ns.vaultDir, path)
	if err != nil {
		return nil, err
	}

	var folders []dto.NoteMetadata

	if recursive {
		err = utils.WalkDir(fullPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() && path != ns.vaultDir {
				relativePath, _ := filepath.Rel(ns.vaultDir, path)
				folders = append(folders, dto.NoteMetadata{
					Name:     info.Name(),
					Path:     relativePath,
					Size:     0,
					Modified: info.ModTime(),
					IsDir:    true,
				})
			}
			return nil
		})
	} else {
		entries, err := utils.ReadDir(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				info, err := entry.Info()
				if err != nil {
					continue
				}

				entryPath := filepath.Join(fullPath, info.Name())
				relativePath, _ := filepath.Rel(ns.vaultDir, entryPath)
				folders = append(folders, dto.NoteMetadata{
					Name:     info.Name(),
					Path:     relativePath,
					Size:     0,
					Modified: info.ModTime(),
					IsDir:    true,
				})
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	result, err := json.MarshalIndent(folders, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal folders list: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(result)),
		},
	}, nil
}
