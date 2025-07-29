package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/KyleBrandon/sibyl/pkg/dto"
	"github.com/gen2brain/go-fitz"
	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"gopkg.in/yaml.v3"
)

// MCPHostsConfig represents the structure of .mcphost.yml
type MCPHostsConfig struct {
	MCPServers map[string]MCPServer `yaml:"mcpServers"`
}

// MCPServer represents a single MCP server configuration
type MCPServer struct {
	Type    string   `yaml:"type"`
	Command []string `yaml:"command"`
	Args    []string `yaml:"args,omitempty"`
}

// Config holds the application configuration
type Config struct {
	MCPHosts         *MCPHostsConfig
	ClaudeAPIKey     string
	SystemPromptFile string
}

// ClaudeRequest represents the structure for Claude API requests
type ClaudeRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type Content struct {
	Type   string       `json:"type"`
	Text   string       `json:"text,omitempty"`
	Source *MediaSource `json:"source,omitempty"`
}

type MediaSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type ClaudeResponse struct {
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
}

func main() {
	// Define command line flags
	mcpHostsConfig := flag.String("mcphosts-config", "", "Path to .mcphost.yml file")
	claudeAPIKey := flag.String("claude-api-key", "", "Claude API key (or set ANTHROPIC_API_KEY env var)")
	systemPromptFile := flag.String("system-prompt", "system_prompt.txt", "Path to system prompt file")
	flag.Parse()

	// Attempt to load environment file, nonfatal
	if err := godotenv.Load(); err != nil {
		slog.Warn("could not load .env file, proceeding without environment file", "error", err)
	}

	fmt.Println(*claudeAPIKey)
	// Get Anthropic Claude API key from environment if not provided
	if *claudeAPIKey == "" {
		*claudeAPIKey = os.Getenv("ANTHROPIC_API_KEY")
		fmt.Println(*claudeAPIKey)
	}

	// Load MCP hosts configuration
	mcpHosts, err := loadMCPHostsConfig(*mcpHostsConfig)
	if err != nil {
		fmt.Printf("Error loading MCP hosts configuration: %v\n", err)
		os.Exit(1)
	}

	// Validate required servers exist
	if err := validateRequiredServers(mcpHosts); err != nil {
		fmt.Printf("Configuration validation error: %v\n", err)
		os.Exit(1)
	}

	if *claudeAPIKey == "" {
		fmt.Println("Error: You must specify --claude-api-key or set ANTHROPIC_API_KEY environment variable")
		flag.Usage()
		os.Exit(1)
	}

	config := &Config{
		MCPHosts:         mcpHosts,
		ClaudeAPIKey:     *claudeAPIKey,
		SystemPromptFile: *systemPromptFile,
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := runPDFToMarkdownWorkflow(ctx, config); err != nil {
		slog.Error("Workflow failed", "error", err)
		os.Exit(1)
	}
}

func runPDFToMarkdownWorkflow(ctx context.Context, config *Config) error {
	// Initialize GCP client
	gcpClient, err := initializeMCPClient(ctx, config.MCPHosts, "gcp-server", "gcp-client")
	if err != nil {
		return fmt.Errorf("failed to initialize GCP client: %w", err)
	}

	// Initialize Notes client
	notesClient, err := initializeMCPClient(ctx, config.MCPHosts, "note-server", "notes-client")
	if err != nil {
		return fmt.Errorf("failed to initialize Notes client: %w", err)
	}

	// Step 1: Prompt user for PDF file selection
	fileID, fileName, err := promptUserForPDFFile(ctx, gcpClient)
	if err != nil {
		return fmt.Errorf("failed to select PDF file: %w", err)
	}

	// Step 2: Download the PDF file
	pdfData, err := downloadPDFFile(ctx, gcpClient, fileID)
	if err != nil {
		return fmt.Errorf("failed to download PDF file: %w", err)
	}

	// Step 3: Convert PDF to Markdown using Claude
	markdown, err := convertPDFToMarkdown(pdfData, fileName, config)
	if err != nil {
		return fmt.Errorf("failed to convert PDF to markdown: %w", err)
	}

	// Step 4: Create or merge with daily note
	err = createOrMergeDailyNote(ctx, notesClient, markdown)
	if err != nil {
		return fmt.Errorf("failed to create/merge daily note: %w", err)
	}

	fmt.Println("✅ Successfully converted PDF to Markdown and added to daily note!")
	return nil
}

// loadMCPHostsConfig loads the MCP hosts configuration from file with precedence
func loadMCPHostsConfig(configPath string) (*MCPHostsConfig, error) {
	var filePath string

	if configPath != "" {
		// Use provided path
		filePath = configPath
	} else {
		// Search in order of precedence
		candidates := []string{
			"./.mcphost.yml",
			filepath.Join(os.Getenv("HOME"), ".mcphost.yml"),
		}

		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				filePath = candidate
				break
			}
		}

		if filePath == "" {
			return nil, fmt.Errorf("no .mcphost.yml file found. Searched: %v", candidates)
		}
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	var config MCPHostsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return &config, nil
}

