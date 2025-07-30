package notes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestNewNotesServer(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	ns := NewNotesServer(ctx, tempDir)

	if ns == nil {
		t.Fatal("NewNotesServer returned nil")
	}

	if ns.ctx != ctx {
		t.Error("Context not set correctly")
	}

	if ns.vaultDir != tempDir {
		t.Errorf("Vault directory not set correctly: expected %s, got %s", tempDir, ns.vaultDir)
	}

	if ns.McpServer == nil {
		t.Error("MCP server not initialized")
	}
}

func TestReadNote_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test note
	noteContent := "# Test Note\n\nThis is test content."
	notePath := filepath.Join(tempDir, "test.md")
	err := os.WriteFile(notePath, []byte(noteContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test note: %v", err)
	}

	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: tempDir,
	}

	request := mcp.CallToolRequest{}
	params := ReadNoteRequest{Path: "test.md"}

	result, err := ns.ReadNote(ctx, request, params)
	if err != nil {
		t.Fatalf("ReadNote failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("ReadNote returned error: %v", result.Content[0])
	}

	if len(result.Content) == 0 {
		t.Fatal("No content returned from ReadNote")
	}

	// Verify the content
	content := result.Content[0].(mcp.TextContent).Text
	if content != noteContent {
		t.Errorf("Content mismatch.\nExpected: %s\nGot: %s", noteContent, content)
	}
}

func TestReadNote_NonexistentFile(t *testing.T) {
	tempDir := t.TempDir()

	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: tempDir,
	}

	request := mcp.CallToolRequest{}
	params := ReadNoteRequest{Path: "nonexistent.md"}

	result, err := ns.ReadNote(ctx, request, params)
	// The actual implementation returns a Go error, not an MCP error
	if err == nil {
		t.Error("Expected Go error for nonexistent file")
	}

	// If we get a result, it should be an error
	if result != nil && !result.IsError {
		t.Error("If result is returned, it should be an error")
	}
}
func TestWriteNote_Success(t *testing.T) {
	tempDir := t.TempDir()

	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: tempDir,
	}

	request := mcp.CallToolRequest{}
	params := WriteNoteRequest{
		Path:    "new-note.md",
		Content: "# New Note\n\nThis is new content.",
	}

	result, err := ns.WriteNote(ctx, request, params)
	if err != nil {
		t.Fatalf("WriteNote failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("WriteNote returned error: %v", result.Content[0])
	}

	// Verify the file was created
	notePath := filepath.Join(tempDir, "new-note.md")
	content, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("Failed to read created note: %v", err)
	}

	expectedContent := "# New Note\n\nThis is new content."
	if string(content) != expectedContent {
		t.Errorf("File content mismatch.\nExpected: %s\nGot: %s", expectedContent, string(content))
	}
}

func TestWriteNote_CreateDirectory(t *testing.T) {
	tempDir := t.TempDir()

	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: tempDir,
	}

	request := mcp.CallToolRequest{}
	params := WriteNoteRequest{
		Path:    "subdir/nested-note.md",
		Content: "# Nested Note\n\nContent in subdirectory.",
	}

	result, err := ns.WriteNote(ctx, request, params)
	if err != nil {
		t.Fatalf("WriteNote failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("WriteNote returned error: %v", result.Content[0])
	}

	// Verify the directory and file were created
	notePath := filepath.Join(tempDir, "subdir", "nested-note.md")
	content, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("Failed to read created note: %v", err)
	}

	expectedContent := "# Nested Note\n\nContent in subdirectory."
	if string(content) != expectedContent {
		t.Errorf("File content mismatch.\nExpected: %s\nGot: %s", expectedContent, string(content))
	}
}

func TestListNotes_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Create test notes
	testNotes := []string{"note1.md", "note2.md", "subdir/note3.md"}
	for _, noteName := range testNotes {
		notePath := filepath.Join(tempDir, noteName)
		dir := filepath.Dir(notePath)
		if dir != tempDir {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}
		}

		err := os.WriteFile(notePath, []byte("# "+noteName), 0644)
		if err != nil {
			t.Fatalf("Failed to create test note: %v", err)
		}
	}

	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: tempDir,
	}

	request := mcp.CallToolRequest{}
	params := ListNotesRequest{
		Path:      "",
		Recursive: true,
	}

	result, err := ns.ListNotes(ctx, request, params)
	if err != nil {
		t.Fatalf("ListNotes failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("ListNotes returned error: %v", result.Content[0])
	}

	if len(result.Content) == 0 {
		t.Fatal("No content returned from ListNotes")
	}

	// Verify the content is valid JSON
	content := result.Content[0].(mcp.TextContent).Text
	var notes []map[string]interface{}
	err = json.Unmarshal([]byte(content), &notes)
	if err != nil {
		t.Errorf("Invalid JSON returned: %v", err)
	}

	if len(notes) != 3 {
		t.Errorf("Expected 3 notes, got %d", len(notes))
	}
}

