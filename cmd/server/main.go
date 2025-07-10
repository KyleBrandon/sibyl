package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ObsidianServer represents our MCP server for Obsidian notes
type ObsidianServer struct {
	mcpServer *server.MCPServer
	vaultDir  string
}

// NoteMetadata represents metadata about a note
type NoteMetadata struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	IsDir    bool      `json:"is_dir"`
}

// SearchResult represents a search result
type SearchResult struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Content string `json:"content"`
	Context string `json:"context"`
}

// Tool request structures

// ReadNoteRequest represents the request to read a note
type ReadNoteRequest struct {
	Path string `json:"path" mcp:"Path to the note file"`
}

// ListNotesRequest represents the request to list notes
type ListNotesRequest struct {
	Path      string `json:"path,omitempty" mcp:"Directory path (optional, defaults to vault root)"`
	Recursive bool   `json:"recursive,omitempty" mcp:"Whether to list recursively"`
}

// SearchNotesRequest represents the request to search notes
type SearchNotesRequest struct {
	Query         string `json:"query" mcp:"Search query"`
	Path          string `json:"path,omitempty" mcp:"Directory to search in (optional, defaults to vault root)"`
	CaseSensitive bool   `json:"case_sensitive,omitempty" mcp:"Whether search should be case sensitive"`
}

// GetFoldersRequest represents the request to get folders
type GetFoldersRequest struct {
	Path      string `json:"path,omitempty" mcp:"Directory path (optional, defaults to vault root)"`
	Recursive bool   `json:"recursive,omitempty" mcp:"Whether to list recursively"`
}

// WriteNoteRequest represents the request to write a note
type WriteNoteRequest struct {
	Path    string `json:"path" mcp:"Path to the note file"`
	Content string `json:"content" mcp:"Content to write to the note"`
}

// CreateFolderRequest represents the request to create a folder
type CreateFolderRequest struct {
	Path string `json:"path" mcp:"Path to the folder to create"`
}

type GetValueInfoRequest struct{}

// VaultInfo represents information about the Obsidian vault
type VaultInfo struct {
	Path        string   `json:"path" mcp:"Path to the ObsidianVault"`
	Tools       []string `json:"tools" mcp:"List of supported tools"`
	NoteCount   int      `json:"note_count" mcp:"Number of notes in the vault"`
	FolderCount int      `json:"folder_count" mcp:"Number of folders in the vault"`
}

// NewObsidianServer creates a new Obsidian MCP server
func NewObsidianServer(vaultDir string) (*ObsidianServer, error) {
	// Verify vault directory exists
	if _, err := os.Stat(vaultDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("vault directory does not exist: %s", vaultDir)
	}

	// Clean the vault directory path
	vaultDir = filepath.Clean(vaultDir)

	mcpServer := server.NewMCPServer("obsidian-notes", "1.0.0")

	s := &ObsidianServer{
		mcpServer: mcpServer,
		vaultDir:  vaultDir,
	}

	// Add tools
	s.addTools()

	return s, nil
}

// addTools adds all the tools to the server
func (s *ObsidianServer) addTools() {
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "read_note",
		Description: "Read the contents of a note",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the note file",
				},
			},
			Required: []string{"path"},
		},
	},
		s.ReadNote,
	)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "write_note",
		Description: "Write the contents to the note",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the note file",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Content to write to the note",
				},
			},
			Required: []string{"path"},
		},
	},
		s.WriteNote,
	)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "list_notes",
		Description: "List notes in a directory",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Directory path (option, defaults to vault root)",
				},
				"recursive": map[string]any{
					"type":        "bool",
					"description": "Whether to list recursively",
				},
			},

			Required: []string{"path"},
		},
	},
		s.ListNotes,
	)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "get_folders",
		Description: "Get a list of folders",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Directory path (option, defaults to vault root)",
				},
				"recursive": map[string]any{
					"type":        "bool",
					"description": "Whether to return all sub folders",
				},
			},

			Required: []string{"path"},
		},
	},
		s.GetFolders,
	)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "search_notes",
		Description: "Search for text within notes",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Directory path (option, defaults to vault root)",
				},
				"query": map[string]any{
					"type":        "string",
					"description": "Search query",
				},
				"case_sensitive": map[string]any{
					"type":        "bool",
					"description": "Whether search should be case sensitive",
				},
			},

			Required: []string{"path"},
		},
	},
		s.SearchNotes,
	)

	s.mcpServer.AddTool(mcp.Tool{
		Name:        "create_folder",
		Description: "Create a new folder",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the folder to create",
				},
			},

			Required: []string{"path"},
		},
	},
		s.CreateFolder,
	)

	// s.mcpServer.AddTool(mcp.Tool{
	// 	Name:        "get_valut_info",
	// 	Description: "Create a new folder",
	// 	// 	Type: "object",
	// 	// InputSchema: mcp.ToolInputSchema{
	// 	// 	Properties: map[string]any{
	// 	// 		"path": map[string]any{
	// 	// 			"type":        "string",
	// 	// 			"description": "Path to the folder to create",
	// 	// 		},
	// 	// 	},
	// 	//
	// 	// 	Required: []string{"path"},
	// 	// },
	// },
	// 	s.GetVaultInfo,
	// )
}

