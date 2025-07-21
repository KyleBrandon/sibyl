package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/KyleBrandon/sibyl/pkg/dto"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	// Define command line flags
	server := flag.String("server", "", "Server command to execute")
	flag.Parse()

	if *server == "" {
		fmt.Println("Error: You must specify the --server <server> --rootDir <note folder>")
		flag.Usage()
		os.Exit(1)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println("Initializing stdio client...")

	c, err := client.NewStdioMCPClient(*server, nil)
	if err != nil {
		slog.Error("Failed to create new client", "error", err)
		os.Exit(1)
	}
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "example-client",
		Version: "1.0.0",
	}

	initResult, err := c.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}
	fmt.Printf(
		"Initialized with server: %s %s\n\n",
		initResult.ServerInfo.Name,
		initResult.ServerInfo.Version,
	)

	// List Tools
	fmt.Println("Listing available tools...")
	toolsRequest := mcp.ListToolsRequest{}
	tools, err := c.ListTools(ctx, toolsRequest)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}
	for _, tool := range tools.Tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}
	fmt.Println()

	req := mcp.CallToolRequest{}
	req.Params.Name = "search_drive_files"
	req.Params.Arguments = map[string]any{"query": "2025"}

	result, err := c.CallTool(ctx, req)
	if err != nil {
		slog.Error("Failed to search Google Drive for a file", "error", err)
		os.Exit(1)
	}

	printToolResult(result)

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		slog.Error("Invalid content returned from search_drive_files")
		os.Exit(1)
	}

	// content := res.Content[0].Text
	var files []dto.DriveFileResult
	err = json.Unmarshal([]byte(textContent.Text), &files)
	if err != nil {
		slog.Error("Failed to unmarshal the files", "content", textContent, "error", err)
		os.Exit(1)
	}

	req = mcp.CallToolRequest{}
	req.Params.Name = "read_drive_file"
	req.Params.Arguments = map[string]any{"file_id": files[0].ID}

	result, err = c.CallTool(ctx, req)
	if err != nil {
		slog.Error("Failed to read the file contents", "error", err)
		os.Exit(1)
	}

	resourceContent, ok := result.Content[0].(mcp.EmbeddedResource)
	if !ok {
		slog.Error("no embedded resource")
		os.Exit(1)
	}

	blobResource, ok := resourceContent.Resource.(mcp.BlobResourceContents)
	if !ok {
		slog.Error("no blob resource")
		os.Exit(1)
	}

	buffer := make([]byte, base64.StdEncoding.DecodedLen(len(blobResource.Blob)))
	_, err = base64.StdEncoding.Decode(buffer, []byte(blobResource.Blob))
	if err != nil {
		slog.Error("Failed to save the PDF file")
		os.Exit(1)
	}
	os.WriteFile("/Users/kyle/workspaces/mcp/sibyl/file.pdf", buffer, 0666)
}

// Helper function to print tool results
func printToolResult(result *mcp.CallToolResult) {
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			fmt.Println(textContent.Text)
		} else {
			jsonBytes, _ := json.MarshalIndent(content, "", "  ")
			fmt.Println(string(jsonBytes))
		}
	}
}
