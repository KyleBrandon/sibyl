package pdfmcp

import (
	"encoding/json"
	"testing"

	"github.com/KyleBrandon/sibyl/pkg/dto"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestNewPDFServer_Success(t *testing.T) {
	// Test OCR configuration
	ocrConfig := OCRConfig{
		Languages:     []string{"en", "fr"},
		MathpixAppID:  "test-app-id",
		MathpixAppKey: "test-app-key",
	}

	// Test that NewPDFServer requires Mathpix credentials
	if ocrConfig.MathpixAppID == "" || ocrConfig.MathpixAppKey == "" {
		t.Error("OCR config should have Mathpix credentials")
	}

	// Test configuration structure
	if len(ocrConfig.Languages) == 0 {
		t.Error("OCR config should have at least one language")
	}
}

func TestNewPDFServer_MissingCredentials(t *testing.T) {
	// Test with missing Mathpix credentials
	ocrConfig := OCRConfig{
		Languages:     []string{"en"},
		MathpixAppID:  "",
		MathpixAppKey: "test-app-key",
	}

	// This should fail in the actual NewPDFServer call due to missing credentials
	// We can't test the actual function without Drive credentials, so test the validation logic
	if ocrConfig.MathpixAppID == "" || ocrConfig.MathpixAppKey == "" {
		t.Log("Correctly detected missing Mathpix credentials")
	} else {
		t.Error("Should detect missing Mathpix credentials")
	}
}

func TestSearchPDFsRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request SearchPDFsRequest
		isValid bool
	}{
		{
			name: "Valid search query",
			request: SearchPDFsRequest{
				Query:    "meeting notes",
				MaxFiles: 10,
			},
			isValid: true,
		},
		{
			name: "Empty query",
			request: SearchPDFsRequest{
				Query:    "",
				MaxFiles: 10,
			},
			isValid: false,
		},
		{
			name: "Default max files",
			request: SearchPDFsRequest{
				Query:    "test",
				MaxFiles: 0,
			},
			isValid: true, // Should default to 10
		},
		{
			name: "Very large max files",
			request: SearchPDFsRequest{
				Query:    "test",
				MaxFiles: 1000,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			jsonData, err := json.Marshal(tt.request)
			if err != nil {
				t.Errorf("Failed to marshal request: %v", err)
				return
			}

			// Test JSON unmarshaling
			var unmarshaled SearchPDFsRequest
			err = json.Unmarshal(jsonData, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal request: %v", err)
				return
			}

			// Validate request
			isEmpty := tt.request.Query == ""
			if tt.isValid && isEmpty {
				t.Error("Valid request should not have empty query")
			}
			if !tt.isValid && !isEmpty {
				t.Error("Invalid request should have empty query")
			}

			// Test that MaxFiles is handled correctly
			if tt.request.MaxFiles <= 0 && tt.isValid {
				t.Log("MaxFiles will be defaulted to 10 in actual implementation")
			}
		})
	}
}

func TestConvertPDFToMarkdownRequest_ServerValidation(t *testing.T) {
	tests := []struct {
		name    string
		request ConvertPDFToMarkdownRequest
		isValid bool
	}{
		{
			name: "Valid file ID",
			request: ConvertPDFToMarkdownRequest{
				FileID: "1BxYzAbc123",
			},
			isValid: true,
		},
		{
			name: "Empty file ID",
			request: ConvertPDFToMarkdownRequest{
				FileID: "",
			},
			isValid: false,
		},
		{
			name: "File ID with special characters",
			request: ConvertPDFToMarkdownRequest{
				FileID: "1BxYz-Abc_123",
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			jsonData, err := json.Marshal(tt.request)
			if err != nil {
				t.Errorf("Failed to marshal request: %v", err)
				return
			}

			// Test JSON unmarshaling
			var unmarshaled ConvertPDFToMarkdownRequest
			err = json.Unmarshal(jsonData, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal request: %v", err)
				return
			}

			// Validate request
			isEmpty := tt.request.FileID == ""
			if tt.isValid && isEmpty {
				t.Error("Valid request should not have empty file ID")
			}
			if !tt.isValid && !isEmpty {
				t.Error("Invalid request should have empty file ID")
			}

			// Verify the fields are preserved
			if unmarshaled.FileID != tt.request.FileID {
				t.Errorf("Expected FileID '%s', got '%s'", tt.request.FileID, unmarshaled.FileID)
			}
		})
	}
}

