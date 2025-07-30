package pdf_mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"

	"github.com/gen2brain/go-fitz"
	"github.com/mark3labs/mcp-go/mcp"
)

// ConvertPDFToImages converts a PDF file to images for LLM processing
func (ps *PDFServer) ConvertPDFToImages(ctx context.Context, request mcp.CallToolRequest, params ConvertPDFToImagesRequest) (*mcp.CallToolResult, error) {
	dpi := params.DPI
	if dpi <= 0 {
		dpi = 150 // Default DPI
	}

	// First get the PDF content
	resp, err := ps.driveService.Files.Get(params.FileID).Download()
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Error downloading PDF: %v", err)),
			},
		}, nil
	}
	defer resp.Body.Close()

	// Read PDF data
	pdfData := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			pdfData = append(pdfData, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Convert PDF to images
	images, err := ps.convertPDFToImages(pdfData, float64(dpi))
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Error converting PDF to images: %v", err)),
			},
		}, nil
	}

	// Prepare result
	result := map[string]interface{}{
		"file_id":    params.FileID,
		"page_count": len(images),
		"dpi":        dpi,
		"images":     images,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(resultJSON)),
		},
	}, nil
}

// GetConversionPrompts returns prompt templates for PDF conversion
func (ps *PDFServer) GetConversionPrompts(ctx context.Context, request mcp.CallToolRequest, params GetPromptsRequest) (*mcp.CallToolResult, error) {
	if params.DocumentType == "" || params.DocumentType == "all" {
		// Return all prompts
		promptsJSON, err := ps.prompts.GetPromptsAsJSON()
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting prompts: %v", err)),
				},
			}, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(promptsJSON),
			},
		}, nil
	}

	// Return specific prompt
	prompt, err := ps.prompts.GetPrompt(params.DocumentType)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Error getting prompt: %v", err)),
			},
		}, nil
	}

	promptJSON, _ := json.MarshalIndent(prompt, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(promptJSON)),
		},
	}, nil
}

// SuggestConversionApproach analyzes a PDF and suggests the best conversion approach
func (ps *PDFServer) SuggestConversionApproach(ctx context.Context, request mcp.CallToolRequest, params GetPDFContentRequest) (*mcp.CallToolResult, error) {
	// Get file metadata
	file, err := ps.driveService.Files.
		Get(params.FileID).
		Fields("id", "name", "size", "modifiedTime").
		Do()
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Error getting file metadata: %v", err)),
			},
		}, nil
	}

	// Get suggestion from prompt manager
	suggestion := ps.prompts.SuggestPromptType(file.Name, file.Size)

	// Add file metadata to suggestion
	result := map[string]interface{}{
		"file_id":         params.FileID,
		"file_name":       file.Name,
		"file_size":       file.Size,
		"suggestion":      suggestion,
		"available_types": ps.prompts.ListPromptTypes(),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(resultJSON)),
		},
	}, nil
}

// convertPDFToImages is the internal method that does the actual conversion
func (ps *PDFServer) convertPDFToImages(pdfData []byte, dpi float64) ([]string, error) {
	// Create a new document from PDF data
	doc, err := fitz.NewFromMemory(pdfData)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	var images []string
	pageCount := doc.NumPage()

	// Convert each page to PNG
	for i := 0; i < pageCount; i++ {
		// Render page as image with specified DPI
		img, err := doc.ImageDPI(i, dpi)
		if err != nil {
			return nil, fmt.Errorf("failed to render page %d: %w", i+1, err)
		}

		// Convert image to PNG bytes
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("failed to encode page %d as PNG: %w", i+1, err)
		}

		// Encode as base64
		imageB64 := base64.StdEncoding.EncodeToString(buf.Bytes())
		images = append(images, imageB64)
	}

	return images, nil
}
