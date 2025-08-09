// Package pdfmcp provides MCP server for PDF processing and conversion
package pdfmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/KyleBrandon/sibyl/pkg/dto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type PDFServer struct {
	ctx          context.Context
	McpServer    *server.MCPServer
	driveService *drive.Service
	folderID     string
	ocrManager   *OCRManager
}

// Request types for MCP tools
type SearchPDFsRequest struct {
	Query    string `json:"query" mcp:"Search query for PDF files"`
	MaxFiles int    `json:"max_files,omitempty" mcp:"Maximum number of files to return (default: 10)"`
}

// Simplified conversion request
type ConvertPDFToMarkdownRequest struct {
	FileID string `json:"file_id" mcp:"Google Drive file ID of the PDF to convert"`
}

// OCRConfig holds Mathpix OCR configuration (required for simplified approach)
type OCRConfig struct {
	Languages     []string `json:"languages"`
	MathpixAppID  string   `json:"mathpix_app_id"`
	MathpixAppKey string   `json:"mathpix_app_key"`
}

func NewPDFServer(ctx context.Context, credentialsPath, folderID string, ocrConfig OCRConfig) (*PDFServer, error) {
	s := &PDFServer{}

	// Initialize Google Drive service
	driveService, err := drive.NewService(
		ctx,
		option.WithCredentialsFile(credentialsPath),
		option.WithScopes(drive.DriveScope))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	// Initialize OCR manager with Mathpix (required for simplified approach)
	if ocrConfig.MathpixAppID == "" || ocrConfig.MathpixAppKey == "" {
		return nil, fmt.Errorf("Mathpix credentials are required for PDF conversion")
	}

	ocrManager := NewOCRManager()

	// Register only Mathpix OCR engine
	mathpix := NewMathpixOCR(ocrConfig.MathpixAppID, ocrConfig.MathpixAppKey, ocrConfig.Languages)
	ocrManager.RegisterEngine("mathpix", mathpix)
	err = ocrManager.SetDefaultEngine("mathpix")
	if err != nil {
		return nil, fmt.Errorf("failed to set default OCR engine: %w", err)
	}

	s.ctx = ctx
	s.driveService = driveService
	s.folderID = folderID
	s.ocrManager = ocrManager
	s.McpServer = server.NewMCPServer("pdf-server", "v1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(false, false))
	s.addTools()
	s.addResources()

	return s, nil
}

func (ps *PDFServer) addResources() {
	// Resource: PDF Documents collection
	documentsResource := mcp.NewResource(
		"pdf://documents/",
		"PDF Documents",
		mcp.WithResourceDescription("Collection of PDF documents in Google Drive folder"),
		mcp.WithMIMEType("application/json"),
	)
	ps.McpServer.AddResource(documentsResource, ps.ListDocuments)
}

func (ps *PDFServer) addTools() {
	// Tool 1: Search for PDF files (keeping this for discovery)
	searchTool := mcp.NewTool(
		"search_pdfs",
		mcp.WithDescription("Search for PDF files in Google Drive"),
		mcp.WithString("query", mcp.Description("Search query for PDF files"), mcp.Required()),
		mcp.WithNumber("max_files", mcp.Description("Maximum number of files to return (default: 10)")),
	)
	ps.McpServer.AddTool(searchTool, mcp.NewTypedToolHandler(ps.SearchPDFs))

	// Tool 2: Simplified PDF to Markdown conversion
	convertTool := mcp.NewTool(
		"convert_pdf_to_markdown",
		mcp.WithDescription("Convert PDF to Markdown using simplified 3-step process: PDF->PNG, PDF->Mathpix OCR, PNG+OCR->LLM refinement"),
		mcp.WithString("file_id", mcp.Description("Google Drive file ID of the PDF to convert"), mcp.Required()),
	)
	ps.McpServer.AddTool(convertTool, mcp.NewTypedToolHandler(ps.ConvertPDFToMarkdown))
}

func (ps *PDFServer) SearchPDFs(ctx context.Context, request mcp.CallToolRequest, params SearchPDFsRequest) (*mcp.CallToolResult, error) {
	maxFiles := params.MaxFiles
	if maxFiles <= 0 {
		maxFiles = 10
	}

	// Build search query for Google Drive
	query := fmt.Sprintf("name contains '%s' and mimeType='application/pdf' and trashed=false and '%s' in parents",
		params.Query, ps.folderID)

	// Search for files
	files, err := ps.driveService.Files.List().
		Q(query).
		PageSize(int64(maxFiles)).
		Spaces("drive").
		Fields("files(id, name, parents, createdTime, modifiedTime, size, webViewLink, thumbnailLink)").
		Do()
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Error searching PDF files: %v", err)),
			},
		}, nil
	}

	// Format results
	results := make([]dto.DriveFileResult, 0)
	for _, file := range files.Files {
		results = append(results, dto.DriveFileResult{
			ID:           file.Id,
			Name:         file.Name,
			MimeType:     file.MimeType,
			Size:         file.Size,
			ModifiedTime: file.ModifiedTime,
			WebViewLink:  file.WebViewLink,
		})
	}

	resultJSON, _ := json.MarshalIndent(results, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(resultJSON)),
		},
	}, nil
}

