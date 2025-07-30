// Package notes_mcp contains the MCP tool implementations for processing Markdown notes
package notes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type NotesServer struct {
	ctx       context.Context
	McpServer *server.MCPServer
	vaultDir  string
}

func NewNotesServer(ctx context.Context, notesFolder string) *NotesServer {
	ns := &NotesServer{}

	// serverOptions := mcp.ServerOptions{
	// 	InitializedHandler:      ns.handleInitialized,
	// 	RootsListChangedHandler: ns.handleRootsListChanged,
	// }

	ns.ctx = ctx
	ns.vaultDir = notesFolder
	ns.McpServer = server.NewMCPServer("note-server", "v1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(false, false))
	ns.addTools()
	ns.addResources()

	return ns
}

// func (ns *NotesServer) setVaultFolder(vaultDir string) {
// 	localPath, err := utils.FileURIToPath(vaultDir)
// 	if err != nil {
// 		ns.vaultDir = ""
// 	} else {
// 		ns.vaultDir = localPath
// 	}
// }

func (ns *NotesServer) addResources() {
	// Resource 1: Note Files collection
	filesResource := mcp.NewResource(
		"notes://files/",
		"Note Files",
		mcp.WithResourceDescription("Collection of markdown note files in the vault"),
		mcp.WithMIMEType("application/json"),
	)
	ns.McpServer.AddResource(filesResource, ns.ListNoteFiles)

	// Resource 2: Note Templates
	templatesResource := mcp.NewResource(
		"notes://templates/",
		"Note Templates",
		mcp.WithResourceDescription("Available note templates for different purposes"),
		mcp.WithMIMEType("application/json"),
	)
	ns.McpServer.AddResource(templatesResource, ns.ListNoteTemplates)

	// Resource 3: Note Collections (by tags/folders)
	collectionsResource := mcp.NewResource(
		"notes://collections/",
		"Note Collections",
		mcp.WithResourceDescription("Grouped collections of notes by tags and folders"),
		mcp.WithMIMEType("application/json"),
	)
	ns.McpServer.AddResource(collectionsResource, ns.ListNoteCollections)
}

// addTools adds all the tools to the server

func (ns *NotesServer) addTools() {
	ns.NewReadNoteTool()
	ns.NewWriteNoteTool()
	ns.NewAppendNoteTool()
	ns.NewCreateFolderTool()
	ns.NewListNotesTool()
	ns.NewListFoldersTool()
	ns.NewSearchNotesTool()

	// Enhanced merge capabilities
	ns.NewMergeNoteTool()
	ns.NewPreviewMergeTool()

	// Template capabilities
	ns.NewGetTemplatesTools()
	ns.NewCreateFromTemplateTool()
}

// func (ns *NotesServer) handleInitialized(ctx context.Context, session *mcp.ServerSession, params *mcp.InitializedParams) {
// 	slog.Info("Initialized", "params", params)
// }
//
// // handleRootsListChanged will receive a "root changed" event from the client and update the note server to use the new root folder
// func (ns *NotesServer) handleRootsListChanged(ctx context.Context, session *mcp.ServerSession, params *mcp.RootsListChangedParams) {
// 	result, err := session.ListRoots(ctx, &mcp.ListRootsParams{})
// 	if err != nil {
// 		slog.Error("Failed to get the roots", "error", err)
// 		return
// 	}
//
// 	if len(result.Roots) != 1 {
// 		slog.Error("We only support a single root at this time")
// 		return
// 	}
//
// 	ns.setVaultFolder(result.Roots[0].URI)
// }

// Resource handlers

func (ns *NotesServer) ListNoteFiles(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	var noteFiles []map[string]interface{}

	err := filepath.Walk(ns.vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process markdown files
		if !info.IsDir() && (strings.HasSuffix(strings.ToLower(info.Name()), ".md") ||
			strings.HasSuffix(strings.ToLower(info.Name()), ".markdown")) {

			relPath, _ := filepath.Rel(ns.vaultDir, path)

			// Read file to get basic metadata
			content, err := os.ReadFile(path)
			if err != nil {
				return nil // Skip files we can't read
			}

			// Extract tags from frontmatter or content
			tags := extractTags(string(content))

			noteFile := map[string]interface{}{
				"path":     relPath,
				"name":     info.Name(),
				"size":     info.Size(),
				"modified": info.ModTime().Format(time.RFC3339),
				"uri":      fmt.Sprintf("notes://files/%s", relPath),
				"tags":     tags,
				"preview":  getContentPreview(string(content)),
			}

			noteFiles = append(noteFiles, noteFile)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking notes directory: %w", err)
	}

	filesJSON, _ := json.MarshalIndent(noteFiles, "", "  ")

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "notes://files/",
			MIMEType: "application/json",
			Text:     string(filesJSON),
		},
	}, nil
}

