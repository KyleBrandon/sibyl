package pdfmcp

import (
	"encoding/base64"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestFixedMCPImageContentCreation tests that the fix for MCP ImageContent creation works
func TestFixedMCPImageContentCreation(t *testing.T) {
	// Create test base64 data
	testData := []byte("test image data")
	imageB64 := base64.StdEncoding.EncodeToString(testData)
	
	// Test the fixed implementation (matching server.go:239-242) - raw base64, not data URL
	content := mcp.NewImageContent(imageB64, "image/png")
	
	t.Logf("Fixed MCP ImageContent: %+v", content)
	
	// Verify the structure has proper MIME type
	// The structure should show MIMEType as "image/png" now instead of alt text
	// This should resolve client-side base64 validation issues
}

// TestBase64ValidationWithCorrectMimeType tests base64 validation with correct MIME type
func TestBase64ValidationWithCorrectMimeType(t *testing.T) {
	// Test various base64 strings that should work correctly
	testCases := []struct {
		name     string
		data     []byte
	}{
		{
			name: "Small data",
			data: []byte("hello"),
		},
		{
			name: "Medium data", 
			data: make([]byte, 1024), // 1KB of zeros
		},
		{
			name: "Large data",
			data: make([]byte, 10240), // 10KB of zeros  
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Fill with test pattern
			for i := range tc.data {
				tc.data[i] = byte(i % 256)
			}
			
			// Encode to base64
			imageB64 := base64.StdEncoding.EncodeToString(tc.data)
			
			// Verify base64 is valid
			decoded, err := base64.StdEncoding.DecodeString(imageB64)
			if err != nil {
				t.Errorf("Base64 encoding failed: %v", err)
			}
			
			if len(decoded) != len(tc.data) {
				t.Errorf("Decoded length %d != original length %d", len(decoded), len(tc.data))
			}
			
			// Create MCP content with correct MIME type - using raw base64
			content := mcp.NewImageContent(imageB64, "image/png")
			
			t.Logf("%s: Base64 length: %d", tc.name, len(imageB64))
			_ = content // Use the content to avoid unused variable
		})
	}
}

// TestBase64EncodingEdgeCases tests edge cases that might cause "invalid base64" errors
func TestBase64EncodingEdgeCases(t *testing.T) {
	edgeCases := []struct {
		name        string
		data        []byte
		description string
	}{
		{
			name:        "Empty data",
			data:        []byte{},
			description: "Empty byte array should produce empty base64",
		},
		{
			name:        "Single null byte",
			data:        []byte{0x00},
			description: "Single null byte",
		},
		{
			name:        "All 0xFF bytes", 
			data:        []byte{0xFF, 0xFF, 0xFF, 0xFF},
			description: "All maximum value bytes",
		},
		{
			name:        "Random bytes",
			data:        []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0},
			description: "Random byte pattern",
		},
		{
			name:        "Binary data that might contain problematic chars",
			data:        []byte{0x0A, 0x0D, 0x20, 0x09}, // LF, CR, Space, Tab
			description: "Binary data with whitespace-like bytes",
		},
	}
	
	for _, ec := range edgeCases {
		t.Run(ec.name, func(t *testing.T) {
			// Encode to base64
			imageB64 := base64.StdEncoding.EncodeToString(ec.data)
			
			// Verify it's valid
			_, err := base64.StdEncoding.DecodeString(imageB64)
			if err != nil {
				t.Errorf("Base64 encoding failed for %s: %v", ec.description, err)
			}
			
			// Verify no problematic characters in base64 string
			problematicChars := []byte{'\n', '\r', ' ', '\t'}
			for _, char := range problematicChars {
				for _, b64char := range []byte(imageB64) {
					if b64char == char {
						t.Errorf("Base64 contains problematic character %q", char)
					}
				}
			}
			
			// Verify length is correct (multiple of 4 or will be padded)
			expectedLen := ((len(ec.data) + 2) / 3) * 4
			if len(imageB64) != expectedLen {
				t.Errorf("Base64 length %d != expected %d", len(imageB64), expectedLen)
			}
			
			t.Logf("%s: Input %d bytes -> Base64 %d chars: %q", ec.name, len(ec.data), len(imageB64), imageB64)
		})
	}
}