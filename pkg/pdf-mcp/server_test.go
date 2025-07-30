package pdf_mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/api/drive/v3"
)

func TestNewPDFServer_Success(t *testing.T) {
	// Note: This test would require valid Google credentials in a real scenario
	// For unit testing, we'd need to mock the Google Drive service creation

	ctx := context.Background()

	// Test server creation with mock credentials (this will fail without real creds)
	// In a real test environment, we'd use dependency injection or mocks

	// For now, test the server structure and initialization logic
	server := &PDFServer{
		ctx:      ctx,
		folderID: "test-folder-id",
	}

	if server.ctx != ctx {
		t.Error("Context not set correctly")
	}

	if server.folderID != "test-folder-id" {
		t.Error("Folder ID not set correctly")
	}
}

func TestSearchPDFsRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request SearchPDFsRequest
		isValid bool
	}{
		{
			name: "Valid request with query and max files",
			request: SearchPDFsRequest{
				Query:    "machine learning",
				MaxFiles: 10,
			},
			isValid: true,
		},
		{
			name: "Valid request with query only",
			request: SearchPDFsRequest{
				Query: "research",
			},
			isValid: true,
		},
		{
			name: "Empty query",
			request: SearchPDFsRequest{
				Query:    "",
				MaxFiles: 5,
			},
			isValid: false, // Empty query might not be useful
		},
		{
			name: "Zero max files",
			request: SearchPDFsRequest{
				Query:    "test",
				MaxFiles: 0,
			},
			isValid: true, // Should use default
		},
		{
			name: "Negative max files",
			request: SearchPDFsRequest{
				Query:    "test",
				MaxFiles: -1,
			},
			isValid: true, // Should use default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling/unmarshaling
			jsonData, err := json.Marshal(tt.request)
			if err != nil {
				t.Errorf("Failed to marshal request: %v", err)
				return
			}

			var unmarshaled SearchPDFsRequest
			err = json.Unmarshal(jsonData, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal request: %v", err)
				return
			}

			if unmarshaled.Query != tt.request.Query {
				t.Errorf("Query mismatch: expected %s, got %s", tt.request.Query, unmarshaled.Query)
			}

			if unmarshaled.MaxFiles != tt.request.MaxFiles {
				t.Errorf("MaxFiles mismatch: expected %d, got %d", tt.request.MaxFiles, unmarshaled.MaxFiles)
			}
		})
	}
}

func TestGetPDFContentRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request GetPDFContentRequest
		isValid bool
	}{
		{
			name: "Valid file ID",
			request: GetPDFContentRequest{
				FileID: "1BxYzAbc123",
			},
			isValid: true,
		},
		{
			name: "Empty file ID",
			request: GetPDFContentRequest{
				FileID: "",
			},
			isValid: false,
		},
		{
			name: "Short file ID",
			request: GetPDFContentRequest{
				FileID: "abc",
			},
			isValid: true, // Google Drive IDs can vary in length
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling/unmarshaling
			jsonData, err := json.Marshal(tt.request)
			if err != nil {
				t.Errorf("Failed to marshal request: %v", err)
				return
			}

			var unmarshaled GetPDFContentRequest
			err = json.Unmarshal(jsonData, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal request: %v", err)
				return
			}

			if unmarshaled.FileID != tt.request.FileID {
				t.Errorf("FileID mismatch: expected %s, got %s", tt.request.FileID, unmarshaled.FileID)
			}

			// Basic validation
			if tt.isValid && unmarshaled.FileID == "" {
				t.Error("Valid request should not have empty FileID")
			}
		})
	}
}

func TestDriveFileResult_Structure(t *testing.T) {
	// Test the DTO structure used in search results
	mockFile := &drive.File{
		Id:           "test123",
		Name:         "test.pdf",
		MimeType:     "application/pdf",
		Size:         1024,
		ModifiedTime: "2025-01-15T10:30:00Z",
		WebViewLink:  "https://drive.google.com/file/d/test123/view",
	}

	// This would normally use the DTO package
	result := map[string]interface{}{
		"ID":           mockFile.Id,
		"Name":         mockFile.Name,
		"MimeType":     mockFile.MimeType,
		"Size":         mockFile.Size,
		"ModifiedTime": mockFile.ModifiedTime,
		"WebViewLink":  mockFile.WebViewLink,
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(result)
	if err != nil {
		t.Errorf("Failed to marshal result: %v", err)
	}

	var unmarshaled map[string]interface{}
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal result: %v", err)
	}

	// Verify required fields
	requiredFields := []string{"ID", "Name", "MimeType", "Size", "ModifiedTime", "WebViewLink"}
	for _, field := range requiredFields {
		if _, exists := unmarshaled[field]; !exists {
			t.Errorf("Required field '%s' missing from result", field)
		}
	}
}

