package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KyleBrandon/sibyl/pkg/notes"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestNotesServerIntegration(t *testing.T) {
	// Create temporary vault for testing
	tempDir := t.TempDir()

	// Create test vault structure
	testVault := map[string]string{
		"daily.md": `---
tags: [daily, productivity]
---

# Daily Note - 2025-01-15

## Today's Focus
- Work on MCP servers
- Write comprehensive tests

## Tasks
- [x] Implement PDF server
- [x] Implement Notes server
- [ ] Add integration tests

## Notes
Testing the integration between different components.

## Reflections
The MCP architecture is working well.`,

		"projects/sibyl.md": `---
tags: [project, development, mcp]
---

# Sibyl MCP Project

## Overview
Sibyl provides MCP servers for PDF processing and note management.

## Components
- PDF Server: Handles PDF conversion and processing
- Notes Server: Advanced note management with merge capabilities

## Status
- PDF Server: ✅ Complete
- Notes Server: ✅ Complete
- Integration Tests: 🚧 In Progress`,

		"meetings/standup-2025-01-15.md": `# Team Standup - 2025-01-15

**Date:** 2025-01-15
**Attendees:** Alice, Bob, Charlie
**Duration:** 30 minutes

## Discussion
- Reviewed progress on MCP servers
- Discussed testing strategy
- Planned next sprint

## Decisions Made
- Implement comprehensive test coverage
- Add integration tests for full workflows

## Action Items
- [ ] Complete integration tests - Alice - 2025-01-16
- [ ] Update documentation - Bob - 2025-01-17
- [ ] Performance testing - Charlie - 2025-01-18`,
	}

	// Create vault structure
	for path, content := range testVault {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if dir != tempDir {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				t.Fatalf("Failed to create directory %s: %v", dir, err)
			}
		}

		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", path, err)
		}
	}

	// Initialize notes server
	ctx := context.Background()
	ns := notes.NewNotesServer(ctx, tempDir)

	t.Run("Full Workflow Test", func(t *testing.T) {
		// Test 1: List all notes
		listResult, err := ns.ListNotes(ctx, mcp.CallToolRequest{}, notes.ListNotesRequest{
			Path:      "",
			Recursive: true,
		})
		if err != nil {
			t.Fatalf("ListNotes failed: %v", err)
		}

		if listResult.IsError {
			t.Fatalf("ListNotes returned error: %v", listResult.Content[0])
		}

		// Verify we found all 3 notes
		var notesList []map[string]interface{}
		err = json.Unmarshal([]byte(listResult.Content[0].(mcp.TextContent).Text), &notesList)
		if err != nil {
			t.Fatalf("Failed to parse notes list: %v", err)
		}

		if len(notesList) != 3 {
			t.Errorf("Expected 3 notes, got %d", len(notesList))
		}

		// Test 2: Search for specific content
		searchResult, err := ns.SearchNotes(ctx, mcp.CallToolRequest{}, notes.SearchNotesRequest{
			Query: "MCP servers",
			Path:  "",
		})
		if err != nil {
			t.Fatalf("SearchNotes failed: %v", err)
		}

		if searchResult.IsError {
			t.Fatalf("SearchNotes returned error: %v", searchResult.Content[0])
		}

		// Should find at least 2 notes mentioning "MCP servers"
		var searchResults []map[string]interface{}
		err = json.Unmarshal([]byte(searchResult.Content[0].(mcp.TextContent).Text), &searchResults)
		if err != nil {
			t.Fatalf("Failed to parse search results: %v", err)
		}

		if len(searchResults) < 2 {
			t.Errorf("Expected at least 2 search results, got %d", len(searchResults))
		}

		// Test 3: Create note from template
		templateResult, err := ns.CreateNoteFromTemplate(ctx, mcp.CallToolRequest{}, notes.CreateFromTemplateRequest{
			Path:         "new-meeting.md",
			TemplateType: "meeting",
			Variables: map[string]string{
				"TITLE":     "Integration Test Meeting",
				"ATTENDEES": "Test Team",
				"DURATION":  "15 minutes",
			},
		})
		if err != nil {
			t.Fatalf("CreateNoteFromTemplate failed: %v", err)
		}

		if templateResult.IsError {
			t.Fatalf("CreateNoteFromTemplate returned error: %v", templateResult.Content[0])
		}

		// Verify the note was created
		readResult, err := ns.ReadNote(ctx, mcp.CallToolRequest{}, notes.ReadNoteRequest{
			Path: "new-meeting.md",
		})
		if err != nil {
			t.Fatalf("ReadNote failed: %v", err)
		}

		if readResult.IsError {
			t.Fatalf("ReadNote returned error: %v", readResult.Content[0])
		}

		noteContent := readResult.Content[0].(mcp.TextContent).Text
		if !contains(noteContent, "Integration Test Meeting") {
			t.Error("Created note should contain template variable substitution")
		}

		// Test 4: Merge content with existing note
		mergeResult, err := ns.MergeNote(ctx, mcp.CallToolRequest{}, notes.MergeNoteRequest{
			Path:     "daily.md",
			Content:  "## Integration Test Results\n\nAll tests passing successfully!",
			Strategy: "append",
		})
		if err != nil {
			t.Fatalf("MergeNote failed: %v", err)
		}

		if mergeResult.IsError {
			t.Fatalf("MergeNote returned error: %v", mergeResult.Content[0])
		}

		// Verify the merge worked
		updatedResult, err := ns.ReadNote(ctx, mcp.CallToolRequest{}, notes.ReadNoteRequest{
			Path: "daily.md",
		})
		if err != nil {
			t.Fatalf("ReadNote after merge failed: %v", err)
		}

		updatedContent := updatedResult.Content[0].(mcp.TextContent).Text
		if !contains(updatedContent, "Integration Test Results") {
			t.Error("Merged content should be present in the note")
		}

		// Test 5: Preview merge before applying
		previewResult, err := ns.PreviewMerge(ctx, mcp.CallToolRequest{}, notes.PreviewMergeRequest{
			Path:     "projects/sibyl.md",
			Content:  "## Testing\n\nComprehensive test suite implemented.",
			Strategy: "topic_merge",
		})
		if err != nil {
			t.Fatalf("PreviewMerge failed: %v", err)
		}

		if previewResult.IsError {
			t.Fatalf("PreviewMerge returned error: %v", previewResult.Content[0])
		}

		// Verify preview contains expected content
		previewContent := previewResult.Content[0].(mcp.TextContent).Text

		// The preview should contain both existing and new content
		if !strings.Contains(previewContent, "Testing") {
			t.Error("Preview should contain new content")
		}
	})

	t.Run("Resource Integration Test", func(t *testing.T) {
		// Test resource discovery and navigation

		// Test 1: List all note files
		filesResource, err := ns.ListNoteFiles(ctx, mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "notes://files/"},
		})
		if err != nil {
			t.Fatalf("ListNoteFiles failed: %v", err)
		}

		var noteFiles []map[string]interface{}
		err = json.Unmarshal([]byte(filesResource[0].(mcp.TextResourceContents).Text), &noteFiles)
		if err != nil {
			t.Fatalf("Failed to parse note files: %v", err)
		}

		// Should include the new note we created
		if len(noteFiles) != 4 { // 3 original + 1 created
			t.Errorf("Expected 4 note files, got %d", len(noteFiles))
		}

		// Test 2: List note collections
		collectionsResource, err := ns.ListNoteCollections(ctx, mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "notes://collections/"},
		})
		if err != nil {
			t.Fatalf("ListNoteCollections failed: %v", err)
		}

		var collections map[string]interface{}
		err = json.Unmarshal([]byte(collectionsResource[0].(mcp.TextResourceContents).Text), &collections)
		if err != nil {
			t.Fatalf("Failed to parse collections: %v", err)
		}

		// Verify folder structure
		folders, ok := collections["folders"].(map[string]interface{})
		if !ok {
			t.Fatal("Collections should contain folders")
		}

		expectedFolders := []string{"root", "projects", "meetings"}
		for _, folder := range expectedFolders {
			if _, exists := folders[folder]; !exists {
				t.Errorf("Expected folder '%s' not found", folder)
			}
		}

		// Verify tag extraction
		tags, ok := collections["tags"].(map[string]interface{})
		if !ok {
			t.Fatal("Collections should contain tags")
		}

		expectedTags := []string{"daily", "productivity", "project", "development", "mcp"}
		foundTags := 0
		for _, tag := range expectedTags {
			if _, exists := tags[tag]; exists {
				foundTags++
			}
		}

		if foundTags < 3 {
			t.Errorf("Expected to find at least 3 tags, found %d", foundTags)
		}

		// Test 3: List templates
		templatesResource, err := ns.ListNoteTemplates(ctx, mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "notes://templates/"},
		})
		if err != nil {
			t.Fatalf("ListNoteTemplates failed: %v", err)
		}

		var templates map[string]interface{}
		err = json.Unmarshal([]byte(templatesResource[0].(mcp.TextResourceContents).Text), &templates)
		if err != nil {
			t.Fatalf("Failed to parse templates: %v", err)
		}

		expectedTemplates := []string{"daily", "meeting", "research", "project"}
		for _, template := range expectedTemplates {
			if _, exists := templates[template]; !exists {
				t.Errorf("Expected template '%s' not found", template)
			}
		}
	})

	t.Run("Error Handling Integration", func(t *testing.T) {
		// Test error scenarios in integrated workflows

		// Test 1: Try to merge with nonexistent file (should succeed by creating new file)
		mergeResult, err := ns.MergeNote(ctx, mcp.CallToolRequest{}, notes.MergeNoteRequest{
			Path:     "nonexistent.md",
			Content:  "Test content",
			Strategy: "append",
		})

		// The actual implementation creates the file if it doesn't exist
		if err != nil {
			t.Errorf("Unexpected error for nonexistent file merge: %v", err)
		}
		if mergeResult == nil || mergeResult.IsError {
			t.Error("Expected successful merge for nonexistent file (should create new file)")
		}
		// Test 2: Try to create note from invalid template
		templateResult, err := ns.CreateNoteFromTemplate(ctx, mcp.CallToolRequest{}, notes.CreateFromTemplateRequest{
			Path:         "invalid-template-note.md",
			TemplateType: "nonexistent",
		})

		// The actual implementation returns an MCP error for invalid templates
		if err != nil {
			t.Errorf("Unexpected Go error for invalid template: %v", err)
		}
		if templateResult == nil || !templateResult.IsError {
			t.Error("Expected MCP error for invalid template")
		}
		// Test 3: Search with empty query
		_, err = ns.SearchNotes(ctx, mcp.CallToolRequest{}, notes.SearchNotesRequest{
			Query: "",
			Path:  "",
		})
		if err != nil {
			t.Fatalf("SearchNotes should not return Go error: %v", err)
		}

		// Empty query might be valid (return all notes) or invalid - depends on implementation
		// Just verify it doesn't crash
	})
}

