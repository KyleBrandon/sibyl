// Package pdf_mcp provides MCP server for PDF processing and conversion
package pdf_mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

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
	ocrManager   *OCRManager
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

// New OCR-related request types
type ExtractTextRequest struct {
	FileID string `json:"file_id" mcp:"Google Drive file ID of the PDF"`
	Engine string `json:"engine,omitempty" mcp:"OCR engine to use: mathpix, mock, or auto"`
}

type ExtractStructuredTextRequest struct {
	FileID       string `json:"file_id" mcp:"Google Drive file ID of the PDF"`
	DocumentType string `json:"document_type,omitempty" mcp:"Type of document for better OCR: handwritten, typed, mixed, research"`
	Engine       string `json:"engine,omitempty" mcp:"OCR engine to use: mathpix, mock, or auto"`
}

type ConvertPDFHybridRequest struct {
	FileID       string `json:"file_id" mcp:"Google Drive file ID of the PDF to convert"`
	DocumentType string `json:"document_type,omitempty" mcp:"Type of document: handwritten, typed, mixed, research"`
	DPI          int    `json:"dpi,omitempty" mcp:"DPI for image conversion (default: 150)"`
	Engine       string `json:"engine,omitempty" mcp:"OCR engine to use: mathpix, mock, or auto"`
}

type AnalyzeDocumentRequest struct {
	FileID string `json:"file_id" mcp:"Google Drive file ID of the PDF to analyze"`
}

type ListOCREnginesRequest struct {
	// No parameters needed
}

// OCRConfig holds OCR-related configuration
type OCRConfig struct {
	DefaultEngine string   `json:"default_engine"`
	Languages     []string `json:"languages"`
	MathpixAppID  string   `json:"mathpix_app_id,omitempty"`
	MathpixAppKey string   `json:"mathpix_app_key,omitempty"`
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

	// Initialize prompt manager
	prompts, err := NewPromptManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize prompt manager: %w", err)
	}

	// Initialize OCR manager
	ocrManager := NewOCRManager()

	// Register available OCR engines based on configuration
	// Register Mathpix if credentials are provided
	if ocrConfig.MathpixAppID != "" && ocrConfig.MathpixAppKey != "" {
		mathpix := NewMathpixOCR(ocrConfig.MathpixAppID, ocrConfig.MathpixAppKey, ocrConfig.Languages)
		ocrManager.RegisterEngine("mathpix", mathpix)
	}

	// Register mock OCR as fallback
	mockOCR := NewMockOCR(ocrConfig.Languages)
	ocrManager.RegisterEngine("mock", mockOCR)

	// Set default engine based on configuration
	if err := ocrManager.SetDefaultEngine(ocrConfig.DefaultEngine); err != nil {
		// Fallback to mathpix, then mock
		if err := ocrManager.SetDefaultEngine("mathpix"); err != nil {
			ocrManager.SetDefaultEngine("mock")
		}
	}

	s.ctx = ctx
	s.driveService = driveService
	s.folderID = folderID
	s.prompts = prompts
	s.ocrManager = ocrManager
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

	// Tool 6: Extract text using OCR
	extractTextTool := mcp.NewTool(
		"extract_text_from_pdf",
		mcp.WithDescription("Extract text from PDF using OCR"),
		mcp.WithString("file_id", mcp.Description("Google Drive file ID"), mcp.Required()),
		mcp.WithString("engine", mcp.Description("OCR engine: mathpix, mock, or auto")),
	)
	ps.McpServer.AddTool(extractTextTool, mcp.NewTypedToolHandler(ps.ExtractText))

	// Tool 7: Extract structured text using OCR
	extractStructuredTool := mcp.NewTool(
		"extract_structured_text",
		mcp.WithDescription("Extract structured text from PDF using OCR with layout information"),
		mcp.WithString("file_id", mcp.Description("Google Drive file ID"), mcp.Required()),
		mcp.WithString("document_type", mcp.Description("Document type: handwritten, typed, mixed, research")),
		mcp.WithString("engine", mcp.Description("OCR engine: mathpix, mock, or auto")),
	)
	ps.McpServer.AddTool(extractStructuredTool, mcp.NewTypedToolHandler(ps.ExtractStructuredText))

	// Tool 8: Hybrid conversion (OCR + Vision)
	hybridTool := mcp.NewTool(
		"convert_pdf_hybrid",
		mcp.WithDescription("Convert PDF using hybrid OCR + Vision approach for best results"),
		mcp.WithString("file_id", mcp.Description("Google Drive file ID"), mcp.Required()),
		mcp.WithString("document_type", mcp.Description("Document type: handwritten, typed, mixed, research")),
		mcp.WithNumber("dpi", mcp.Description("DPI for image conversion (default: 150)")),
		mcp.WithString("engine", mcp.Description("OCR engine: mathpix, mock, or auto")),
	)
	ps.McpServer.AddTool(hybridTool, mcp.NewTypedToolHandler(ps.ConvertPDFHybrid))

	// Tool 9: Analyze document for OCR recommendations
	analyzeTool := mcp.NewTool(
		"analyze_document",
		mcp.WithDescription("Analyze PDF document and recommend best OCR approach"),
		mcp.WithString("file_id", mcp.Description("Google Drive file ID"), mcp.Required()),
	)
	ps.McpServer.AddTool(analyzeTool, mcp.NewTypedToolHandler(ps.AnalyzeDocument))

	// Tool 10: List available OCR engines
	listEnginesTool := mcp.NewTool(
		"list_ocr_engines",
		mcp.WithDescription("List available OCR engines and their capabilities"),
	)
	ps.McpServer.AddTool(listEnginesTool, mcp.NewTypedToolHandler(ps.ListOCREngines))
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

