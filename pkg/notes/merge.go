package notes

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/KyleBrandon/sibyl/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"
)

// MergeStrategy defines how content should be merged with existing notes
type MergeStrategy string

const (
	MergeAppend      MergeStrategy = "append"       // Add to end of file
	MergePrepend     MergeStrategy = "prepend"      // Add to beginning of file
	MergeDateSection MergeStrategy = "date_section" // Add as new dated section
	MergeTopicMerge  MergeStrategy = "topic_merge"  // Intelligent topic-based merging
	MergeReplace     MergeStrategy = "replace"      // Replace entire file
)

// MergeNoteRequest represents a request to merge content with an existing note
type MergeNoteRequest struct {
	Path     string        `json:"path" mcp:"Path to the note file"`
	Content  string        `json:"content" mcp:"Content to merge"`
	Strategy MergeStrategy `json:"strategy,omitempty" mcp:"Merge strategy: append, prepend, date_section, topic_merge, replace"`
	Title    string        `json:"title,omitempty" mcp:"Title for the new section (used with date_section)"`
}

// PreviewMergeRequest represents a request to preview a merge operation
type PreviewMergeRequest struct {
	Path     string        `json:"path" mcp:"Path to the note file"`
	Content  string        `json:"content" mcp:"Content to merge"`
	Strategy MergeStrategy `json:"strategy,omitempty" mcp:"Merge strategy to preview"`
}

// MergeResult represents the result of a merge operation
type MergeResult struct {
	Success      bool   `json:"success"`
	Path         string `json:"path"`
	Strategy     string `json:"strategy"`
	BytesWritten int    `json:"bytes_written"`
	Message      string `json:"message"`
}

// MergePreview represents a preview of what a merge would look like
type MergePreview struct {
	Path           string `json:"path"`
	Strategy       string `json:"strategy"`
	PreviewContent string `json:"preview_content"`
	OriginalLength int    `json:"original_length"`
	NewLength      int    `json:"new_length"`
	WouldOverwrite bool   `json:"would_overwrite"`
}

func (ns *NotesServer) NewMergeNoteTool() {
	tool := mcp.NewTool(
		"merge_note",
		mcp.WithDescription("Merge content with an existing note using various strategies"),
		mcp.WithString("path", mcp.Description("Path to the note file"), mcp.Required()),
		mcp.WithString("content", mcp.Description("Content to merge"), mcp.Required()),
		mcp.WithString("strategy", mcp.Description("Merge strategy: append, prepend, date_section, topic_merge, replace")),
		mcp.WithString("title", mcp.Description("Title for the new section (used with date_section)")),
	)

	ns.McpServer.AddTool(tool, mcp.NewTypedToolHandler(ns.MergeNote))
}

func (ns *NotesServer) NewPreviewMergeTool() {
	tool := mcp.NewTool(
		"preview_merge",
		mcp.WithDescription("Preview what a merge operation would look like without executing it"),
		mcp.WithString("path", mcp.Description("Path to the note file"), mcp.Required()),
		mcp.WithString("content", mcp.Description("Content to merge"), mcp.Required()),
		mcp.WithString("strategy", mcp.Description("Merge strategy to preview")),
	)

	ns.McpServer.AddTool(tool, mcp.NewTypedToolHandler(ns.PreviewMerge))
}