func TestSearchNotes_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Create test notes with different content
	testNotes := map[string]string{
		"note1.md": "# Note 1\n\nThis contains the search term.",
		"note2.md": "# Note 2\n\nThis does not contain it.",
		"note3.md": "# Note 3\n\nAnother search term match here.",
	}

	for noteName, content := range testNotes {
		notePath := filepath.Join(tempDir, noteName)
		err := os.WriteFile(notePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test note: %v", err)
		}
	}

	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: tempDir,
	}

	request := mcp.CallToolRequest{}
	params := SearchNotesRequest{
		Query: "search term",
		Path:  "",
	}

	result, err := ns.SearchNotes(ctx, request, params)
	if err != nil {
		t.Fatalf("SearchNotes failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("SearchNotes returned error: %v", result.Content[0])
	}

	if len(result.Content) == 0 {
		t.Fatal("No content returned from SearchNotes")
	}

	// Verify the search results
	content := result.Content[0].(mcp.TextContent).Text
	var searchResults []map[string]interface{}
	err = json.Unmarshal([]byte(content), &searchResults)
	if err != nil {
		t.Errorf("Invalid JSON returned: %v", err)
	}

	// Should find 2 notes with "search term"
	if len(searchResults) != 2 {
		t.Errorf("Expected 2 search results, got %d", len(searchResults))
	}
}

func TestRequestValidation(t *testing.T) {
	// Test various request types for JSON marshaling/unmarshaling

	t.Run("ReadNoteRequest", func(t *testing.T) {
		request := ReadNoteRequest{Path: "test.md"}

		jsonData, err := json.Marshal(request)
		if err != nil {
			t.Errorf("Failed to marshal request: %v", err)
		}

		var unmarshaled ReadNoteRequest
		err = json.Unmarshal(jsonData, &unmarshaled)
		if err != nil {
			t.Errorf("Failed to unmarshal request: %v", err)
		}

		if unmarshaled.Path != request.Path {
			t.Errorf("Path mismatch: expected %s, got %s", request.Path, unmarshaled.Path)
		}
	})

	t.Run("WriteNoteRequest", func(t *testing.T) {
		request := WriteNoteRequest{
			Path:    "test.md",
			Content: "Test content",
		}

		jsonData, err := json.Marshal(request)
		if err != nil {
			t.Errorf("Failed to marshal request: %v", err)
		}

		var unmarshaled WriteNoteRequest
		err = json.Unmarshal(jsonData, &unmarshaled)
		if err != nil {
			t.Errorf("Failed to unmarshal request: %v", err)
		}

		if unmarshaled.Path != request.Path {
			t.Errorf("Path mismatch: expected %s, got %s", request.Path, unmarshaled.Path)
		}

		if unmarshaled.Content != request.Content {
			t.Errorf("Content mismatch: expected %s, got %s", request.Content, unmarshaled.Content)
		}
	})

	t.Run("ListNotesRequest", func(t *testing.T) {
		request := ListNotesRequest{
			Path:      "subdir",
			Recursive: true,
		}

		jsonData, err := json.Marshal(request)
		if err != nil {
			t.Errorf("Failed to marshal request: %v", err)
		}

		var unmarshaled ListNotesRequest
		err = json.Unmarshal(jsonData, &unmarshaled)
		if err != nil {
			t.Errorf("Failed to unmarshal request: %v", err)
		}

		if unmarshaled.Path != request.Path {
			t.Errorf("Path mismatch: expected %s, got %s", request.Path, unmarshaled.Path)
		}

		if unmarshaled.Recursive != request.Recursive {
			t.Errorf("Recursive mismatch: expected %v, got %v", request.Recursive, unmarshaled.Recursive)
		}
	})

	t.Run("SearchNotesRequest", func(t *testing.T) {
		request := SearchNotesRequest{
			Query: "test query",
			Path:  "search/path",
		}

		jsonData, err := json.Marshal(request)
		if err != nil {
			t.Errorf("Failed to marshal request: %v", err)
		}

		var unmarshaled SearchNotesRequest
		err = json.Unmarshal(jsonData, &unmarshaled)
		if err != nil {
			t.Errorf("Failed to unmarshal request: %v", err)
		}

		if unmarshaled.Query != request.Query {
			t.Errorf("Query mismatch: expected %s, got %s", request.Query, unmarshaled.Query)
		}

		if unmarshaled.Path != request.Path {
			t.Errorf("Path mismatch: expected %s, got %s", request.Path, unmarshaled.Path)
		}
	})
}

