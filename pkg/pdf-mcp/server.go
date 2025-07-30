// Package pdf_mcp provides MCP server for PDF processing and conversion
package pdf_mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

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
	prompts      *PromptManager
}

// Request types for MCP tools
type SearchPDFsRequest struct {
	Query    string `json:"query" mcp:"Search query for PDF files"`
	MaxFiles int    `json:"max_files,omitempty" mcp:"Maximum number of files to return (default: 10)"`
}

type GetPDFContentRequest struct {
	FileID string `json:"file_id" mcp:"Google Drive file ID of the PDF"`
}

type ConvertPDFToImagesRequest struct {
	FileID string `json:"file_id" mcp:"Google Drive file ID of the PDF to convert"`
	DPI    int    `json:"dpi,omitempty" mcp:"DPI for image conversion (default: 150)"`
}

type GetPromptsRequest struct {
	DocumentType string `json:"document_type,omitempty" mcp:"Type of document: handwritten, typed, mixed, or all"`
}

func NewPDFServer(ctx context.Context, credentialsPath, folderID string) (*PDFServer, error) {
	s := &PDFServer{}

	// Initialize Google Drive service
	driveService, err := drive.NewService(
		ctx,
		option.WithCredentialsFile(credentialsPath),
		option.WithScopes(drive.DriveScope))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	// Initialize prompt manager
	prompts, err := NewPromptManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize prompt manager: %w", err)
	}

	s.ctx = ctx
	s.driveService = driveService
	s.folderID = folderID
	s.prompts = prompts
	s.McpServer = server.NewMCPServer("pdf-server", "v1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(false, false))
	s.addTools()
	s.addResources()

	return s, nil
}

func (ps *PDFServer) addResources() {
	// Resource 1: PDF Documents collection
	documentsResource := mcp.NewResource(
		"pdf://documents/",
		"PDF Documents",
		mcp.WithResourceDescription("Collection of PDF documents in Google Drive folder"),
		mcp.WithMIMEType("application/json"),
	)
	ps.McpServer.AddResource(documentsResource, ps.ListDocuments)

	// Resource 2: Conversion Templates
	templatesResource := mcp.NewResource(
		"pdf://templates/",
		"Conversion Templates",
		mcp.WithResourceDescription("Available PDF to Markdown conversion templates"),
		mcp.WithMIMEType("application/json"),
	)
	ps.McpServer.AddResource(templatesResource, ps.ListTemplates)
}
func (ps *PDFServer) addTools() {
	// Tool 1: Search for PDF files
	searchTool := mcp.NewTool(
		"search_pdfs",
		mcp.WithDescription("Search for PDF files in Google Drive"),
		mcp.WithString("query", mcp.Description("Search query for PDF files"), mcp.Required()),
		mcp.WithNumber("max_files", mcp.Description("Maximum number of files to return (default: 10)")),
	)
	ps.McpServer.AddTool(searchTool, mcp.NewTypedToolHandler(ps.SearchPDFs))

	// Tool 2: Get PDF content and metadata
	getContentTool := mcp.NewTool(
		"get_pdf_content",
		mcp.WithDescription("Get PDF file content and metadata from Google Drive"),
		mcp.WithString("file_id", mcp.Description("Google Drive file ID"), mcp.Required()),
	)
	ps.McpServer.AddTool(getContentTool, mcp.NewTypedToolHandler(ps.GetPDFContent))

	// Tool 3: Convert PDF to images
	convertTool := mcp.NewTool(
		"convert_pdf_to_images",
		mcp.WithDescription("Convert PDF file to images for LLM processing"),
		mcp.WithString("file_id", mcp.Description("Google Drive file ID"), mcp.Required()),
		mcp.WithNumber("dpi", mcp.Description("DPI for image conversion (default: 150)")),
	)
	ps.McpServer.AddTool(convertTool, mcp.NewTypedToolHandler(ps.ConvertPDFToImages))

	// Tool 4: Get conversion prompts
	promptsTool := mcp.NewTool(
		"get_conversion_prompts",
		mcp.WithDescription("Get prompt templates for PDF to Markdown conversion"),
		mcp.WithString("document_type", mcp.Description("Document type: handwritten, typed, mixed, or all")),
	)
	ps.McpServer.AddTool(promptsTool, mcp.NewTypedToolHandler(ps.GetConversionPrompts))

	// Tool 5: Suggest conversion approach
	suggestTool := mcp.NewTool(
		"suggest_conversion_approach",
		mcp.WithDescription("Analyze PDF and suggest best conversion approach"),
		mcp.WithString("file_id", mcp.Description("Google Drive file ID"), mcp.Required()),
	)
	ps.McpServer.AddTool(suggestTool, mcp.NewTypedToolHandler(ps.SuggestConversionApproach))
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

func (ps *PDFServer) GetPDFContent(ctx context.Context, request mcp.CallToolRequest, params GetPDFContentRequest) (*mcp.CallToolResult, error) {
	// Get file metadata
	file, err := ps.driveService.Files.
		Get(params.FileID).
		Fields("id", "name", "mimeType", "size", "modifiedTime", "webViewLink").
		Do()
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Error getting file metadata: %v", err)),
			},
		}, nil
	}

	// Download file content
	resp, err := ps.driveService.Files.Get(params.FileID).Download()
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Error downloading file: %v", err)),
			},
		}, nil
	}
	defer resp.Body.Close()

	// Read content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Error reading file content: %v", err)),
			},
		}, nil
	}

	// Return as embedded resource
	fileResource := mcp.BlobResourceContents{
		URI:      file.WebViewLink,
		MIMEType: file.MimeType,
		Blob:     base64.StdEncoding.EncodeToString(content),
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewEmbeddedResource(fileResource),
		},
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

func (ps *PDFServer) ListTemplates(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Get all available templates from the prompt manager
	allPrompts := ps.prompts.GetAllPrompts()
	templates := make(map[string]interface{})

	for promptType, template := range allPrompts {
		templates[promptType] = map[string]interface{}{
			"name":        template.Name,
			"description": template.Description,
			"uri":         fmt.Sprintf("pdf://templates/%s", promptType),
			"type":        template.Type,
			"use_case":    template.UseCase,
			"prompt":      template.Prompt,
		}
	}

	templatesJSON, _ := json.MarshalIndent(templates, "", "  ")

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "pdf://templates/",
			MIMEType: "application/json",
			Text:     string(templatesJSON),
		},
	}, nil
}
