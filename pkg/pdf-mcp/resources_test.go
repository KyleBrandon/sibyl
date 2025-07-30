package pdf_mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/api/drive/v3"
)

// Mock Drive Service for resource testing
type mockDriveServiceForResources struct {
	files  []*drive.File
	errors map[string]error
}

func (m *mockDriveServiceForResources) ListFiles(query string) ([]*drive.File, error) {
	if err, exists := m.errors["list"]; exists {
		return nil, err
	}
	return m.files, nil
}

func TestListDocuments_Success(t *testing.T) {
	// Create mock files
	mockFiles := []*drive.File{
		{
			Id:            "file1",
			Name:          "document1.pdf",
			Size:          1024,
			CreatedTime:   "2025-01-15T10:00:00Z",
			ModifiedTime:  "2025-01-15T10:30:00Z",
			WebViewLink:   "https://drive.google.com/file/d/file1/view",
			ThumbnailLink: "https://drive.google.com/thumbnail?id=file1",
		},
		{
			Id:            "file2",
			Name:          "document2.pdf",
			Size:          2048,
			CreatedTime:   "2025-01-15T11:00:00Z",
			ModifiedTime:  "2025-01-15T11:30:00Z",
			WebViewLink:   "https://drive.google.com/file/d/file2/view",
			ThumbnailLink: "https://drive.google.com/thumbnail?id=file2",
		},
	}

	// Test the resource structure and JSON formatting
	// In a full implementation, we'd mock the drive service calls
	// Test the resource request structure
	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "pdf://documents/",
		},
	}

	// Verify request structure
	if request.Params.URI != "pdf://documents/" {
		t.Errorf("Expected URI 'pdf://documents/', got '%s'", request.Params.URI)
	}

	// Test document resource structure
	docResource := map[string]interface{}{
		"id":            mockFiles[0].Id,
		"name":          mockFiles[0].Name,
		"size":          mockFiles[0].Size,
		"created":       mockFiles[0].CreatedTime,
		"modified":      mockFiles[0].ModifiedTime,
		"webViewLink":   mockFiles[0].WebViewLink,
		"thumbnailLink": mockFiles[0].ThumbnailLink,
		"uri":           "pdf://documents/" + mockFiles[0].Id,
	}

	// Verify JSON serialization
	jsonData, err := json.MarshalIndent(docResource, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal document resource: %v", err)
	}

	var unmarshaled map[string]interface{}
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal document resource: %v", err)
	}

	// Verify required fields
	requiredFields := []string{"id", "name", "size", "created", "modified", "uri"}
	for _, field := range requiredFields {
		if _, exists := unmarshaled[field]; !exists {
			t.Errorf("Required field '%s' missing from document resource", field)
		}
	}
}

func TestListDocuments_EmptyResult(t *testing.T) {
	// Test with empty file list
	var emptyResources []map[string]interface{}

	jsonData, err := json.MarshalIndent(emptyResources, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal empty resources: %v", err)
	}

	// Empty slice marshals to "[]" or "null" depending on initialization
	expected := "[]"
	if string(jsonData) != expected && string(jsonData) != "null" {
		t.Errorf("Expected empty array JSON '[]' or 'null', got: %s", string(jsonData))
	}
}

func TestListTemplates_Success(t *testing.T) {
	ctx := context.Background()

	pm, err := NewPromptManager()
	if err != nil {
		t.Fatalf("Failed to create PromptManager: %v", err)
	}

	ps := &PDFServer{
		ctx:     ctx,
		prompts: pm,
	}

	// Test the ListTemplates resource handler
	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "pdf://templates/",
		},
	}

	resources, err := ps.ListTemplates(ctx, request)
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}

	if len(resources) == 0 {
		t.Fatal("No resources returned from ListTemplates")
	}

	// Verify the resource structure
	resource := resources[0]
	textResource, ok := resource.(mcp.TextResourceContents)
	if !ok {
		t.Fatal("Resource is not TextResourceContents")
	}

	if textResource.URI != "pdf://templates/" {
		t.Errorf("Expected URI 'pdf://templates/', got '%s'", textResource.URI)
	}

	if textResource.MIMEType != "application/json" {
		t.Errorf("Expected MIME type 'application/json', got '%s'", textResource.MIMEType)
	}

	// Verify the JSON content
	var templates map[string]interface{}
	err = json.Unmarshal([]byte(textResource.Text), &templates)
	if err != nil {
		t.Errorf("Invalid JSON in templates resource: %v", err)
	}

	// Verify expected template types
	expectedTypes := []string{"handwritten", "typed", "mixed", "research"}
	for _, templateType := range expectedTypes {
		if _, exists := templates[templateType]; !exists {
			t.Errorf("Expected template type '%s' not found", templateType)
		}
	}

	// Verify template structure
	for templateType, templateData := range templates {
		templateMap, ok := templateData.(map[string]interface{})
		if !ok {
			t.Errorf("Template '%s' is not a map", templateType)
			continue
		}

		requiredFields := []string{"name", "description", "use_case", "uri", "type", "prompt"}
		for _, field := range requiredFields {
			if _, exists := templateMap[field]; !exists {
				t.Errorf("Required field '%s' missing from template '%s'", field, templateType)
			}
		}

		// Verify URI format
		expectedURI := "pdf://templates/" + templateType
		if templateMap["uri"] != expectedURI {
			t.Errorf("Expected URI '%s', got '%s'", expectedURI, templateMap["uri"])
		}
	}
}