// validatePath ensures the path is within the vault directory
// TODO: break this out
func (s *ObsidianServer) validatePath(inputPath string) (string, error) {
	if inputPath == "" {
		slog.Info("input path is empty, use vault root")
		return s.vaultDir, nil
	}

	// Convert relative path to absolute path within vault
	var fullPath string
	if filepath.IsAbs(inputPath) {
		slog.Info("input path is absolute path", "inputPath", inputPath)
		fullPath = inputPath
	} else {
		fullPath = filepath.Join(s.vaultDir, inputPath)
		slog.Info("input path is relative", "inputPath", inputPath, "fullPath", fullPath)
	}

	// Clean the path
	fullPath = filepath.Clean(fullPath)

	// Ensure the path is within the vault directory
	if !strings.HasPrefix(fullPath, s.vaultDir) {
		return "", fmt.Errorf("path is outside vault directory: %s", inputPath)
	}

	return fullPath, nil
}

// ReadNote reads the contents of a note
// }, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
func (s *ObsidianServer) ReadNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		slog.Error("Missing 'path'", "error", err)
		return nil, err
	}

	fullPath, err := s.validatePath(path)
	if err != nil {
		slog.Error("Failed to validate path", "path", path, "error", err)
		return nil, err
	}

	// Check if file exists and is a file
	// TODO: validate file?
	info, err := s.Stat(fullPath)
	if err != nil {
		slog.Error("file not found", "fullPath", fullPath, "error", err)
		return nil, fmt.Errorf("file not found: %s", path)
	}

	if info.IsDir() {
		slog.Error("path is a directory not a file", "fullPath", fullPath, "error", err)
		return nil, fmt.Errorf("path is a directory, not a file: %s", path)
	}

	// Read file contents
	content, err := s.ReadFile(fullPath)
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

// WriteNote writes content to a note
func (s *ObsidianServer) WriteNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		slog.Error("Missing 'path'", "error", err)
		return nil, err
	}

	content, err := req.RequireString("content")
	if err != nil {
		slog.Error("Misdsing 'content'", "error", err)
		return nil, err
	}

	fullPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	// Permissions:
	// 	Owner=rwx
	// 	Group=rx
	// 	Other=rx
	if err := s.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	// Permissions:
	// 	Owner=rw
	// 	Group=r
	// 	Other=r
	if err := s.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	relativePath, _ := filepath.Rel(s.vaultDir, fullPath)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Successfully wrote to note: %s", relativePath)),
		},
	}, nil
}