// validateRequiredServers ensures required servers are present in config
func validateRequiredServers(config *MCPHostsConfig) error {
	requiredServers := []string{"note-server", "gcp-server"}

	for _, serverName := range requiredServers {
		server, exists := config.MCPServers[serverName]
		if !exists {
			return fmt.Errorf("required server '%s' not found in configuration", serverName)
		}

		if server.Type != "local" {
			return fmt.Errorf("server '%s' must have type 'local', got '%s'", serverName, server.Type)
		}

		if len(server.Command) == 0 {
			return fmt.Errorf("server '%s' must have a command specified", serverName)
		}
	}

	return nil
}

func initializeMCPClient(ctx context.Context, config *MCPHostsConfig, serverName, clientName string) (*client.Client, error) {
	fmt.Printf("Initializing %s...\n", clientName)

	server, exists := config.MCPServers[serverName]
	if !exists {
		return nil, fmt.Errorf("server '%s' not found in configuration", serverName)
	}

	// Combine command and args for NewStdioMCPClient
	var fullCmd []string
	fullCmd = append(fullCmd, server.Command...)
	fullCmd = append(fullCmd, server.Args...)
	slog.Info("initializeMCPClient", "server", serverName, "args", fullCmd)

	if len(fullCmd) == 0 {
		return nil, fmt.Errorf("no command specified for server '%s'", serverName)
	}

	// First element is the command, rest are args
	var args []string
	if len(fullCmd) > 1 {
		args = fullCmd[1:]
	}

	c, err := client.NewStdioMCPClient(fullCmd[0], []string{}, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    clientName,
		Version: "1.0.0",
	}

	initResult, err := c.Initialize(ctx, initRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}

	fmt.Printf("✅ Initialized %s: %s %s\n", clientName, initResult.ServerInfo.Name, initResult.ServerInfo.Version)
	return c, nil
}

func promptUserForPDFFile(ctx context.Context, gcpClient *client.Client) (string, string, error) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Enter search query for PDF files (or press Enter to search for 'pdf'): ")
	scanner.Scan()
	query := strings.TrimSpace(scanner.Text())
	if query == "" {
		query = "pdf"
	}

	// Search for PDF files
	req := mcp.CallToolRequest{}
	req.Params.Name = "search_drive_files"
	req.Params.Arguments = map[string]any{"query": query}

	result, err := gcpClient.CallTool(ctx, req)
	if err != nil {
		return "", "", fmt.Errorf("failed to search Google Drive: %w", err)
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		return "", "", fmt.Errorf("invalid content returned from search_drive_files")
	}

	var files []dto.DriveFileResult
	if err := json.Unmarshal([]byte(textContent.Text), &files); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal files: %w", err)
	}

	if len(files) == 0 {
		return "", "", fmt.Errorf("no files found matching query: %s", query)
	}

	// Display files to user
	fmt.Println("\nFound files:")
	for i, file := range files {
		fmt.Printf("%d. %s (ID: %s)\n", i+1, file.Name, file.ID)
	}

	// Get user selection
	fmt.Print("\nEnter the number of the file to select: ")
	scanner.Scan()
	selection, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || selection < 1 || selection > len(files) {
		return "", "", fmt.Errorf("invalid selection")
	}

	selectedFile := files[selection-1]
	return selectedFile.ID, selectedFile.Name, nil
}

