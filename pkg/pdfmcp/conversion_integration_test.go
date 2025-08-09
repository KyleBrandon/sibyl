package pdfmcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestConvertPDFToMarkdownIntegration tests the full convert_pdf_to_markdown flow
func TestConvertPDFToMarkdownIntegration(t *testing.T) {
	// Skip if we don't have a test PDF file
	testPDFPath := "testdata/sample.pdf"
	if _, err := os.Stat(testPDFPath); os.IsNotExist(err) {
		t.Skip("Skipping integration test - no test PDF available")
	}

	// Create a server with mock dependencies
	ctx := context.Background()
	
	ocrManager := NewOCRManager()
	mockOCR := NewMockOCR([]string{"en"})
	ocrManager.RegisterEngine("mathpix", mockOCR)
	if err := ocrManager.SetDefaultEngine("mathpix"); err != nil {
		t.Fatalf("Failed to set default OCR engine: %v", err)
	}

	ps := &PDFServer{
		ctx:        ctx,
		ocrManager: ocrManager,
	}

	// Read the test PDF
	pdfData, err := os.ReadFile(testPDFPath)
	if err != nil {
		t.Skipf("Cannot read test PDF: %v", err)
	}

	// Test the PDF to images conversion directly
	images, err := ps.convertPDFToImages(pdfData, 150.0)
	if err != nil {
		t.Fatalf("convertPDFToImages failed: %v", err)
	}

	// Validate each image
	for i, imageB64 := range images {
		t.Run(fmt.Sprintf("Image_%d", i), func(t *testing.T) {
			// Check base64 validity
			if !isValidBase64(imageB64) {
				t.Errorf("Image %d has invalid base64 encoding", i)
			}

			// Check that it decodes to valid image data
			imageData, err := base64.StdEncoding.DecodeString(imageB64)
			if err != nil {
				t.Errorf("Image %d base64 decode failed: %v", i, err)
				return
			}

			// Check PNG signature
			if len(imageData) < 8 {
				t.Errorf("Image %d data too short", i)
				return
			}

			pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
			if !bytes.HasPrefix(imageData, pngSignature) {
				t.Errorf("Image %d does not have PNG signature", i)
			}

			t.Logf("Image %d: %d bytes, base64 length: %d", i, len(imageData), len(imageB64))
		})
	}
}

// TestBase64ErrorScenarios tests potential base64 error scenarios
func TestBase64ErrorScenarios(t *testing.T) {
	testCases := []struct {
		name           string
		base64Input    string
		expectError    bool
		expectedIssue  string
	}{
		{
			name:        "Valid base64",
			base64Input: "SGVsbG8gV29ybGQ=", // "Hello World"
			expectError: false,
		},
		{
			name:           "Missing padding",
			base64Input:    "SGVsbG8gV29ybGQ", // Missing =
			expectError:    true,
			expectedIssue:  "Missing padding",
		},
		{
			name:           "Invalid characters",
			base64Input:    "SGVsbG8gV29ybGQ@", // @ is not valid base64
			expectError:    true,
			expectedIssue:  "Invalid characters",
		},
		{
			name:           "Contains newline",
			base64Input:    "SGVsbG8g\nV29ybGQ=",
			expectError:    false, // Go base64 decoder handles newlines
			expectedIssue:  "Contains newline",
		},
		{
			name:           "Contains space",
			base64Input:    "SGVsbG8g V29ybGQ=",
			expectError:    true,
			expectedIssue:  "Contains space",
		},
		{
			name:           "Wrong length",
			base64Input:    "SGVsbG8", // Length not multiple of 4 without padding
			expectError:    true,
			expectedIssue:  "Wrong length",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := base64.StdEncoding.DecodeString(tc.base64Input)
			
			if tc.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tc.expectedIssue)
			}
			
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			// Additional validation checks
			if strings.Contains(tc.base64Input, "\n") {
				t.Logf("Contains newline character")
			}
			if strings.Contains(tc.base64Input, " ") {
				t.Logf("Contains space character")
			}
			if len(tc.base64Input)%4 != 0 {
				t.Logf("Length not multiple of 4: %d", len(tc.base64Input))
			}
		})
	}
}