// Helper function to convert base64 images to byte arrays
func (ps *PDFServer) convertBase64ImagesToBytes(base64Images []string) ([][]byte, error) {
	images := make([][]byte, len(base64Images))
	for i, b64 := range base64Images {
		imageData, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode image %d: %w", i, err)
		}
		images[i] = imageData
	}
	return images, nil
}

// ExtractText extracts text from PDF using OCR
func (ps *PDFServer) ExtractText(ctx context.Context, request mcp.CallToolRequest, params ExtractTextRequest) (*mcp.CallToolResult, error) {
	// Get PDF content first
	pdfContent, err := ps.getPDFContentBytes(params.FileID)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Failed to get PDF content: %v", err)),
			},
		}, nil
	}

	// Convert PDF to images (returns base64 strings)
	base64Images, err := ps.convertPDFToImages(pdfContent, 150) // Default DPI
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

	// Convert base64 images to byte arrays
	images, err := ps.convertBase64ImagesToBytes(base64Images)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Failed to decode images: %v", err)),
			},
		}, nil
	}

	// Extract text from first page using OCR
	var engine OCREngine
	var engineErr error

	if params.Engine == "" || params.Engine == "auto" {
		engine, engineErr = ps.ocrManager.GetEngine("")
	} else {
		engine, engineErr = ps.ocrManager.GetEngine(params.Engine)
	}

	if engineErr != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("OCR engine error: %v", engineErr)),
			},
		}, nil
	}

	// Extract text from all pages
	var allText strings.Builder
	for i, imageData := range images {
		result, err := engine.ExtractText(ctx, imageData)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("OCR failed on page %d: %v", i+1, err)),
				},
			}, nil
		}

		allText.WriteString(fmt.Sprintf("=== Page %d ===\n", i+1))
		allText.WriteString(result.Text)
		allText.WriteString("\n\n")
	}

	// Return OCR results
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("OCR Text Extraction Results:\n\n%s", allText.String())),
		},
	}, nil
}