func downloadPDFFile(ctx context.Context, gcpClient *client.Client, fileID string) ([]byte, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = "read_drive_file"
	req.Params.Arguments = map[string]any{"file_id": fileID}

	result, err := gcpClient.CallTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to read file contents: %w", err)
	}

	resourceContent, ok := result.Content[0].(mcp.EmbeddedResource)
	if !ok {
		return nil, fmt.Errorf("no embedded resource found")
	}

	blobResource, ok := resourceContent.Resource.(mcp.BlobResourceContents)
	if !ok {
		return nil, fmt.Errorf("no blob resource found")
	}

	buffer := make([]byte, base64.StdEncoding.DecodedLen(len(blobResource.Blob)))
	n, err := base64.StdEncoding.Decode(buffer, []byte(blobResource.Blob))
	if err != nil {
		return nil, fmt.Errorf("failed to decode PDF data: %w", err)
	}

	return buffer[:n], nil
}

// convertPDFToImages converts PDF data to a slice of PNG images
func convertPDFToImages(pdfData []byte) ([][]byte, error) {
	// Create a new document from PDF data
	doc, err := fitz.NewFromMemory(pdfData)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	var images [][]byte
	pageCount := doc.NumPage()

	slog.Info("Converting PDF to images", "pages", pageCount)

	// Convert each page to PNG
	for i := 0; i < pageCount; i++ {
		// Render page as image
		img, err := doc.Image(i)
		if err != nil {
			return nil, fmt.Errorf("failed to render page %d: %w", i+1, err)
		}

		// Convert image to PNG bytes
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("failed to encode page %d as PNG: %w", i+1, err)
		}

		images = append(images, buf.Bytes())
		slog.Info("Converted page to PNG", "page", i+1, "size", buf.Len())
	}

	return images, nil
}

func convertPDFToMarkdown(pdfData []byte, fileName string, config *Config) (string, error) {
	// Load system prompt
	systemPrompt, err := loadSystemPrompt(config.SystemPromptFile)
	if err != nil {
		return "", fmt.Errorf("failed to load system prompt: %w", err)
	}

	// Convert PDF to images
	images, err := convertPDFToImages(pdfData)
	if err != nil {
		return "", fmt.Errorf("failed to convert PDF to images: %w", err)
	}

	if len(images) == 0 {
		return "", fmt.Errorf("no pages found in PDF")
	}

	// Process images with Claude
	var allMarkdown strings.Builder

	for i, imageData := range images {
		pageNum := i + 1
		slog.Info("Processing page with Claude", "page", pageNum, "totalPages", len(images))

		// Prepare Claude API request for this page
		var promptText string
		if len(images) == 1 {
			promptText = fmt.Sprintf("%s\n\nPlease convert this PDF page (%s) to Markdown format:", systemPrompt, fileName)
		} else {
			promptText = fmt.Sprintf("%s\n\nPlease convert this PDF page (%s, page %d of %d) to Markdown format:", systemPrompt, fileName, pageNum, len(images))
		}

		claudeReq := ClaudeRequest{
			Model:     "claude-3-5-sonnet-20241022",
			MaxTokens: 4000,
			Messages: []Message{
				{
					Role: "user",
					Content: []Content{
						{
							Type: "text",
							Text: promptText,
						},
						{
							Type: "image",
							Source: &MediaSource{
								Type:      "base64",
								MediaType: "image/png", // Changed from application/pdf
								Data:      base64.StdEncoding.EncodeToString(imageData),
							},
						},
					},
				},
			},
		}

		// Make request to Claude API
		jsonData, err := json.Marshal(claudeReq)
		if err != nil {
			return "", fmt.Errorf("failed to marshal Claude request for page %d: %w", pageNum, err)
		}

		req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
		if err != nil {
			return "", fmt.Errorf("failed to create HTTP request for page %d: %w", pageNum, err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", config.ClaudeAPIKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to make Claude API request for page %d: %w", pageNum, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("claude API returned status %d for page %d: %s", resp.StatusCode, pageNum, string(body))
		}

		var claudeResp ClaudeResponse
		if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
			return "", fmt.Errorf("failed to decode Claude response for page %d: %w", pageNum, err)
		}

		if len(claudeResp.Content) == 0 {
			return "", fmt.Errorf("no content in Claude response for page %d", pageNum)
		}

		// Add page content to result
		if pageNum > 1 {
			allMarkdown.WriteString("\n\n---\n\n") // Page separator
			allMarkdown.WriteString(fmt.Sprintf("## Page %d\n\n", pageNum))
		}
		allMarkdown.WriteString(claudeResp.Content[0].Text)

		slog.Info("Successfully processed page", "page", pageNum)
	}

	return allMarkdown.String(), nil
}