func (ns *NotesServer) MergeNote(ctx context.Context, req mcp.CallToolRequest, params MergeNoteRequest) (*mcp.CallToolResult, error) {
	// Default strategy
	if params.Strategy == "" {
		params.Strategy = MergeAppend
	}

	fullPath, err := utils.ValidatePath(ns.vaultDir, params.Path)
	if err != nil {
		slog.Error("Failed to validate path", "path", params.Path, "error", err)
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Invalid path: %v", err)),
			},
		}, nil
	}

	// Read existing content if file exists
	var existingContent string
	if _, err := os.Stat(fullPath); err == nil {
		content, err := utils.ReadFile(fullPath)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Failed to read existing file: %v", err)),
				},
			}, nil
		}
		existingContent = string(content)
	}

	// Perform merge based on strategy
	mergedContent, err := ns.performMerge(existingContent, params.Content, params.Strategy, params.Title)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Merge failed: %v", err)),
			},
		}, nil
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Failed to create directory: %v", err)),
			},
		}, nil
	}

	// Write merged content
	if err := utils.WriteFile(fullPath, []byte(mergedContent), 0644); err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Failed to write file: %v", err)),
			},
		}, nil
	}

	result := MergeResult{
		Success:      true,
		Path:         params.Path,
		Strategy:     string(params.Strategy),
		BytesWritten: len(mergedContent),
		Message:      fmt.Sprintf("Successfully merged content using %s strategy", params.Strategy),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(resultJSON)),
		},
	}, nil
}

func (ns *NotesServer) PreviewMerge(ctx context.Context, req mcp.CallToolRequest, params PreviewMergeRequest) (*mcp.CallToolResult, error) {
	// Default strategy
	if params.Strategy == "" {
		params.Strategy = MergeAppend
	}

	fullPath, err := utils.ValidatePath(ns.vaultDir, params.Path)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Invalid path: %v", err)),
			},
		}, nil
	}

	// Read existing content if file exists
	var existingContent string
	fileExists := false
	if _, err := os.Stat(fullPath); err == nil {
		content, err := utils.ReadFile(fullPath)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Failed to read existing file: %v", err)),
				},
			}, nil
		}
		existingContent = string(content)
		fileExists = true
	}

	// Perform merge preview
	mergedContent, err := ns.performMerge(existingContent, params.Content, params.Strategy, "")
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Preview failed: %v", err)),
			},
		}, nil
	}

	preview := MergePreview{
		Path:           params.Path,
		Strategy:       string(params.Strategy),
		PreviewContent: mergedContent,
		OriginalLength: len(existingContent),
		NewLength:      len(mergedContent),
		WouldOverwrite: fileExists && params.Strategy == MergeReplace,
	}

	previewJSON, _ := json.MarshalIndent(preview, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(previewJSON)),
		},
	}, nil
}

func (ns *NotesServer) performMerge(existing, newContent string, strategy MergeStrategy, title string) (string, error) {
	switch strategy {
	case MergeAppend:
		if existing == "" {
			return newContent, nil
		}
		return existing + "\n\n" + newContent, nil

	case MergePrepend:
		if existing == "" {
			return newContent, nil
		}
		return newContent + "\n\n" + existing, nil

	case MergeDateSection:
		sectionTitle := title
		if sectionTitle == "" {
			sectionTitle = "Added Content"
		}

		dateHeader := fmt.Sprintf("## %s - %s\n\n", sectionTitle, time.Now().Format("2006-01-02 15:04"))
		sectionContent := dateHeader + newContent

		if existing == "" {
			return sectionContent, nil
		}
		return existing + "\n\n---\n\n" + sectionContent, nil

	case MergeTopicMerge:
		// Simple topic-based merging - could be enhanced with more sophisticated logic
		if existing == "" {
			return newContent, nil
		}

		// Add as a new section with topic detection
		topicHeader := "## New Content\n\n"
		if strings.Contains(strings.ToLower(newContent), "meeting") {
			topicHeader = "## Meeting Notes\n\n"
		} else if strings.Contains(strings.ToLower(newContent), "research") {
			topicHeader = "## Research Notes\n\n"
		} else if strings.Contains(strings.ToLower(newContent), "todo") || strings.Contains(strings.ToLower(newContent), "task") {
			topicHeader = "## Tasks\n\n"
		}

		return existing + "\n\n---\n\n" + topicHeader + newContent, nil

	case MergeReplace:
		return newContent, nil

	default:
		return "", fmt.Errorf("unknown merge strategy: %s", strategy)
	}
}