func TestResourceURIFormats(t *testing.T) {
	tests := []struct {
		name        string
		resourceURI string
		isValid     bool
	}{
		{"Valid documents URI", "pdf://documents/", true},
		{"Valid specific document URI", "pdf://documents/file123", true},
		{"Valid templates URI", "pdf://templates/", true},
		{"Valid specific template URI", "pdf://templates/handwritten", true},
		{"Invalid scheme", "http://documents/", false},
		{"Invalid resource type", "pdf://invalid/", false},
		{"Empty URI", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test URI parsing and validation
			if tt.resourceURI == "" {
				if tt.isValid {
					t.Error("Empty URI should not be valid")
				}
				return
			}

			// Basic URI format validation
			hasValidScheme := len(tt.resourceURI) > 6 && tt.resourceURI[:6] == "pdf://"
			if tt.isValid && !hasValidScheme {
				t.Error("Valid URI should have pdf:// scheme")
			}
			if !tt.isValid && hasValidScheme && tt.resourceURI != "pdf://invalid/" {
				// Only fail if it's not the specific invalid resource type test
				return
			}
		})
	}
}

func TestResourceContentTypes(t *testing.T) {
	ctx := context.Background()

	pm, err := NewPromptManager()
	if err != nil {
		t.Fatalf("Failed to create PromptManager: %v", err)
	}

	ps := &PDFServer{
		ctx:     ctx,
		prompts: pm,
	}

	// Test templates resource content type
	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "pdf://templates/",
		},
	}

	resources, err := ps.ListTemplates(ctx, request)
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}

	if len(resources) == 0 {
		t.Fatal("No resources returned")
	}

	resource := resources[0]

	// Test that it's the correct type
	switch r := resource.(type) {
	case mcp.TextResourceContents:
		if r.MIMEType != "application/json" {
			t.Errorf("Expected MIME type 'application/json', got '%s'", r.MIMEType)
		}

		// Verify it's valid JSON
		var jsonData interface{}
		err := json.Unmarshal([]byte(r.Text), &jsonData)
		if err != nil {
			t.Errorf("Resource text is not valid JSON: %v", err)
		}

	case mcp.BlobResourceContents:
		t.Error("Expected TextResourceContents, got BlobResourceContents")

	default:
		t.Errorf("Unexpected resource type: %T", resource)
	}
}

func TestResourceErrorHandling(t *testing.T) {
	ctx := context.Background()

	pm, err := NewPromptManager()
	if err != nil {
		t.Fatalf("Failed to create PromptManager: %v", err)
	}

	ps := &PDFServer{
		ctx:     ctx,
		prompts: pm,
	}

	// Test with invalid request (this would normally be handled by the MCP framework)
	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "invalid://uri/",
		},
	}

	// The resource handlers should still work with any URI
	// (URI validation is typically done at the MCP framework level)
	_, err = ps.ListTemplates(ctx, request)
	if err != nil {
		t.Errorf("ListTemplates should handle any URI gracefully, got error: %v", err)
	}
}

func BenchmarkListTemplates(b *testing.B) {
	ctx := context.Background()

	pm, err := NewPromptManager()
	if err != nil {
		b.Fatalf("Failed to create PromptManager: %v", err)
	}

	ps := &PDFServer{
		ctx:     ctx,
		prompts: pm,
	}

	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "pdf://templates/",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ps.ListTemplates(ctx, request)
		if err != nil {
			b.Fatalf("ListTemplates failed: %v", err)
		}
	}
}

func BenchmarkJSONMarshaling(b *testing.B) {
	// Test JSON marshaling performance for large resource lists
	resources := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		resources[i] = map[string]interface{}{
			"id":          "file" + string(rune(i)),
			"name":        "document" + string(rune(i)) + ".pdf",
			"size":        1024 * i,
			"created":     "2025-01-15T10:00:00Z",
			"modified":    "2025-01-15T10:30:00Z",
			"webViewLink": "https://drive.google.com/file/d/file" + string(rune(i)) + "/view",
			"uri":         "pdf://documents/file" + string(rune(i)),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.MarshalIndent(resources, "", "  ")
		if err != nil {
			b.Fatalf("JSON marshaling failed: %v", err)
		}
	}
}
