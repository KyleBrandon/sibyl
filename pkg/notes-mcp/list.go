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
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListNotesRequest struct {
	Path      string `json:"path,omitempty" mcp:"Directory path (optional, defaults to vault root)"`
	Recursive bool   `json:"recursive,omitempty" mcp:"Whether to list recursively"`
}

type ListFoldersRequest struct {
	Path      string `json:"path,omitempty" mcp:"Directory path (optional, defaults to vault root)"`
	Recursive bool   `json:"recursive,omitempty" mcp:"Whether to list recursively"`
}

func (ns *NotesServer) NewListNotesTool() *mcp.ServerTool {
	return mcp.NewServerTool(
		"list_notes",
		"List notes in a directory",
		ns.ListNotes,
		mcp.Input(
			mcp.Property("path", mcp.Description("Directory path (option, defaults to vault root)"), mcp.Required(false)),
			mcp.Property("recursive", mcp.Description("Whether to list recursively"), mcp.Required(false)),
		),
	)
}

// ListNotes lists notes in a directory
func (ns *NotesServer) ListNotes(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[ListNotesRequest]) (*mcp.CallToolResultFor[any], error) {
	path := req.Arguments.Path
	if path == "" {
		path = ns.vaultDir
	}
	recursive := req.Arguments.Recursive

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

	return &mcp.CallToolResultFor[any]{
		Content: []*mcp.Content{
			mcp.NewTextContent(string(result)),
		},
	}, nil
}

func (ns *NotesServer) NewListFoldersTool() *mcp.ServerTool {
	return mcp.NewServerTool(
		"list_folders",
		"List the folders at the given path",
		ns.ListFolders,
		mcp.Input(
			mcp.Property("path", mcp.Description("Directory path (option, defaults to vault root)"), mcp.Required(true)),
			mcp.Property("recursive", mcp.Description("Whether to return all sub folders")),
		),
	)
}

// ListFolders gets a list of folders at the 'path' location.
func (ns *NotesServer) ListFolders(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[ListFoldersRequest]) (*mcp.CallToolResultFor[any], error) {
	path := req.Arguments.Path
	recursive := req.Arguments.Recursive

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

	return &mcp.CallToolResultFor[any]{
		Content: []*mcp.Content{
			mcp.NewTextContent(string(result)),
		},
	}, nil
}
