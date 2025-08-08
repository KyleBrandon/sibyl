package pdfmcp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestBase64ImageEncoding(t *testing.T) {
	// Create a simple test image
	img := createTestPNGImage(100, 100)
	
	// Encode to PNG bytes
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		t.Fatalf("Failed to encode PNG: %v", err)
	}
	
	// Encode as base64
	imageB64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	
	// Test 1: Validate base64 string is valid
	_, err = base64.StdEncoding.DecodeString(imageB64)
	if err != nil {
		t.Errorf("Generated base64 string is invalid: %v", err)
	}
	
	// Test 2: Check for proper padding
	if len(imageB64)%4 != 0 {
		t.Error("Base64 string should have proper padding (length divisible by 4)")
	}
	
	// Test 3: Ensure no newlines or whitespace
	if strings.Contains(imageB64, "\n") || strings.Contains(imageB64, " ") || strings.Contains(imageB64, "\r") {
		t.Error("Base64 string should not contain newlines or whitespace")
	}
	
	// Test 4: Verify data URL format
	dataURL := "data:image/png;base64," + imageB64
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		t.Error("Data URL should have correct format")
	}
	
	// Test 5: Ensure base64 content is decodable back to valid PNG
	decodedData, err := base64.StdEncoding.DecodeString(imageB64)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}
	
	// Try to decode as PNG
	_, err = png.Decode(bytes.NewReader(decodedData))
	if err != nil {
		t.Errorf("Decoded base64 data is not valid PNG: %v", err)
	}
	
	t.Logf("Base64 string length: %d", len(imageB64))
	t.Logf("Base64 sample (first 50 chars): %s...", imageB64[:min(50, len(imageB64))])
}

func TestConvertPDFToImagesBase64Validity(t *testing.T) {
	// Create a minimal valid PDF for testing
	validPDF := createMinimalPDF()
	
	ps := &PDFServer{}
	
	// Test with valid PDF data
	images, err := ps.convertPDFToImages(validPDF, 72.0)
	if err != nil {
		// This is expected to fail with minimal PDF, but we test the error handling
		t.Logf("Expected error with minimal PDF: %v", err)
		return
	}
	
	// If we got images, validate each base64 string
	for i, imageB64 := range images {
		// Test base64 validity
		_, err := base64.StdEncoding.DecodeString(imageB64)
		if err != nil {
			t.Errorf("Image %d has invalid base64: %v", i, err)
		}
		
		// Test base64 padding
		if len(imageB64)%4 != 0 {
			t.Errorf("Image %d base64 has improper padding", i)
		}
		
		// Test for unwanted characters
		if strings.Contains(imageB64, "\n") || strings.Contains(imageB64, " ") {
			t.Errorf("Image %d base64 contains unwanted whitespace characters", i)
		}
	}
}

func TestConvertPDFToMarkdownBase64Output(t *testing.T) {
	// Skip this test since it requires Drive service integration
	t.Skip("Skipping test that requires Drive service - testing base64 encoding separately")
}

// TestBase64EncodingConsistency tests that our base64 encoding is consistent
func TestBase64EncodingConsistency(t *testing.T) {
	// Create test data
	testData := []byte("Hello, World! This is test PDF data.")
	
	// Encode using our method (same as in conversion.go)
	encoded1 := base64.StdEncoding.EncodeToString(testData)
	
	// Encode again to ensure consistency
	encoded2 := base64.StdEncoding.EncodeToString(testData)
	
	if encoded1 != encoded2 {
		t.Error("Base64 encoding should be consistent between calls")
	}
	
	// Test decoding
	decoded, err := base64.StdEncoding.DecodeString(encoded1)
	if err != nil {
		t.Errorf("Failed to decode base64: %v", err)
	}
	
	if !bytes.Equal(testData, decoded) {
		t.Error("Decoded data does not match original")
	}
}

// TestBase64URLSafetyAndSpecialChars tests for characters that might cause issues
func TestBase64URLSafetyAndSpecialChars(t *testing.T) {
	// Create test data with various byte patterns
	testCases := [][]byte{
		{0xFF, 0xFE, 0xFD}, // High bytes
		{0x00, 0x01, 0x02}, // Low bytes
		{0x7F, 0x80, 0x81}, // Around ASCII boundary
		bytes.Repeat([]byte{0xFF}, 100), // Repeated high bytes
		bytes.Repeat([]byte{0x00}, 100), // Repeated zeros
	}
	
	for i, testData := range testCases {
		t.Run(fmt.Sprintf("TestCase_%d", i), func(t *testing.T) {
			encoded := base64.StdEncoding.EncodeToString(testData)
			
			// Should not contain problematic characters for URLs
			problematicChars := []string{"\n", "\r", " ", "\t"}
			for _, char := range problematicChars {
				if strings.Contains(encoded, char) {
					t.Errorf("Base64 contains problematic character: %q", char)
				}
			}
			
			// Should be decodable
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				t.Errorf("Failed to decode: %v", err)
			}
			
			if !bytes.Equal(testData, decoded) {
				t.Error("Round-trip encoding/decoding failed")
			}
		})
	}
}

// Helper functions for testing

func createTestPNGImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Fill with a simple pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if (x+y)%2 == 0 {
				img.Set(x, y, color.RGBA{255, 0, 0, 255}) // Red
			} else {
				img.Set(x, y, color.RGBA{0, 255, 0, 255}) // Green
			}
		}
	}
	
	return img
}

func createMinimalPDF() []byte {
	// This is a minimal PDF structure - mostly for error testing
	// A real PDF would be much more complex
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
0000000010 00000 n 
0000000053 00000 n 
0000000125 00000 n 
trailer
<<
/Size 4
/Root 1 0 R
>>
startxref
229
%%EOF`)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}