// ExtractStructuredText extracts structured text from PDF using OCR
func (ps *PDFServer) ExtractStructuredText(ctx context.Context, request mcp.CallToolRequest, params ExtractStructuredTextRequest) (*mcp.CallToolResult, error) {
	// Get PDF content first
	pdfContent, err := ps.getPDFContentBytes(params.FileID)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Failed to get PDF content: %v", err)),
			},
		}, nil
	}

	// Convert PDF to images
	base64Images, err := ps.convertPDFToImages(pdfContent, 150)
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

	// Convert base64 images to byte arrays
	images, err := ps.convertBase64ImagesToBytes(base64Images)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Failed to decode images: %v", err)),
			},
		}, nil
	}

	// Get OCR engine
	var engine OCREngine
	var engineErr error

	if params.Engine == "" || params.Engine == "auto" {
		// Use document type to suggest best engine
		engineName := ps.ocrManager.SuggestEngine(params.DocumentType, len(images[0]))
		engine, engineErr = ps.ocrManager.GetEngine(engineName)
	} else {
		engine, engineErr = ps.ocrManager.GetEngine(params.Engine)
	}

	if engineErr != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("OCR engine error: %v", engineErr)),
			},
		}, nil
	}

	// Extract structured text from all pages
	var results []StructuredOCRResult
	for i, imageData := range images {
		result, err := engine.ExtractStructuredText(ctx, imageData, params.DocumentType)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Structured OCR failed on page %d: %v", i+1, err)),
				},
			}, nil
		}
		results = append(results, *result)
	}

	// Format results as JSON
	resultsJSON, _ := json.MarshalIndent(results, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Structured OCR Results:\n\n%s", string(resultsJSON))),
		},
	}, nil
}

// ConvertPDFHybrid performs hybrid OCR + Vision conversion
func (ps *PDFServer) ConvertPDFHybrid(ctx context.Context, request mcp.CallToolRequest, params ConvertPDFHybridRequest) (*mcp.CallToolResult, error) {
	// Get PDF content first
	pdfContent, err := ps.getPDFContentBytes(params.FileID)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Failed to get PDF content: %v", err)),
			},
		}, nil
	}

	// Set default DPI
	dpi := params.DPI
	if dpi <= 0 {
		dpi = 150
	}

	// Convert PDF to images
	base64Images, err := ps.convertPDFToImages(pdfContent, float64(dpi))
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

	// Convert base64 images to byte arrays
	images, err := ps.convertBase64ImagesToBytes(base64Images)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Failed to decode images: %v", err)),
			},
		}, nil
	}

	// Get OCR engine
	var engine OCREngine
	var engineErr error

	if params.Engine == "" || params.Engine == "auto" {
		engineName := ps.ocrManager.SuggestEngine(params.DocumentType, len(images[0]))
		engine, engineErr = ps.ocrManager.GetEngine(engineName)
	} else {
		engine, engineErr = ps.ocrManager.GetEngine(params.Engine)
	}

	if engineErr != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("OCR engine error: %v", engineErr)),
			},
		}, nil
	}

	// Extract text from first page for demonstration
	ocrResult, err := engine.ExtractText(ctx, images[0])
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("OCR failed: %v", err)),
			},
		}, nil
	}

	// Get base prompt for document type
	basePromptTemplate, err := ps.prompts.GetPrompt(params.DocumentType)
	var basePrompt string
	if err != nil {
		// Fallback to typed document prompt
		fallbackTemplate, fallbackErr := ps.prompts.GetPrompt("typed")
		if fallbackErr != nil {
			basePrompt = "Convert this document to well-structured Markdown format."
		} else {
			basePrompt = fallbackTemplate.Prompt
		}
	} else {
		basePrompt = basePromptTemplate.Prompt
	}
	// Build enhanced hybrid prompt
	hybridPrompt := fmt.Sprintf(`%s

=== HYBRID OCR + VISION PROCESSING ===

OCR EXTRACTED TEXT (Confidence: %.2f):
%s

INSTRUCTIONS FOR LLM:
1. Use the OCR text above as your primary source for accuracy
2. Refer to the images below to:
   - Correct any OCR errors you notice
   - Understand visual layout, emphasis, and formatting
   - Capture information that OCR might have missed (diagrams, special formatting)
3. Combine both sources for the most accurate and complete conversion
4. If OCR confidence is low (< 0.8), rely more heavily on visual analysis
5. Preserve the document structure and formatting as much as possible

OCR Engine Used: %s
Document Type: %s
Processing Time: %v
`, basePrompt, ocrResult.Confidence, ocrResult.Text, ocrResult.Engine, params.DocumentType, ocrResult.ProcessingTime)

	// Prepare content with OCR results and images
	content := []mcp.Content{
		mcp.NewTextContent(hybridPrompt),
	}

	// Add images
	for i, imageData := range images {
		content = append(content, mcp.NewImageContent(
			base64.StdEncoding.EncodeToString(imageData),
			fmt.Sprintf("image/png"),
		))

		// Limit to first few pages to avoid overwhelming the LLM
		if i >= 4 { // Max 5 pages
			content = append(content, mcp.NewTextContent(fmt.Sprintf("... and %d more pages", len(images)-i-1)))
			break
		}
	}

	return &mcp.CallToolResult{
		Content: content,
	}, nil
}