// TestMCPImageContentCreation tests the MCP image content creation
func TestMCPImageContentCreation(t *testing.T) {
	// Create a simple test image base64
	testImageB64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChAI9fINglgAAAABJRU5ErkJggg==" // 1x1 red pixel

	// Validate the base64 first
	if !isValidBase64(testImageB64) {
		t.Fatal("Test base64 is invalid")
	}

	// Test MCP content creation (similar to server.go:239-242) - using raw base64
	content := mcp.NewImageContent(testImageB64, "image/png")

	// The NewImageContent returns a Content interface, so we need to inspect its properties
	t.Logf("Content created successfully: %+v", content)

	// Validate the base64 string directly
	if !isValidBase64(testImageB64) {
		t.Error("Base64 string is invalid")
	}
}

// TestLargeBase64Handling tests handling of larger base64 strings
func TestLargeBase64Handling(t *testing.T) {
	// Create a larger test image (100x100 solid color PNG)
	img := createTestPNGImage(100, 100)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("Failed to encode PNG: %v", err)
	}

	largeBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	
	// Validate it's still good base64
	if !isValidBase64(largeBase64) {
		t.Error("Large base64 string is invalid")
	}

	// Test creating MCP content with large base64 - using raw base64
	content := mcp.NewImageContent(largeBase64, "image/png")

	if len(largeBase64) == 0 {
		t.Error("Base64 string is empty")
	}

	t.Logf("Large base64 length: %d", len(largeBase64))
	t.Logf("Content created: %+v", content)
}

// Helper function to validate base64 strings
func isValidBase64(s string) bool {
	// Check for invalid characters
	if strings.ContainsAny(s, " \n\r\t") {
		return false
	}
	
	// Check length (must be multiple of 4)
	if len(s)%4 != 0 {
		return false
	}
	
	// Try to decode
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

// TestDownloadPDFContentMock tests the PDF download simulation
func TestDownloadPDFContentMock(t *testing.T) {
	// Create mock HTTP response body
	testPDFData := []byte("%PDF-1.4\nTest PDF content\n%%EOF")
	mockBody := io.NopCloser(bytes.NewReader(testPDFData))

	// Mock the response reading logic from server.go:302-313
	pdfData := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := mockBody.Read(buf)
		if n > 0 {
			pdfData = append(pdfData, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	if !bytes.Equal(testPDFData, pdfData) {
		t.Error("PDF data reading logic failed")
	}

	t.Logf("Successfully read %d bytes of PDF data", len(pdfData))
}

// TestHTTPResponseHandling tests potential HTTP response issues
func TestHTTPResponseHandling(t *testing.T) {
	testCases := []struct {
		name         string
		responseBody []byte
		expectError  bool
	}{
		{
			name:         "Valid PDF",
			responseBody: []byte("%PDF-1.4\nContent\n%%EOF"),
			expectError:  false,
		},
		{
			name:         "Empty response",
			responseBody: []byte{},
			expectError:  false, // Should handle gracefully
		},
		{
			name:         "Large response",
			responseBody: bytes.Repeat([]byte("test"), 10000),
			expectError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate reading HTTP response body
			mockBody := io.NopCloser(bytes.NewReader(tc.responseBody))
			defer mockBody.Close()

			pdfData := make([]byte, 0)
			buf := make([]byte, 1024)
			for {
				n, err := mockBody.Read(buf)
				if n > 0 {
					pdfData = append(pdfData, buf[:n]...)
				}
				if err != nil {
					break
				}
			}

			if !tc.expectError && len(pdfData) != len(tc.responseBody) {
				t.Errorf("Expected %d bytes, got %d", len(tc.responseBody), len(pdfData))
			}

			t.Logf("Read %d bytes from mock response", len(pdfData))
		})
	}
}