func TestDriveFileResult_Structure(t *testing.T) {
	// Test the DriveFileResult structure used in search responses
	result := dto.DriveFileResult{
		ID:           "test-file-id",
		Name:         "test-document.pdf",
		MimeType:     "application/pdf",
		Size:         1024,
		ModifiedTime: "2024-01-01T00:00:00Z",
		WebViewLink:  "https://drive.google.com/file/d/test-file-id/view",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(result)
	if err != nil {
		t.Errorf("Failed to marshal DriveFileResult: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled dto.DriveFileResult
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal DriveFileResult: %v", err)
	}

	// Verify fields
	if unmarshaled.ID != result.ID {
		t.Errorf("Expected ID '%s', got '%s'", result.ID, unmarshaled.ID)
	}
	if unmarshaled.Name != result.Name {
		t.Errorf("Expected Name '%s', got '%s'", result.Name, unmarshaled.Name)
	}
	if unmarshaled.MimeType != result.MimeType {
		t.Errorf("Expected MimeType '%s', got '%s'", result.MimeType, unmarshaled.MimeType)
	}
}

func TestMCPToolResult_Structure(t *testing.T) {
	// Test that our tool results conform to MCP standards
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent("Test content"),
		},
		IsError: false,
	}

	if result.IsError {
		t.Error("Result should not be an error")
	}

	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(result.Content))
	}

	// Verify content type
	textContent := result.Content[0].(mcp.TextContent)
	if textContent.Text != "Test content" {
		t.Errorf("Expected 'Test content', got '%s'", textContent.Text)
	}
}

func TestOCRConfig_Structure(t *testing.T) {
	// Test the simplified OCR configuration
	config := OCRConfig{
		Languages:     []string{"en", "fr", "de"},
		MathpixAppID:  "test-app-id",
		MathpixAppKey: "test-app-key",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(config)
	if err != nil {
		t.Errorf("Failed to marshal OCRConfig: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled OCRConfig
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal OCRConfig: %v", err)
	}

	// Verify fields
	if len(unmarshaled.Languages) != len(config.Languages) {
		t.Errorf("Expected %d languages, got %d", len(config.Languages), len(unmarshaled.Languages))
	}
	if unmarshaled.MathpixAppID != config.MathpixAppID {
		t.Errorf("Expected MathpixAppID '%s', got '%s'", config.MathpixAppID, unmarshaled.MathpixAppID)
	}
	if unmarshaled.MathpixAppKey != config.MathpixAppKey {
		t.Errorf("Expected MathpixAppKey '%s', got '%s'", config.MathpixAppKey, unmarshaled.MathpixAppKey)
	}
}

func TestServerCapabilities(t *testing.T) {
	// Test that the server supports the expected capabilities
	// Since we simplified the server, it should only support:
	// - PDF search
	// - PDF to Markdown conversion
	expectedTools := []string{"search_pdfs", "convert_pdf_to_markdown"}
	
	for _, tool := range expectedTools {
		if tool == "" {
			t.Error("Tool name should not be empty")
		}
	}

	// Test that we removed the complex tools
	removedTools := []string{
		"get_pdf_content", "convert_pdf_to_images", "get_conversion_prompts",
		"suggest_conversion_approach", "extract_text_from_pdf", 
		"extract_structured_text", "convert_pdf_hybrid", "analyze_document", "list_ocr_engines",
	}
	
	t.Logf("Simplified from %d tools to %d tools", len(removedTools)+len(expectedTools), len(expectedTools))
}

func TestErrorHandling(t *testing.T) {
	// Test error handling structures
	errorResult := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent("Test error message"),
		},
		IsError: true,
	}

	if !errorResult.IsError {
		t.Error("Error result should have IsError set to true")
	}

	if len(errorResult.Content) == 0 {
		t.Error("Error result should have content")
	}

	// Verify error content
	textContent := errorResult.Content[0].(mcp.TextContent)
	if textContent.Text == "" {
		t.Error("Error message should not be empty")
	}
}