// ConvertPDFToMarkdown implements the simplified 3-step conversion process:
// 1. Convert PDF to PNG images
// 2. Send PDF to Mathpix for OCR
// 3. Send PNG + OCR text to LLM for refinement (returned as structured content)
func (ps *PDFServer) ConvertPDFToMarkdown(ctx context.Context, request mcp.CallToolRequest, params ConvertPDFToMarkdownRequest) (*mcp.CallToolResult, error) {
	// Step 1: Get PDF content
	pdfContent, err := ps.getPDFContentBytes(params.FileID)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Failed to get PDF content: %v", err)),
			},
		}, nil
	}

	// Step 2: Convert PDF to PNG images (150 DPI default)
	base64Images, err := ps.convertPDFToImages(pdfContent, 150.0)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Failed to convert PDF to images: %v", err)),
			},
		}, nil
	}

	if len(base64Images) == 0 {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent("No images generated from PDF"),
			},
		}, nil
	}

	// Step 3: Process PDF with Mathpix OCR
	mathpixEngine, err := ps.ocrManager.GetEngine("mathpix")
	if err != nil || mathpixEngine == nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent("Mathpix OCR engine not available"),
			},
		}, nil
	}

	ocrResult, err := mathpixEngine.ProcessPDF(ctx, pdfContent)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Failed to process PDF with Mathpix: %v", err)),
			},
		}, nil
	}

	// Step 4: Return both the OCR text and images for LLM refinement
	// The LLM host will receive both and can use them together for optimal results
	content := []mcp.Content{
		mcp.NewTextContent(fmt.Sprintf(`# PDF Conversion Results

## Mathpix OCR Output
%s

## Processing Info
- Engine: %s
- Confidence: %.2f
- Processing time: %v
- Pages converted: %d

## Instructions for LLM Refinement
The above is the OCR output from Mathpix. You also have access to the PNG images of each page below.
Please review both the OCR text and the images to create the most accurate Markdown conversion.
Correct any OCR errors you can identify by comparing with the visual images.

`, ocrResult.Text, ocrResult.Engine, ocrResult.Confidence, ocrResult.ProcessingTime, len(base64Images))),
	}

	// Add all the images
	for _, imageB64 := range base64Images {
		content = append(content, mcp.NewImageContent(
			imageB64,
			"image/png",
		))
	}

	return &mcp.CallToolResult{
		Content: content,
	}, nil
}

// Resource handlers

func (ps *PDFServer) ListDocuments(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// List all PDF files in the folder
	query := fmt.Sprintf("mimeType='application/pdf' and trashed=false and '%s' in parents", ps.folderID)

	files, err := ps.driveService.Files.List().
		Q(query).
		PageSize(100).
		Spaces("drive").
		Fields("files(id, name, parents, createdTime, modifiedTime, size, webViewLink, thumbnailLink)").
		Do()
	if err != nil {
		return nil, fmt.Errorf("error listing PDF files: %w", err)
	}

	// Create document resources
	var resources []mcp.ResourceContents
	for _, file := range files.Files {
		docResource := map[string]interface{}{
			"id":            file.Id,
			"name":          file.Name,
			"size":          file.Size,
			"created":       file.CreatedTime,
			"modified":      file.ModifiedTime,
			"webViewLink":   file.WebViewLink,
			"thumbnailLink": file.ThumbnailLink,
			"uri":           fmt.Sprintf("pdf://documents/%s", file.Id),
		}

		docJSON, _ := json.MarshalIndent(docResource, "", "  ")
		resources = append(resources, mcp.TextResourceContents{
			URI:      fmt.Sprintf("pdf://documents/%s", file.Id),
			MIMEType: "application/json",
			Text:     string(docJSON),
		})
	}

	return resources, nil
}

// Helper function to get PDF content as bytes
func (ps *PDFServer) getPDFContentBytes(fileID string) ([]byte, error) {
	resp, err := ps.driveService.Files.Get(fileID).Download()
	if err != nil {
		return nil, fmt.Errorf("error downloading PDF: %w", err)
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
	return pdfData, nil
}

