package pdf_mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/api/drive/v3"
)

// Mock HTTP response for testing
type mockHTTPResponse struct {
	body   io.ReadCloser
	status int
}

func (m *mockHTTPResponse) Read(p []byte) (n int, err error) {
	return m.body.Read(p)
}

func (m *mockHTTPResponse) Close() error {
	return m.body.Close()
}

// Mock Drive Service for testing
type mockDriveService struct {
	files   map[string]*drive.File
	content map[string][]byte
	errors  map[string]error
}

func (m *mockDriveService) GetFile(fileID string) (*drive.File, error) {
	if err, exists := m.errors[fileID]; exists {
		return nil, err
	}
	if file, exists := m.files[fileID]; exists {
		return file, nil
	}
	return &drive.File{}, nil
}

func (m *mockDriveService) DownloadFile(fileID string) (*http.Response, error) {
	if err, exists := m.errors[fileID+"_download"]; exists {
		return nil, err
	}
	if content, exists := m.content[fileID]; exists {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(content)),
		}, nil
	}
	return &http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(strings.NewReader("Not found")),
	}, nil
}

// Create a minimal valid PDF for testing
func createTestPDF() []byte {
	// This is a minimal PDF structure that go-fitz can parse
	pdfContent := `%PDF-1.4
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
/Contents 4 0 R
>>
endobj

4 0 obj
<<
/Length 44
>>
stream
BT
/F1 12 Tf
100 700 Td
(Test PDF) Tj
ET
endstream
endobj

xref
0 5
0000000000 65535 f 
0000000009 00000 n 
0000000058 00000 n 
0000000115 00000 n 
0000000189 00000 n 
trailer
<<
/Size 5
/Root 1 0 R
>>
startxref
284
%%EOF`
	return []byte(pdfContent)
}

func TestConvertPDFToImages_Success(t *testing.T) {
	// Create a mock PDF server (we'll need to modify the actual server to accept mocks)
	// For now, test the internal conversion function
	testPDF := createTestPDF()

	// Test the internal conversion function
	ps := &PDFServer{}
	images, err := ps.convertPDFToImages(testPDF, 150.0)
	if err != nil {
		t.Fatalf("convertPDFToImages failed: %v", err)
	}

	if len(images) == 0 {
		t.Fatal("No images returned from PDF conversion")
	}

	// Verify the first image is valid base64
	_, err = base64.StdEncoding.DecodeString(images[0])
	if err != nil {
		t.Errorf("First image is not valid base64: %v", err)
	}
}

func TestConvertPDFToImages_InvalidPDF(t *testing.T) {
	ps := &PDFServer{}

	// Test with invalid PDF data
	invalidPDF := []byte("This is not a PDF")

	_, err := ps.convertPDFToImages(invalidPDF, 150.0)
	if err == nil {
		t.Error("Expected error for invalid PDF, but got none")
	}
}

func TestConvertPDFToImages_DifferentDPI(t *testing.T) {
	ps := &PDFServer{}
	testPDF := createTestPDF()

	dpiTests := []float64{72, 150, 300}

	for _, dpi := range dpiTests {
		t.Run(string(rune(dpi)), func(t *testing.T) {
			images, err := ps.convertPDFToImages(testPDF, dpi)
			if err != nil {
				t.Errorf("convertPDFToImages failed at DPI %f: %v", dpi, err)
				return
			}

			if len(images) == 0 {
				t.Errorf("No images returned at DPI %f", dpi)
			}
		})
	}
}

