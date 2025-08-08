package pdfmcp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"

	"github.com/gen2brain/go-fitz"
)

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
