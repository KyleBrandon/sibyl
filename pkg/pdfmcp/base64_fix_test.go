package pdfmcp

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestBase64PaddingFix tests for base64 padding issues that could cause "invalid base64 string" errors
func TestBase64PaddingFix(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "Single byte",
			input:    []byte{0x41}, // 'A'
			expected: "QQ==",       // Should have 2 padding chars
		},
		{
			name:     "Two bytes", 
			input:    []byte{0x41, 0x42}, // 'AB'
			expected: "QUI=",             // Should have 1 padding char
		},
		{
			name:     "Three bytes",
			input:    []byte{0x41, 0x42, 0x43}, // 'ABC'
			expected: "QUJD",               // Should have no padding
		},
		{
			name:     "Four bytes",
			input:    []byte{0x41, 0x42, 0x43, 0x44}, // 'ABCD'
			expected: "QUJDRA==",                     // Should have 2 padding chars
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test our encoding matches expected
			encoded := base64.StdEncoding.EncodeToString(tc.input)
			if encoded != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, encoded)
			}

			// Test that it's valid base64
			_, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				t.Errorf("Generated base64 is invalid: %v", err)
			}

			// Test length is multiple of 4
			if len(encoded)%4 != 0 {
				t.Errorf("Base64 length %d is not multiple of 4", len(encoded))
			}

			t.Logf("Input: %v -> Base64: %s (length: %d)", tc.input, encoded, len(encoded))
		})
	}
}

// TestBase64WithDataURL tests the full data URL construction
func TestBase64WithDataURL(t *testing.T) {
	testData := []byte("test image data")
	encoded := base64.StdEncoding.EncodeToString(testData)
	
	dataURL := fmt.Sprintf("data:image/png;base64,%s", encoded)
	
	// Extract base64 part
	prefix := "data:image/png;base64,"
	if !strings.HasPrefix(dataURL, prefix) {
		t.Fatalf("Data URL missing expected prefix")
	}
	
	extractedB64 := strings.TrimPrefix(dataURL, prefix)
	if extractedB64 != encoded {
		t.Errorf("Extracted base64 doesn't match original: %s vs %s", extractedB64, encoded)
	}
	
	// Validate extracted base64
	decoded, err := base64.StdEncoding.DecodeString(extractedB64)
	if err != nil {
		t.Errorf("Extracted base64 is invalid: %v", err)
	}
	
	if string(decoded) != string(testData) {
		t.Errorf("Round-trip failed: %s vs %s", decoded, testData)
	}
	
	t.Logf("Data URL: %s", dataURL)
	t.Logf("Base64 part: %s", extractedB64)
}

// TestMCPImageContentStructure tests the MCP ImageContent structure
func TestMCPImageContentStructure(t *testing.T) {
	testData := []byte("test")
	encoded := base64.StdEncoding.EncodeToString(testData)
	
	// Test with raw base64 (correct implementation)
	content := mcp.NewImageContent(encoded, "image/png")
	
	// Check the structure
	t.Logf("MCP ImageContent structure: %+v", content)
	
	// The structure appears to be:
	// {Annotated:{Annotations:<nil>} Type:image Data:dataURL MIMEType:altText}
	// This suggests the API might be: NewImageContent(data, mimeType) not NewImageContent(data, altText)
}

// TestCorrectMCPImageContentUsage tests if there's a better way to create image content
func TestCorrectMCPImageContentUsage(t *testing.T) {
	testData := []byte("test image")
	encoded := base64.StdEncoding.EncodeToString(testData)
	
	// Test with raw base64 and MIME type (correct usage)
	content1 := mcp.NewImageContent(encoded, "image/png")
	t.Logf("With raw base64 and MIME type: %+v", content1)
	
	// Test with raw base64 and different MIME type
	content2 := mcp.NewImageContent(encoded, "image/jpeg")
	t.Logf("With raw base64 and different MIME type: %+v", content2)
	
	// Based on the structure, it looks like the second parameter is treated as MIMEType,
	// not alt text. This might explain client-side validation issues.
}

// TestBase64StrictValidation tests stricter base64 validation
func TestBase64StrictValidation(t *testing.T) {
	testCases := []struct {
		name    string
		b64     string
		isValid bool
	}{
		{
			name:    "Valid base64",
			b64:     "SGVsbG8gV29ybGQ=",
			isValid: true,
		},
		{
			name:    "Invalid without padding", 
			b64:     "SGVsbG8gV29ybGQ", // Go's decoder is stricter than expected
			isValid: false,
		},
		{
			name:    "Invalid character",
			b64:     "SGVs!bG8gV29ybGQ=",
			isValid: false,
		},
		{
			name:    "Empty string",
			b64:     "",
			isValid: true, // Empty string is valid base64
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := base64.StdEncoding.DecodeString(tc.b64)
			isValid := err == nil
			
			if isValid != tc.isValid {
				t.Errorf("Expected isValid=%v, got isValid=%v for %s", tc.isValid, isValid, tc.b64)
			}
			
			if err != nil {
				t.Logf("Decode error: %v", err)
			}
		})
	}
}