func TestGetConversionPrompts_AllTypes(t *testing.T) {
	ctx := context.Background()

	// Create prompt manager
	pm, err := NewPromptManager()
	if err != nil {
		t.Fatalf("Failed to create PromptManager: %v", err)
	}

	ps := &PDFServer{
		ctx:     ctx,
		prompts: pm,
	}

	// Test getting all prompts
	request := mcp.CallToolRequest{}
	params := GetPromptsRequest{DocumentType: "all"}

	result, err := ps.GetConversionPrompts(ctx, request, params)
	if err != nil {
		t.Fatalf("GetConversionPrompts failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("GetConversionPrompts returned error: %v", result.Content[0])
	}

	if len(result.Content) == 0 {
		t.Fatal("No content returned from GetConversionPrompts")
	}

	// Verify the content is valid JSON
	content := result.Content[0].(mcp.TextContent).Text
	var prompts map[string]PromptTemplate
	err = json.Unmarshal([]byte(content), &prompts)
	if err != nil {
		t.Errorf("Invalid JSON returned: %v", err)
	}

	// Verify expected prompt types are present
	expectedTypes := []string{"handwritten", "typed", "mixed", "research"}
	for _, promptType := range expectedTypes {
		if _, exists := prompts[promptType]; !exists {
			t.Errorf("Expected prompt type '%s' not found", promptType)
		}
	}
}

func TestGetConversionPrompts_SpecificType(t *testing.T) {
	ctx := context.Background()

	pm, err := NewPromptManager()
	if err != nil {
		t.Fatalf("Failed to create PromptManager: %v", err)
	}

	ps := &PDFServer{
		ctx:     ctx,
		prompts: pm,
	}

	tests := []struct {
		name         string
		documentType string
		expectError  bool
	}{
		{"Handwritten type", "handwritten", false},
		{"Typed type", "typed", false},
		{"Mixed type", "mixed", false},
		{"Research type", "research", false},
		{"Invalid type", "nonexistent", true},
		{"Empty type", "", false}, // Should return all
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{}
			params := GetPromptsRequest{DocumentType: tt.documentType}

			result, err := ps.GetConversionPrompts(ctx, request, params)
			if err != nil {
				t.Fatalf("GetConversionPrompts failed: %v", err)
			}

			if tt.expectError {
				if !result.IsError {
					t.Error("Expected error result, but got success")
				}
				return
			}

			if result.IsError {
				t.Errorf("Unexpected error result: %v", result.Content[0])
				return
			}

			if len(result.Content) == 0 {
				t.Error("No content returned")
			}
		})
	}
}

func TestSuggestConversionApproach_Success(t *testing.T) {
	pm, err := NewPromptManager()
	if err != nil {
		t.Fatalf("Failed to create PromptManager: %v", err)
	}

	// Test the suggestion logic directly
	suggestion := pm.SuggestPromptType("handwritten_notes.pdf", 1024)
	if suggestion.RecommendedType == "" {
		t.Error("No recommended type returned")
	}

	if suggestion.Confidence <= 0 {
		t.Error("Confidence should be greater than 0")
	}

	if suggestion.Reasoning == "" {
		t.Error("No reasoning provided")
	}
}

func TestConvertPDFToImagesRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request ConvertPDFToImagesRequest
		isValid bool
	}{
		{
			name: "Valid request with DPI",
			request: ConvertPDFToImagesRequest{
				FileID: "test123",
				DPI:    150,
			},
			isValid: true,
		},
		{
			name: "Valid request without DPI",
			request: ConvertPDFToImagesRequest{
				FileID: "test123",
			},
			isValid: true,
		},
		{
			name: "Invalid request - empty FileID",
			request: ConvertPDFToImagesRequest{
				FileID: "",
				DPI:    150,
			},
			isValid: false,
		},
		{
			name: "Invalid request - negative DPI",
			request: ConvertPDFToImagesRequest{
				FileID: "test123",
				DPI:    -1,
			},
			isValid: true, // Should use default DPI
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

			var unmarshaled ConvertPDFToImagesRequest
			err = json.Unmarshal(jsonData, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal request: %v", err)
				return
			}

			if unmarshaled.FileID != tt.request.FileID {
				t.Errorf("FileID mismatch: expected %s, got %s", tt.request.FileID, unmarshaled.FileID)
			}

			if unmarshaled.DPI != tt.request.DPI {
				t.Errorf("DPI mismatch: expected %d, got %d", tt.request.DPI, unmarshaled.DPI)
			}
		})
	}
}

func TestGetPromptsRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request GetPromptsRequest
	}{
		{
			name: "Valid request with document type",
			request: GetPromptsRequest{
				DocumentType: "handwritten",
			},
		},
		{
			name:    "Valid request without document type",
			request: GetPromptsRequest{},
		},
		{
			name: "Valid request with 'all' type",
			request: GetPromptsRequest{
				DocumentType: "all",
			},
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

			var unmarshaled GetPromptsRequest
			err = json.Unmarshal(jsonData, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal request: %v", err)
				return
			}

			if unmarshaled.DocumentType != tt.request.DocumentType {
				t.Errorf("DocumentType mismatch: expected %s, got %s", tt.request.DocumentType, unmarshaled.DocumentType)
			}
		})
	}
}

func BenchmarkConvertPDFToImages(b *testing.B) {
	ps := &PDFServer{}
	testPDF := createTestPDF()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ps.convertPDFToImages(testPDF, 150.0)
		if err != nil {
			b.Fatalf("convertPDFToImages failed: %v", err)
		}
	}
}

func BenchmarkBase64Encoding(b *testing.B) {
	// Create some test image data
	testData := make([]byte, 1024*1024) // 1MB of test data
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = base64.StdEncoding.EncodeToString(testData)
	}
}
