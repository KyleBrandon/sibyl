package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/KyleBrandon/sibyl/pkg/dto"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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
	client := mcp.NewClient("mcp-client", "v1.0.0", nil)

	// Create stdio transport with verbose logging
	// Use shell to allow full command string with arguments
	stdioTransport := mcp.NewCommandTransport(exec.Command("sh", "-c", *server))

	// Create client with the transport
	session, err := client.Connect(ctx, stdioTransport)
	if err != nil {
		slog.Error("Failed to connect the client", "command", *server, "error", err)
		os.Exit(1)
	}

	defer session.Close()

	// Get a lits of the server tools
	toolResult, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		slog.Info("Failed to list the server tools", "error", err)
	} else {
		for _, tool := range toolResult.Tools {
			slog.Info("Server Tool:", "name", tool.Name, "description", tool.Description)
		}
	}

	// List available resources if the server supports them
	resourcesResult, err := session.ListResources(ctx, &mcp.ListResourcesParams{})
	if err != nil {
		slog.Info("Failed to list the server resources", "error", err)
	} else {
		for _, resource := range resourcesResult.Resources {
			slog.Info("Server Tool:", "name", resource.Name, "description", resource.Description)
		}
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_drive_files",
		Arguments: map[string]any{"query": "2025"},
	})
	if err != nil {
		slog.Error("Failed to search Google Drive for a file", "error", err)
		os.Exit(1)
	}

	if len(res.Content) != 1 {
		slog.Error("We should have one result with a JSON list of files")
		os.Exit(1)
	}

	content := res.Content[0].Text
	var files []dto.DriveFileResult
	err = json.Unmarshal([]byte(content), &files)
	if err != nil {
		slog.Error("Failed to unmarshal the files", "content", content, "error", err)
		os.Exit(1)
	}

	res, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "read_drive_file",
		Arguments: map[string]any{"file_id": files[0].ID},
	})
	if err != nil {
		slog.Error("Failed to read the file contents", "error", err)
		os.Exit(1)
	}

	text := res.Content[0].Text
	slog.Info(text)
	// buffer := res.Content[1].Resource.Blob
	// os.WriteFile("/Users/kyle/workspaces/mcp/sibyl/file.pdf", buffer, 0666)

	session.Wait()
}