func loadSystemPrompt(promptFile string) (string, error) {
	// Create default system prompt if file doesn't exist
	defaultPrompt := `You are an expert at converting documents from images to well-structured Markdown format. 

Please follow these guidelines:
1. Preserve all text content accurately
2. Use appropriate Markdown formatting (headers, lists, emphasis, etc.)
3. Structure the content logically with proper headings
4. If there are diagrams or drawings, describe them in [brackets]
5. Maintain the original organization and flow of the content
6. Use bullet points or numbered lists where appropriate
7. Bold important terms or concepts
8. If text is unclear, use [unclear] notation
9. For multi-page documents, focus on the content of this specific page
10. Maintain consistency in formatting across pages

Convert the image content to clean, readable Markdown while preserving all meaningful information.`

	// Check if custom prompt file exists
	if _, err := os.Stat(promptFile); os.IsNotExist(err) {
		// Create default prompt file
		if err := os.WriteFile(promptFile, []byte(defaultPrompt), 0644); err != nil {
			return "", fmt.Errorf("failed to create default system prompt file: %w", err)
		}
		fmt.Printf("📝 Created default system prompt file: %s\n", promptFile)
		return defaultPrompt, nil
	}

	// Read existing prompt file
	data, err := os.ReadFile(promptFile)
	if err != nil {
		return "", fmt.Errorf("failed to read system prompt file: %w", err)
	}

	return string(data), nil
}

func createOrMergeDailyNote(ctx context.Context, notesClient *client.Client, markdown string) error {
	// Generate today's date in YYYY-mm-dd format
	today := time.Now().Format("2006-01-02")
	dailyNotePath := fmt.Sprintf("%s.md", today)

	// Check if daily note already exists
	req := mcp.CallToolRequest{}
	req.Params.Name = "read_note"
	req.Params.Arguments = map[string]any{"path": dailyNotePath}

	result, err := notesClient.CallTool(ctx, req)
	if err != nil || result.IsError {
		// File doesn't exist, create new daily note
		fmt.Printf("📝 Creating new daily note: %s\n", dailyNotePath)
		return writeNote(ctx, notesClient, dailyNotePath, formatDailyNote(today, markdown))
	}

	// File exists, append to it
	fmt.Printf("📝 Merging with existing daily note: %s\n", dailyNotePath)
	appendContent := fmt.Sprintf("\n\n---\n\n## PDF Conversion - %s\n\n%s", time.Now().Format("15:04"), markdown)
	return appendNote(ctx, notesClient, dailyNotePath, appendContent)
}

func formatDailyNote(date, content string) string {
	return fmt.Sprintf(`# Daily Note - %s

## PDF Conversion - %s

%s`, date, time.Now().Format("15:04"), content)
}

func writeNote(ctx context.Context, notesClient *client.Client, path, content string) error {
	req := mcp.CallToolRequest{}
	req.Params.Name = "write_note"
	req.Params.Arguments = map[string]any{
		"path":    path,
		"content": content,
	}

	result, err := notesClient.CallTool(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to write note: %w", err)
	}

	if result.IsError {
		return fmt.Errorf("error writing note: %v", result.Content)
	}

	return nil
}

func appendNote(ctx context.Context, notesClient *client.Client, path, content string) error {
	req := mcp.CallToolRequest{}
	req.Params.Name = "append_note"
	req.Params.Arguments = map[string]any{
		"path":    path,
		"content": content,
	}

	result, err := notesClient.CallTool(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to append note: %w", err)
	}

	if result.IsError {
		return fmt.Errorf("error appending note: %v", result.Content)
	}

	return nil
}