func (ns *NotesServer) ListNoteTemplates(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Get available templates from the builtin templates
	templates := ns.getBuiltinTemplates()

	templatesMap := make(map[string]interface{})
	for name, template := range templates {
		templatesMap[name] = map[string]interface{}{
			"name":        template.Name,
			"description": template.Description,
			"use_case":    template.UseCase,
			"uri":         fmt.Sprintf("notes://templates/%s", name),
			"content":     template.Content,
		}
	}
	templatesJSON, _ := json.MarshalIndent(templatesMap, "", "  ")

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "notes://templates/",
			MIMEType: "application/json",
			Text:     string(templatesJSON),
		},
	}, nil
}

func (ns *NotesServer) ListNoteCollections(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	collections := make(map[string]interface{})

	// Group by folders
	folderMap := make(map[string][]string)
	tagMap := make(map[string][]string)

	err := filepath.Walk(ns.vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(strings.ToLower(info.Name()), ".md") ||
			strings.HasSuffix(strings.ToLower(info.Name()), ".markdown")) {

			relPath, _ := filepath.Rel(ns.vaultDir, path)
			folder := filepath.Dir(relPath)
			if folder == "." {
				folder = "root"
			}

			folderMap[folder] = append(folderMap[folder], relPath)

			// Extract tags
			content, err := os.ReadFile(path)
			if err == nil {
				tags := extractTags(string(content))
				for _, tag := range tags {
					tagMap[tag] = append(tagMap[tag], relPath)
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking notes directory: %w", err)
	}

	// Build collections
	collections["folders"] = folderMap
	collections["tags"] = tagMap

	collectionsJSON, _ := json.MarshalIndent(collections, "", "  ")

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "notes://collections/",
			MIMEType: "application/json",
			Text:     string(collectionsJSON),
		},
	}, nil
}

// Helper functions

func extractTags(content string) []string {
	var tags []string
	lines := strings.Split(content, "\n")

	// Look for tags in frontmatter
	inFrontmatter := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "---" {
			inFrontmatter = !inFrontmatter
			continue
		}

		if inFrontmatter && strings.HasPrefix(line, "tags:") {
			// Parse YAML tags array
			tagLine := strings.TrimPrefix(line, "tags:")
			tagLine = strings.Trim(tagLine, " []")
			if tagLine != "" {
				for _, tag := range strings.Split(tagLine, ",") {
					tag = strings.Trim(tag, " \"'")
					if tag != "" {
						tags = append(tags, tag)
					}
				}
			}
		}
	}

	// Also look for hashtags in content
	for _, line := range lines {
		words := strings.Fields(line)
		for _, word := range words {
			if strings.HasPrefix(word, "#") && len(word) > 1 {
				tag := strings.TrimPrefix(word, "#")
				tag = strings.Trim(tag, ".,!?;:")
				if tag != "" {
					// Avoid duplicates
					found := false
					for _, existing := range tags {
						if existing == tag {
							found = true
							break
						}
					}
					if !found {
						tags = append(tags, tag)
					}
				}
			}
		}
	}

	return tags
}

func getContentPreview(content string) string {
	lines := strings.Split(content, "\n")

	// Skip frontmatter
	startIdx := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				startIdx = i + 1
				break
			}
		}
	}

	// Get first few lines of actual content
	var previewLines []string
	for i := startIdx; i < len(lines) && len(previewLines) < 3; i++ {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			previewLines = append(previewLines, line)
		}
	}

	preview := strings.Join(previewLines, " ")
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}

	return preview
}