// ListNotes lists notes in a directory
func (s *ObsidianServer) ListNotes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		slog.Error("Missing 'path'", "error", err)
		return nil, err
	}

	recursive, err := req.RequireBool("recursive")
	if err != nil {
		slog.Error("Missing 'recursive' flag", "error", err)
		return nil, err
	}

	fullPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	var notes []NoteMetadata

	if recursive {
		err = s.walkDir(fullPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Only include markdown files for notes
			if !info.IsDir() && (strings.HasSuffix(strings.ToLower(info.Name()), ".md") || strings.HasSuffix(strings.ToLower(info.Name()), ".markdown")) {
				relativePath, _ := filepath.Rel(s.vaultDir, path)
				notes = append(notes, NoteMetadata{
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
		entries, err := s.ReadDir(fullPath)
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
				notes = append(notes, NoteMetadata{
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

// SearchNotes searches for text within notes
func (s *ObsidianServer) SearchNotes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return nil, err
	}

	query, err := req.RequireString("query")
	if err != nil {
		return nil, err
	}

	caseSensitive, err := req.RequireBool("case_sensitive")
	if err != nil {
		return nil, err
	}

	fullPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	if !caseSensitive {
		query = strings.ToLower(query)
	}

	// Compile regex for search
	var pattern *regexp.Regexp
	if caseSensitive {
		pattern = regexp.MustCompile(regexp.QuoteMeta(query))
	} else {
		pattern = regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))
	}

	err = s.walkDir(fullPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only search in markdown files
		if info.IsDir() || (!strings.HasSuffix(strings.ToLower(info.Name()), ".md") && !strings.HasSuffix(strings.ToLower(info.Name()), ".markdown")) {
			return nil
		}

		content, err := s.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			if pattern.MatchString(line) {
				relativePath, _ := filepath.Rel(s.vaultDir, path)

				// Get context (3 lines before and after)
				contextStart := max(0, lineNum-3)
				contextEnd := min(len(lines), lineNum+4)
				context := strings.Join(lines[contextStart:contextEnd], "\n")

				results = append(results, SearchResult{
					Path:    relativePath,
					Line:    lineNum + 1,
					Content: strings.TrimSpace(line),
					Context: context,
				})
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search notes: %w", err)
	}

	result, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search results: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(result)),
		},
	}, nil
}

// GetFolders gets a list of folders
func (s *ObsidianServer) GetFolders(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return nil, err
	}

	recursive, err := req.RequireBool("recursive")
	if err != nil {
		slog.Error("Missing 'recursive' flag", "error", err)
		return nil, err
	}

	fullPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	var folders []NoteMetadata

	if recursive {
		err = s.walkDir(fullPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() && path != s.vaultDir {
				relativePath, _ := filepath.Rel(s.vaultDir, path)
				folders = append(folders, NoteMetadata{
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
		entries, err := s.ReadDir(fullPath)
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
				folders = append(folders, NoteMetadata{
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

// CreateFolder creates a new folder
func (s *ObsidianServer) CreateFolder(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return nil, err
	}

	fullPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	if err := s.MkdirAll(fullPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	relativePath, _ := filepath.Rel(s.vaultDir, fullPath)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Successfully created folder: %s", relativePath)),
		},
	}, nil
}

// GetVaultInfo gets information about the vault
func (s *ObsidianServer) GetVaultInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	info := VaultInfo{
		Path: s.vaultDir,
		Tools: []string{
			"read_note",
			"write_note",
			"list_notes",
			"search_notes",
			"get_folders",
			"create_folder",
			"get_vault_info",
		},
	}

	// Count notes and folders
	noteCount := 0
	folderCount := 0

	_ = s.walkDir(s.vaultDir, func(path string, fileInfo fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fileInfo.IsDir() {
			if path != s.vaultDir {
				folderCount++
			}
		} else if strings.HasSuffix(strings.ToLower(fileInfo.Name()), ".md") || strings.HasSuffix(strings.ToLower(fileInfo.Name()), ".markdown") {
			noteCount++
		}
		return nil
	})

	info.NoteCount = noteCount
	info.FolderCount = folderCount

	result, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vault info: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(result)),
		},
	}, nil
}

// File system abstraction methods for easier testing

func (s *ObsidianServer) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

func (s *ObsidianServer) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (s *ObsidianServer) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (s *ObsidianServer) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (s *ObsidianServer) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

func (s *ObsidianServer) walkDir(root string, walkFn filepath.WalkFunc) error {
	return filepath.Walk(root, walkFn)
}

// Utility functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Run starts the server
func (s *ObsidianServer) Run(ctx context.Context) error {
	log.Println("Starting sampling example server...")
	if err := server.ServeStdio(s.mcpServer); err != nil {
		slog.Error("Server error", "error", err)
		return err
	}

	return nil
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: obsidian-mcp-server <vault-directory>")
	}

	vaultDir := os.Args[1]

	server, err := NewObsidianServer(vaultDir)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	ctx := context.Background()
	if err := server.Run(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
