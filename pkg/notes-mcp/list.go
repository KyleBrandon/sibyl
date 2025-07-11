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

func (ns *NotesServer) NewListNotesTool() mcp.Tool {
	return mcp.NewTool(
		"list_notes",
		mcp.WithDescription("List notes in a directory"),
		mcp.WithString("path",
			mcp.Description("Directory path (option, defaults to vault root)"),
		),
		mcp.WithBoolean("recursive",
			mcp.DefaultBool(false),
			mcp.Description("Whether to list recursively"),
		),
	)
}

// ListNotes lists notes in a directory
func (s *NotesServer) ListNotes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		// no path specified, default to notes root
		path = ""
	}

	recursive, err := req.RequireBool("recursive")
	if err != nil {
		return nil, err
	}

	fullPath, err := utils.ValidatePath(s.vaultDir, path)
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
				relativePath, _ := filepath.Rel(s.vaultDir, path)
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
				relativePath, _ := filepath.Rel(s.vaultDir, entryPath)
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

func (ns *NotesServer) NewListFoldersTool() mcp.Tool {
	return mcp.NewTool(
		"list_folders",
		mcp.WithDescription("List the folders at the given path"),
		mcp.WithString("path",
			mcp.Description("Directory path (option, defaults to vault root)"),
		),
		mcp.WithBoolean("recursive",
			mcp.DefaultBool(false),
			mcp.Description("Whether to return all sub folders"),
		),
	)
}

// ListFolders gets a list of folders at the 'path' location.
func (s *NotesServer) ListFolders(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		// no path specified, default to notes root
		path = ""
	}

	recursive, err := req.RequireBool("recursive")
	if err != nil {
		return nil, err
	}

	fullPath, err := utils.ValidatePath(s.vaultDir, path)
	if err != nil {
		return nil, err
	}

	var folders []dto.NoteMetadata

	if recursive {
		err = utils.WalkDir(fullPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() && path != s.vaultDir {
				relativePath, _ := filepath.Rel(s.vaultDir, path)
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
				relativePath, _ := filepath.Rel(s.vaultDir, entryPath)
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
