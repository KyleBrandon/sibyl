package pdfmcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// OCREngine defines the interface for OCR implementations
type OCREngine interface {
	ExtractText(ctx context.Context, imageData []byte) (*OCRResult, error)
	ExtractStructuredText(ctx context.Context, imageData []byte, documentType string) (*StructuredOCRResult, error)
	ProcessPDF(ctx context.Context, pdfData []byte) (*OCRResult, error)
	GetEngineInfo() OCREngineInfo
}

// OCRResult represents the result of basic text extraction
type OCRResult struct {
	Text           string        `json:"text"`
	Confidence     float64       `json:"confidence"`
	Language       string        `json:"language"`
	ProcessingTime time.Duration `json:"processing_time"`
	Engine         string        `json:"engine"`
}

// StructuredOCRResult represents the result of structured text extraction
type StructuredOCRResult struct {
	OCRResult
	Blocks []TextBlock `json:"blocks"`
	Tables []Table     `json:"tables,omitempty"`
	Layout LayoutInfo  `json:"layout"`
}

// TextBlock represents a block of text with positioning
type TextBlock struct {
	Text        string      `json:"text"`
	Confidence  float64     `json:"confidence"`
	BoundingBox BoundingBox `json:"bounding_box"`
	Type        string      `json:"type"` // "paragraph", "line", "word"
}

// Table represents a detected table structure
type Table struct {
	Rows        []TableRow  `json:"rows"`
	BoundingBox BoundingBox `json:"bounding_box"`
	Confidence  float64     `json:"confidence"`
}

// TableRow represents a row in a table
type TableRow struct {
	Cells []TableCell `json:"cells"`
}

// TableCell represents a cell in a table
type TableCell struct {
	Text        string      `json:"text"`
	BoundingBox BoundingBox `json:"bounding_box"`
	ColumnSpan  int         `json:"column_span"`
	RowSpan     int         `json:"row_span"`
}

// BoundingBox represents the position of text elements
type BoundingBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// LayoutInfo provides information about document layout
type LayoutInfo struct {
	PageWidth   int    `json:"page_width"`
	PageHeight  int    `json:"page_height"`
	Orientation string `json:"orientation"` // "portrait", "landscape"
	ColumnCount int    `json:"column_count"`
	HasTables   bool   `json:"has_tables"`
	HasDiagrams bool   `json:"has_diagrams"`
}

// OCREngineInfo provides information about the OCR engine
type OCREngineInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Languages    []string `json:"supported_languages"`
	Features     []string `json:"features"`
	IsLocal      bool     `json:"is_local"`
	RequiresAuth bool     `json:"requires_auth"`
}

// Mathpix API constants
const (
	MathpixPdfApiURL    = "https://api.mathpix.com/v3/pdf"
	MathpixPollInterval = 5 * time.Second
)

// Mathpix API response types
type MathpixErrorInfo struct {
	ID      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}

type MathpixUploadResponse struct {
	PdfID     string           `json:"pdf_id"`
	Error     string           `json:"error,omitempty"`
	ErrorInfo MathpixErrorInfo `json:"error_info,omitempty"`
}

type MathpixPollResponse struct {
	Status      string `json:"status"`
	PdfMarkdown string `json:"pdf_md,omitempty"`
}

// MathpixOCR implements OCR using Mathpix API
type MathpixOCR struct {
	appID     string
	appKey    string
	languages []string
}

// NewMathpixOCR creates a new Mathpix OCR engine
func NewMathpixOCR(appID, appKey string, languages []string) *MathpixOCR {
	if len(languages) == 0 {
		languages = []string{"en"}
	}
	return &MathpixOCR{
		appID:     appID,
		appKey:    appKey,
		languages: languages,
	}
}