func TestErrorHandling(t *testing.T) {
	tempDir := t.TempDir()

	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: tempDir,
	}

	tests := []struct {
		name        string
		operation   string
		expectError bool
	}{
		{"Read nonexistent file", "read_nonexistent", true},
		{"Write to invalid path", "write_invalid", true},
		{"List invalid directory", "list_invalid", true},
		{"Search with empty query", "search_empty", false}, // Empty query might be valid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result *mcp.CallToolResult
			var err error

			switch tt.operation {
			case "read_nonexistent":
				result, err = ns.ReadNote(ctx, mcp.CallToolRequest{}, ReadNoteRequest{Path: "nonexistent.md"})
			case "write_invalid":
				// Try to write to a path that would cause an error (e.g., invalid characters)
				result, err = ns.WriteNote(ctx, mcp.CallToolRequest{}, WriteNoteRequest{Path: "", Content: "test"})
			case "list_invalid":
				result, err = ns.ListNotes(ctx, mcp.CallToolRequest{}, ListNotesRequest{Path: "nonexistent/path"})
			case "search_empty":
				result, err = ns.SearchNotes(ctx, mcp.CallToolRequest{}, SearchNotesRequest{Query: "", Path: ""})
			}

			if tt.expectError {
				// The actual implementation returns Go errors for these cases
				if err == nil {
					t.Error("Expected Go error for error case")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected Go error: %v", err)
				}
				if result != nil && result.IsError {
					t.Errorf("Unexpected MCP error: %v", result.Content[0])
				}
			}
		})
	}
}

func TestServerCapabilities(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	ns := NewNotesServer(ctx, tempDir)

	// Verify server is properly initialized
	if ns.McpServer == nil {
		t.Error("MCP server should be initialized")
	}

	// In a real implementation, we'd test:
	// - Tool capabilities are enabled
	// - Resource capabilities are enabled
	// - Correct server name and version
	// - All expected tools are registered
	// - All expected resources are registered
}

func BenchmarkReadNote(b *testing.B) {
	tempDir := b.TempDir()

	// Create a test note
	noteContent := "# Benchmark Note\n\n" + strings.Repeat("This is test content. ", 100)
	notePath := filepath.Join(tempDir, "benchmark.md")
	err := os.WriteFile(notePath, []byte(noteContent), 0644)
	if err != nil {
		b.Fatalf("Failed to create test note: %v", err)
	}

	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: tempDir,
	}

	request := mcp.CallToolRequest{}
	params := ReadNoteRequest{Path: "benchmark.md"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ns.ReadNote(ctx, request, params)
		if err != nil {
			b.Fatalf("ReadNote failed: %v", err)
		}
		if result.IsError {
			b.Fatalf("ReadNote returned error: %v", result.Content[0])
		}
	}
}

func BenchmarkWriteNote(b *testing.B) {
	tempDir := b.TempDir()

	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: tempDir,
	}

	noteContent := "# Benchmark Note\n\n" + strings.Repeat("This is test content. ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		request := mcp.CallToolRequest{}
		params := WriteNoteRequest{
			Path:    fmt.Sprintf("benchmark%d.md", i),
			Content: noteContent,
		}

		result, err := ns.WriteNote(ctx, request, params)
		if err != nil {
			b.Fatalf("WriteNote failed: %v", err)
		}
		if result.IsError {
			b.Fatalf("WriteNote returned error: %v", result.Content[0])
		}
	}
}

func BenchmarkListNotes(b *testing.B) {
	tempDir := b.TempDir()

	// Create many test notes
	for i := 0; i < 100; i++ {
		notePath := filepath.Join(tempDir, fmt.Sprintf("note%d.md", i))
		err := os.WriteFile(notePath, []byte(fmt.Sprintf("# Note %d", i)), 0644)
		if err != nil {
			b.Fatalf("Failed to create test note: %v", err)
		}
	}

	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: tempDir,
	}

	request := mcp.CallToolRequest{}
	params := ListNotesRequest{
		Path:      "",
		Recursive: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ns.ListNotes(ctx, request, params)
		if err != nil {
			b.Fatalf("ListNotes failed: %v", err)
		}
		if result.IsError {
			b.Fatalf("ListNotes returned error: %v", result.Content[0])
		}
	}
}
