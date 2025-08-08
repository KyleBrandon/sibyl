// Package testutils provides common utilities and helpers for testing
package testutils

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/KyleBrandon/sibyl/pkg/notes"
	"github.com/KyleBrandon/sibyl/pkg/pdfmcp"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestVault represents a test notes vault structure
type TestVault struct {
	Dir   string
	Files map[string]string
}

// CreateTestVault creates a temporary vault with test files
func CreateTestVault(t *testing.T, files map[string]string) *TestVault {
	t.Helper()

	tempDir := t.TempDir()

	for path, content := range files {
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

	return &TestVault{
		Dir:   tempDir,
		Files: files,
	}
}

// SetupNotesServer creates a notes server with a test vault
func SetupNotesServer(t *testing.T, vault *TestVault) *notes.NotesServer {
	t.Helper()

	ctx := context.Background()
	return notes.NewNotesServer(ctx, vault.Dir)
}

// SetupPDFServer creates a PDF server for testing (with mock credentials)
func SetupPDFServer(t *testing.T) *pdfmcp.PDFServer {
	t.Helper()

	// Note: This would require mock credentials in a real test
	// For now, just return a basic server structure
	ctx := context.Background()

	// In a real implementation, we'd use dependency injection or mocks
	// to avoid requiring actual Google credentials
	server := &pdfmcp.PDFServer{}
	_ = ctx // Use context to avoid unused variable warning

	return server
}

// AssertNoteContent verifies that a note has expected content
func AssertNoteContent(t *testing.T, ns *notes.NotesServer, path, expectedContent string) {
	t.Helper()

	ctx := context.Background()
	result, err := ns.ReadNote(ctx, mcp.CallToolRequest{}, notes.ReadNoteRequest{Path: path})
	if err != nil {
		t.Fatalf("Failed to read note %s: %v", path, err)
	}

	if result.IsError {
		t.Fatalf("ReadNote returned error for %s: %v", path, result.Content[0])
	}

	if len(result.Content) == 0 {
		t.Fatalf("No content returned for note %s", path)
	}

	actualContent := result.Content[0].(mcp.TextContent).Text
	if actualContent != expectedContent {
		t.Errorf("Content mismatch for %s.\nExpected:\n%s\n\nActual:\n%s", path, expectedContent, actualContent)
	}
}

// AssertNoteExists verifies that a note exists
func AssertNoteExists(t *testing.T, ns *notes.NotesServer, path string) {
	t.Helper()

	ctx := context.Background()
	result, err := ns.ReadNote(ctx, mcp.CallToolRequest{}, notes.ReadNoteRequest{Path: path})
	if err != nil {
		t.Fatalf("Failed to read note %s: %v", path, err)
	}

	if result.IsError {
		t.Fatalf("Note %s should exist but got error: %v", path, result.Content[0])
	}
}

// AssertNoteNotExists verifies that a note does not exist
func AssertNoteNotExists(t *testing.T, ns *notes.NotesServer, path string) {
	t.Helper()

	ctx := context.Background()
	result, err := ns.ReadNote(ctx, mcp.CallToolRequest{}, notes.ReadNoteRequest{Path: path})
	if err != nil {
		t.Fatalf("ReadNote should not return Go error: %v", err)
	}

	if !result.IsError {
		t.Fatalf("Note %s should not exist but was found", path)
	}
}

// AssertResourceStructure verifies that a resource has the expected structure
func AssertResourceStructure(t *testing.T, resource mcp.ResourceContents, expectedFields []string) {
	t.Helper()

	textResource, ok := resource.(mcp.TextResourceContents)
	if !ok {
		t.Fatal("Resource should be TextResourceContents")
	}

	var data map[string]interface{}
	err := json.Unmarshal([]byte(textResource.Text), &data)
	if err != nil {
		t.Fatalf("Resource should contain valid JSON: %v", err)
	}

	for _, field := range expectedFields {
		if _, exists := data[field]; !exists {
			t.Errorf("Resource should contain field '%s'", field)
		}
	}
}

// AssertJSONStructure verifies that JSON content has expected structure
func AssertJSONStructure(t *testing.T, jsonContent string, expectedFields []string) {
	t.Helper()

	var data interface{}
	err := json.Unmarshal([]byte(jsonContent), &data)
	if err != nil {
		t.Fatalf("Content should be valid JSON: %v", err)
	}

	// Handle both objects and arrays
	switch v := data.(type) {
	case map[string]interface{}:
		for _, field := range expectedFields {
			if _, exists := v[field]; !exists {
				t.Errorf("JSON should contain field '%s'", field)
			}
		}
	case []interface{}:
		if len(v) == 0 {
			t.Error("JSON array should not be empty")
			return
		}

		// Check first item structure
		if firstItem, ok := v[0].(map[string]interface{}); ok {
			for _, field := range expectedFields {
				if _, exists := firstItem[field]; !exists {
					t.Errorf("JSON array items should contain field '%s'", field)
				}
			}
		}
	default:
		t.Errorf("Unexpected JSON structure type: %T", data)
	}
}

// AssertMCPResult verifies that an MCP result is successful
func AssertMCPResult(t *testing.T, result *mcp.CallToolResult, operation string) {
	t.Helper()

	if result == nil {
		t.Fatalf("%s should return a result", operation)
	}

	if result.IsError {
		t.Fatalf("%s should not return error: %v", operation, result.Content[0])
	}

	if len(result.Content) == 0 {
		t.Fatalf("%s should return content", operation)
	}
}

// AssertMCPError verifies that an MCP result is an error
func AssertMCPError(t *testing.T, result *mcp.CallToolResult, operation string) {
	t.Helper()

	if result == nil {
		t.Fatalf("%s should return a result", operation)
	}

	if !result.IsError {
		t.Fatalf("%s should return error but got success", operation)
	}
}

// CreateTestNotes creates a standard set of test notes
func CreateTestNotes() map[string]string {
	return map[string]string{
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
Testing the integration between different components.`,

		"projects/sibyl.md": `---
tags: [project, development, mcp]
---

# Sibyl MCP Project

## Overview
Sibyl provides MCP servers for PDF processing and note management.

## Components
- PDF Server: Handles PDF conversion and processing
- Notes Server: Advanced note management with merge capabilities`,

		"meetings/standup.md": `# Team Standup

**Date:** 2025-01-15
**Attendees:** Alice, Bob, Charlie

## Discussion
- Reviewed progress on MCP servers
- Discussed testing strategy

## Action Items
- [ ] Complete integration tests - Alice
- [ ] Update documentation - Bob`,

		"research/ai-notes.md": `# AI Research Notes

## Key Concepts
- Machine Learning fundamentals
- Neural network architectures
- Training methodologies

## References
- Paper 1: "Deep Learning Foundations"
- Paper 2: "Advanced Neural Networks"`,
	}
}

// CreateLargeTestVault creates a vault with many files for performance testing
func CreateLargeTestVault(t *testing.T, fileCount int) *TestVault {
	t.Helper()

	files := make(map[string]string)

	for i := 0; i < fileCount; i++ {
		filename := filepath.Join("batch", "note"+string(rune(i))+".md")
		content := `---
tags: [test, batch, performance]
---

# Test Note ` + string(rune(i)) + `

This is test content for performance testing.
It contains multiple paragraphs and various elements.

## Section 1
Content here with #hashtag.

## Section 2
More content for testing search and indexing performance.`

		files[filename] = content
	}

	return CreateTestVault(t, files)
}

// BenchmarkHelper provides utilities for benchmark tests
type BenchmarkHelper struct {
	vault *TestVault
	ns    *notes.NotesServer
}

// NewBenchmarkHelper creates a new benchmark helper
func NewBenchmarkHelper(b *testing.B, fileCount int) *BenchmarkHelper {
	b.Helper()

	vault := CreateLargeTestVault(&testing.T{}, fileCount)
	ns := SetupNotesServer(&testing.T{}, vault)

	return &BenchmarkHelper{
		vault: vault,
		ns:    ns,
	}
}

// GetNotesServer returns the notes server for benchmarking
func (bh *BenchmarkHelper) GetNotesServer() *notes.NotesServer {
	return bh.ns
}

// GetVaultDir returns the vault directory path
func (bh *BenchmarkHelper) GetVaultDir() string {
	return bh.vault.Dir
}

// MockPDFData creates mock PDF data for testing
func MockPDFData() []byte {
	// Minimal PDF structure for testing
	return []byte(`%PDF-1.4
1 0 obj
<<
/Type /Catalog
/Pages 2 0 R
>>
endobj

2 0 obj
<<
/Type /Pages
/Kids [3 0 R]
/Count 1
>>
endobj

3 0 obj
<<
/Type /Page
/Parent 2 0 R
/MediaBox [0 0 612 792]
>>
endobj

xref
0 4
0000000000 65535 f 
0000000009 00000 n 
0000000058 00000 n 
0000000115 00000 n 
trailer
<<
/Size 4
/Root 1 0 R
>>
startxref
189
%%EOF`)
}

// CompareJSON compares two JSON strings for equality
func CompareJSON(t *testing.T, expected, actual string) {
	t.Helper()

	var expectedData, actualData interface{}

	err := json.Unmarshal([]byte(expected), &expectedData)
	if err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %v", err)
	}

	err = json.Unmarshal([]byte(actual), &actualData)
	if err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %v", err)
	}

	expectedJSON, _ := json.MarshalIndent(expectedData, "", "  ")
	actualJSON, _ := json.MarshalIndent(actualData, "", "  ")

	if string(expectedJSON) != string(actualJSON) {
		t.Errorf("JSON mismatch.\nExpected:\n%s\n\nActual:\n%s", expectedJSON, actualJSON)
	}
}