func (m *MathpixOCR) ExtractText(ctx context.Context, imageData []byte) (*OCRResult, error) {
	start := time.Now()

	// For single images, we'll use the PDF API with the image data
	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "image.png")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy image data to form
	_, err = part.Write(imageData)
	if err != nil {
		return nil, fmt.Errorf("failed to write image data: %w", err)
	}
	writer.Close()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", MathpixPdfApiURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("app_id", m.appID)
	req.Header.Set("app_key", m.appKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Parse upload response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var uploadResp MathpixUploadResponse
	err = json.Unmarshal(respBody, &uploadResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if uploadResp.Error != "" {
		return nil, fmt.Errorf("mathpix error: %s - %s", uploadResp.Error, uploadResp.ErrorInfo.Message)
	}

	// Poll for results
	markdown, err := m.pollForResults(ctx, uploadResp.PdfID)
	if err != nil {
		return nil, fmt.Errorf("failed to get results: %w", err)
	}

	return &OCRResult{
		Text:           markdown,
		Confidence:     0.95, // Mathpix generally has high confidence
		Language:       strings.Join(m.languages, ","),
		ProcessingTime: time.Since(start),
		Engine:         "mathpix",
	}, nil
}

func (m *MathpixOCR) ExtractStructuredText(ctx context.Context, imageData []byte, documentType string) (*StructuredOCRResult, error) {
	// Get basic text first
	basicResult, err := m.ExtractText(ctx, imageData)
	if err != nil {
		return nil, err
	}

	// Parse markdown into structured blocks
	blocks := m.parseMarkdownToBlocks(basicResult.Text)

	return &StructuredOCRResult{
		OCRResult: *basicResult,
		Blocks:    blocks,
		Layout: LayoutInfo{
			PageWidth:   800, // Default values
			PageHeight:  1000,
			Orientation: "portrait",
			ColumnCount: 1,
			HasTables:   strings.Contains(basicResult.Text, "|"),  // Simple table detection
			HasDiagrams: strings.Contains(basicResult.Text, "$$"), // Math/diagram detection
		},
	}, nil
}

func (m *MathpixOCR) GetEngineInfo() OCREngineInfo {
	return OCREngineInfo{
		Name:         "Mathpix",
		Version:      "v3",
		Languages:    m.languages,
		Features:     []string{"text_extraction", "math_recognition", "table_extraction", "high_accuracy"},
		IsLocal:      false,
		RequiresAuth: true,
	}
}

func (m *MathpixOCR) pollForResults(ctx context.Context, pdfID string) (string, error) {
	pollURL := fmt.Sprintf("%s/%s", MathpixPdfApiURL, pdfID)

	// Poll with timeout
	timeout := time.After(5 * time.Minute) // 5 minute timeout
	ticker := time.NewTicker(MathpixPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for mathpix results")
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, "GET", pollURL, nil)
			if err != nil {
				return "", fmt.Errorf("failed to create poll request: %w", err)
			}

			req.Header.Set("app_id", m.appID)
			req.Header.Set("app_key", m.appKey)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return "", fmt.Errorf("failed to poll status: %w", err)
			}

			if resp.StatusCode > 299 {
				resp.Body.Close()
				return "", fmt.Errorf("poll request failed with status %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return "", fmt.Errorf("failed to read poll response: %w", err)
			}

			var pollResp MathpixPollResponse
			err = json.Unmarshal(body, &pollResp)
			if err != nil {
				return "", fmt.Errorf("failed to unmarshal poll response: %w", err)
			}

			switch pollResp.Status {
			case "completed":
				// Get the actual results
				return m.getConversionResults(ctx, pdfID)
			case "error":
				return "", fmt.Errorf("mathpix processing failed")
			case "processing":
				// Continue polling
				continue
			default:
				slog.Debug("Mathpix status", "status", pollResp.Status)
				continue
			}
		}
	}
}

func (m *MathpixOCR) getConversionResults(ctx context.Context, pdfID string) (string, error) {
	resultsURL := fmt.Sprintf("%s/%s.md", MathpixPdfApiURL, pdfID)

	req, err := http.NewRequestWithContext(ctx, "GET", resultsURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create results request: %w", err)
	}

	req.Header.Set("app_id", m.appID)
	req.Header.Set("app_key", m.appKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return "", fmt.Errorf("results request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read results: %w", err)
	}

	return string(body), nil
}