func TestNotesServerPerformance(t *testing.T) {
	// Create a large vault for performance testing
	tempDir := t.TempDir()

	// Create 100 test notes
	for i := 0; i < 100; i++ {
		content := `---
tags: [test, performance, batch` + string(rune(i%10)) + `]
---

# Test Note ` + string(rune(i)) + `

This is test content for performance testing.
It contains multiple paragraphs and various elements.

## Section 1
Content here with #hashtag` + string(rune(i%5)) + `.

## Section 2
More content for testing search and indexing performance.`

		filename := fmt.Sprintf("note%d.md", i)
		if i%10 == 0 {
			// Create some in subdirectories
			subdir := fmt.Sprintf("batch%d", i/10)
			err := os.MkdirAll(filepath.Join(tempDir, subdir), 0755)
			if err != nil {
				t.Fatalf("Failed to create subdirectory: %v", err)
			}
			filename = filepath.Join(subdir, filename)
		}

		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test note: %v", err)
		}
	}

	ctx := context.Background()
	ns := notes.NewNotesServer(ctx, tempDir)

	t.Run("List Performance", func(t *testing.T) {
		// Test listing performance with many files
		result, err := ns.ListNotes(ctx, mcp.CallToolRequest{}, notes.ListNotesRequest{
			Path:      "",
			Recursive: true,
		})
		if err != nil {
			t.Fatalf("ListNotes failed: %v", err)
		}

		if result.IsError {
			t.Fatalf("ListNotes returned error: %v", result.Content[0])
		}

		var notesList []map[string]interface{}
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &notesList)
		if err != nil {
			t.Fatalf("Failed to parse notes list: %v", err)
		}

		if len(notesList) != 100 {
			t.Errorf("Expected 100 notes, got %d", len(notesList))
		}
	})

	t.Run("Search Performance", func(t *testing.T) {
		// Test search performance across many files
		result, err := ns.SearchNotes(ctx, mcp.CallToolRequest{}, notes.SearchNotesRequest{
			Query: "performance testing",
			Path:  "",
		})
		if err != nil {
			t.Fatalf("SearchNotes failed: %v", err)
		}

		if result.IsError {
			t.Fatalf("SearchNotes returned error: %v", result.Content[0])
		}

		var searchResults []map[string]interface{}
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &searchResults)
		if err != nil {
			t.Fatalf("Failed to parse search results: %v", err)
		}

		// Should find all 100 notes (they all contain "performance testing")
		if len(searchResults) != 100 {
			t.Errorf("Expected 100 search results, got %d", len(searchResults))
		}
	})

	t.Run("Resource Performance", func(t *testing.T) {
		// Test resource listing performance
		result, err := ns.ListNoteFiles(ctx, mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "notes://files/"},
		})
		if err != nil {
			t.Fatalf("ListNoteFiles failed: %v", err)
		}

		var noteFiles []map[string]interface{}
		err = json.Unmarshal([]byte(result[0].(mcp.TextResourceContents).Text), &noteFiles)
		if err != nil {
			t.Fatalf("Failed to parse note files: %v", err)
		}

		if len(noteFiles) != 100 {
			t.Errorf("Expected 100 note files, got %d", len(noteFiles))
		}

		// Verify tag extraction worked for all files
		totalTags := 0
		for _, noteFile := range noteFiles {
			tags, ok := noteFile["tags"].([]interface{})
			if ok {
				totalTags += len(tags)
			}
		}

		if totalTags == 0 {
			t.Error("No tags extracted from any files")
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsAt(s, substr, 1))))
}

func containsAt(s, substr string, start int) bool {
	if start >= len(s) {
		return false
	}
	if start+len(substr) > len(s) {
		return containsAt(s, substr, start+1)
	}
	if s[start:start+len(substr)] == substr {
		return true
	}
	return containsAt(s, substr, start+1)
}