// AnalyzeDocument analyzes a PDF and recommends the best OCR approach
func (ps *PDFServer) AnalyzeDocument(ctx context.Context, request mcp.CallToolRequest, params AnalyzeDocumentRequest) (*mcp.CallToolResult, error) {
	// Get PDF content first
	pdfContent, err := ps.getPDFContentBytes(params.FileID)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Failed to get PDF content: %v", err)),
			},
		}, nil
	}

	// Convert first page to image for analysis
	base64Images, err := ps.convertPDFToImages(pdfContent, 150)
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

	// Convert base64 images to byte arrays for analysis
	images, err := ps.convertBase64ImagesToBytes(base64Images)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Failed to decode images: %v", err)),
			},
		}, nil
	}

	// Analyze document characteristics
	imageSize := len(images[0])
	pageCount := len(images)

	// Simple heuristics for document type detection
	var documentType string
	var confidence float64

	if imageSize > 2*1024*1024 { // Large images might be scanned documents
		documentType = "mixed"
		confidence = 0.7
	} else if pageCount > 10 { // Many pages likely typed documents
		documentType = "typed"
		confidence = 0.8
	} else {
		documentType = "mixed" // Default to mixed for safety
		confidence = 0.6
	}

	// Get recommended engine
	recommendedEngine := ps.ocrManager.SuggestEngine(documentType, imageSize)

	// Get available engines
	engines := ps.ocrManager.ListEngines()

	analysis := map[string]interface{}{
		"document_analysis": map[string]interface{}{
			"page_count":         pageCount,
			"average_image_size": imageSize,
			"detected_type":      documentType,
			"confidence":         confidence,
		},
		"recommendations": map[string]interface{}{
			"document_type":      documentType,
			"recommended_engine": recommendedEngine,
			"processing_approach": map[string]string{
				"best_for_speed":    "mock",
				"best_for_accuracy": "mathpix",
				"recommended":       recommendedEngine,
			},
		},
		"available_engines": engines,
	}

	analysisJSON, _ := json.MarshalIndent(analysis, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Document Analysis Results:\n\n%s", string(analysisJSON))),
		},
	}, nil
}

// ListOCREngines lists available OCR engines and their capabilities
func (ps *PDFServer) ListOCREngines(ctx context.Context, request mcp.CallToolRequest, params ListOCREnginesRequest) (*mcp.CallToolResult, error) {
	engines := ps.ocrManager.ListEngines()

	enginesJSON, _ := json.MarshalIndent(engines, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Available OCR Engines:\n\n%s", string(enginesJSON))),
		},
	}, nil
}