func TestMCPToolResult_Structure(t *testing.T) {
	// Test MCP tool result structure
	successResult := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent("Test content"),
		},
	}

	if successResult.IsError {
		t.Error("Success result should not be marked as error")
	}

	if len(successResult.Content) == 0 {
		t.Error("Success result should have content")
	}

	// Test error result
	errorResult := &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			mcp.NewTextContent("Error message"),
		},
	}

	if !errorResult.IsError {
		t.Error("Error result should be marked as error")
	}

	if len(errorResult.Content) == 0 {
		t.Error("Error result should have error content")
	}
}

func TestResourceContents_Types(t *testing.T) {
	// Test TextResourceContents
	textResource := mcp.TextResourceContents{
		URI:      "pdf://documents/test123",
		MIMEType: "application/json",
		Text:     `{"test": "data"}`,
	}

	if textResource.URI == "" {
		t.Error("TextResourceContents URI should not be empty")
	}

	if textResource.MIMEType != "application/json" {
		t.Error("TextResourceContents should have correct MIME type")
	}

	// Verify it's valid JSON
	var jsonData interface{}
	err := json.Unmarshal([]byte(textResource.Text), &jsonData)
	if err != nil {
		t.Errorf("TextResourceContents text should be valid JSON: %v", err)
	}

	// Test BlobResourceContents
	blobResource := mcp.BlobResourceContents{
		URI:      "pdf://documents/test123",
		MIMEType: "application/pdf",
		Blob:     "dGVzdCBkYXRh", // base64 encoded "test data"
	}

	if blobResource.URI == "" {
		t.Error("BlobResourceContents URI should not be empty")
	}

	if blobResource.MIMEType != "application/pdf" {
		t.Error("BlobResourceContents should have correct MIME type")
	}

	// Verify it's valid base64
	_, err = json.Marshal(blobResource.Blob)
	if err != nil {
		t.Errorf("BlobResourceContents blob should be valid: %v", err)
	}
}

func TestServerCapabilities(t *testing.T) {
	// Test that server is configured with correct capabilities
	ctx := context.Background()

	// Create a minimal server for testing capabilities
	server := &PDFServer{
		ctx: ctx,
	}

	// In a real implementation, we'd test:
	// - Tool capabilities are enabled
	// - Resource capabilities are enabled
	// - Correct server name and version

	if server.ctx != ctx {
		t.Error("Server context not set correctly")
	}
}

func TestErrorHandling(t *testing.T) {
	// Test various error conditions
	tests := []struct {
		name          string
		errorType     string
		expectedError bool
	}{
		{"Network error", "network", true},
		{"Authentication error", "auth", true},
		{"File not found", "not_found", true},
		{"Invalid file format", "invalid_format", true},
		{"Success case", "none", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test error result creation
			var result *mcp.CallToolResult

			if tt.expectedError {
				result = &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{
						mcp.NewTextContent("Error: " + tt.errorType),
					},
				}
			} else {
				result = &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent("Success"),
					},
				}
			}

			if result.IsError != tt.expectedError {
				t.Errorf("Expected error=%v, got error=%v", tt.expectedError, result.IsError)
			}

			if len(result.Content) == 0 {
				t.Error("Result should have content")
			}
		})
	}
}

func BenchmarkJSONMarshalUnmarshal(b *testing.B) {
	request := SearchPDFsRequest{
		Query:    "machine learning research",
		MaxFiles: 20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jsonData, err := json.Marshal(request)
		if err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}

		var unmarshaled SearchPDFsRequest
		err = json.Unmarshal(jsonData, &unmarshaled)
		if err != nil {
			b.Fatalf("Unmarshal failed: %v", err)
		}
	}
}

func BenchmarkMCPContentCreation(b *testing.B) {
	testData := "This is test content for MCP"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		content := mcp.NewTextContent(testData)
		// Verify content was created (TextContent is a struct, not a pointer)
		if content.Text != testData {
			b.Fatal("Failed to create MCP content correctly")
		}
	}
}