func (m *MathpixOCR) parseMarkdownToBlocks(markdown string) []TextBlock {
	lines := strings.Split(markdown, "\n")
	blocks := make([]TextBlock, 0)

	y := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			y += 20
			continue
		}

		blockType := "paragraph"
		confidence := 0.95

		// Detect different markdown elements
		if strings.HasPrefix(line, "#") {
			blockType = "heading"
		} else if strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|") {
			blockType = "table_row"
		} else if strings.HasPrefix(line, "$$") || strings.HasSuffix(line, "$$") {
			blockType = "math"
		}

		blocks = append(blocks, TextBlock{
			Text:       line,
			Confidence: confidence,
			BoundingBox: BoundingBox{
				X:      0,
				Y:      y,
				Width:  len(line) * 8, // Approximate
				Height: 20,
			},
			Type: blockType,
		})

		y += 25
	}

	return blocks
}

// ProcessPDF processes a PDF file directly with Mathpix API
func (m *MathpixOCR) ProcessPDF(ctx context.Context, pdfData []byte) (*OCRResult, error) {
	start := time.Now()

	// Create multipart form data for PDF
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	// Add PDF file to form
	part, err := writer.CreateFormFile("file", "document.pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	
	_, err = part.Write(pdfData)
	if err != nil {
		return nil, fmt.Errorf("failed to write PDF data: %w", err)
	}
	
	// Add conversion options
	err = writer.WriteField("options_json", `{"conversion_formats": {"md": true}}`)
	if err != nil {
		return nil, fmt.Errorf("failed to write options field: %w", err)
	}
	writer.Close()

	// Create HTTP request to Mathpix PDF API
	req, err := http.NewRequestWithContext(ctx, "POST", MathpixPdfApiURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("app_id", m.appID)
	req.Header.Set("app_key", m.appKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mathpix API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse initial response to get PDF ID
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var initialResp MathpixUploadResponse
	err = json.Unmarshal(respBody, &initialResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Poll for completion and get results
	markdownText, err := m.pollForResults(ctx, initialResp.PdfID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversion results: %w", err)
	}

	processingTime := time.Since(start)

	return &OCRResult{
		Text:           markdownText,
		Confidence:     0.95, // Mathpix generally has high confidence
		Language:       "en",
		ProcessingTime: processingTime,
		Engine:         "mathpix",
	}, nil
}

// MockOCR implements a mock OCR engine for testing and fallback
type MockOCR struct {
	languages []string
}

// NewMockOCR creates a new mock OCR engine
func NewMockOCR(languages []string) *MockOCR {
	if len(languages) == 0 {
		languages = []string{"eng"}
	}
	return &MockOCR{
		languages: languages,
	}
}

func (m *MockOCR) ExtractText(ctx context.Context, imageData []byte) (*OCRResult, error) {
	start := time.Now()

	// Mock OCR that returns placeholder text
	mockText := "This is mock OCR text extracted from the image. In a real implementation, this would be the actual text content from the PDF page."

	return &OCRResult{
		Text:           mockText,
		Confidence:     0.85,
		Language:       strings.Join(m.languages, ","),
		ProcessingTime: time.Since(start),
		Engine:         "mock",
	}, nil
}

func (m *MockOCR) ExtractStructuredText(ctx context.Context, imageData []byte, documentType string) (*StructuredOCRResult, error) {
	basicResult, err := m.ExtractText(ctx, imageData)
	if err != nil {
		return nil, err
	}

	// Create mock structured blocks
	blocks := []TextBlock{
		{
			Text:        "Mock Title",
			Confidence:  0.9,
			BoundingBox: BoundingBox{X: 50, Y: 50, Width: 300, Height: 30},
			Type:        "title",
		},
		{
			Text:        "Mock paragraph content with multiple lines of text that would be extracted from the document.",
			Confidence:  0.85,
			BoundingBox: BoundingBox{X: 50, Y: 100, Width: 400, Height: 60},
			Type:        "paragraph",
		},
	}

	return &StructuredOCRResult{
		OCRResult: *basicResult,
		Blocks:    blocks,
		Layout: LayoutInfo{
			PageWidth:   600,
			PageHeight:  800,
			Orientation: "portrait",
			ColumnCount: 1,
			HasTables:   false,
			HasDiagrams: false,
		},
	}, nil
}

func (m *MockOCR) ProcessPDF(ctx context.Context, pdfData []byte) (*OCRResult, error) {
	// Mock implementation - just return simple mock content
	mockText := `# Mock PDF Conversion

This is a mock conversion of a PDF document for testing purposes.

## Content

The PDF contained text that has been extracted and converted to Markdown format.

- Item 1
- Item 2
- Item 3

Mock processing complete.`

	return &OCRResult{
		Text:           mockText,
		Confidence:     0.80,
		Language:       "en",
		ProcessingTime: time.Millisecond * 100, // Simulate fast processing
		Engine:         "mock",
	}, nil
}

func (m *MockOCR) GetEngineInfo() OCREngineInfo {
	return OCREngineInfo{
		Name:         "Mock OCR",
		Version:      "1.0",
		Languages:    m.languages,
		Features:     []string{"text_extraction", "basic_layout", "testing"},
		IsLocal:      true,
		RequiresAuth: false,
	}
}

// OCRManager manages multiple OCR engines and provides smart selection
type OCRManager struct {
	engines       map[string]OCREngine
	defaultEngine string
}

// NewOCRManager creates a new OCR manager
func NewOCRManager() *OCRManager {
	return &OCRManager{
		engines: make(map[string]OCREngine),
	}
}

// RegisterEngine registers an OCR engine
func (m *OCRManager) RegisterEngine(name string, engine OCREngine) {
	m.engines[name] = engine
	if m.defaultEngine == "" {
		m.defaultEngine = name
	}
}

// SetDefaultEngine sets the default OCR engine
func (m *OCRManager) SetDefaultEngine(name string) error {
	if _, exists := m.engines[name]; !exists {
		return fmt.Errorf("engine %s not registered", name)
	}
	m.defaultEngine = name
	return nil
}

// GetEngine returns an OCR engine by name
func (m *OCRManager) GetEngine(name string) (OCREngine, error) {
	if name == "" {
		name = m.defaultEngine
	}

	engine, exists := m.engines[name]
	if !exists {
		return nil, fmt.Errorf("engine %s not found", name)
	}

	return engine, nil
}

// ListEngines returns information about all registered engines
func (m *OCRManager) ListEngines() map[string]OCREngineInfo {
	info := make(map[string]OCREngineInfo)
	for name, engine := range m.engines {
		info[name] = engine.GetEngineInfo()
	}
	return info
}

// SuggestEngine suggests the best OCR engine for a document type
func (m *OCRManager) SuggestEngine(documentType string, imageSize int) string {
	// For any document type, prefer Mathpix if available
	if _, exists := m.engines["mathpix"]; exists {
		return "mathpix"
	}

	// Fallback to any available engine
	if m.defaultEngine != "" {
		return m.defaultEngine
	}

	// Last resort: return first available engine
	for name := range m.engines {
		return name
	}

	return "mock" // Ultimate fallback
}

// ExtractTextWithBestEngine automatically selects and uses the best OCR engine
func (m *OCRManager) ExtractTextWithBestEngine(ctx context.Context, imageData []byte, documentType string) (*OCRResult, error) {
	engineName := m.SuggestEngine(documentType, len(imageData))
	engine, err := m.GetEngine(engineName)
	if err != nil {
		return nil, err
	}

	slog.Info("Using OCR engine", "engine", engineName, "document_type", documentType, "image_size", len(imageData))

	return engine.ExtractText(ctx, imageData)
}

// ExtractStructuredTextWithBestEngine automatically selects and uses the best OCR engine for structured extraction
func (m *OCRManager) ExtractStructuredTextWithBestEngine(ctx context.Context, imageData []byte, documentType string) (*StructuredOCRResult, error) {
	engineName := m.SuggestEngine(documentType, len(imageData))
	engine, err := m.GetEngine(engineName)
	if err != nil {
		return nil, err
	}

	slog.Info("Using OCR engine for structured extraction", "engine", engineName, "document_type", documentType, "image_size", len(imageData))

	return engine.ExtractStructuredText(ctx, imageData, documentType)
}
