package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Define command line flags
	server := flag.String("server", "", "Server command to execute")
	rootDir := flag.String("rootDir", "", "Local folder for the server to use")
	flag.Parse()

	if *server == "" || *rootDir == "" {
		fmt.Println("Error: You must specify the --server <server> --rootDir <note folder>")
		flag.Usage()
		os.Exit(1)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Initializing stdio client...")
	client := mcp.NewClient("mcp-client", "v1.0.0", nil)

	// Create stdio transport with verbose logging
	stdioTransport := mcp.NewCommandTransport(exec.Command(*server))

	// Create client with the transport
	session, err := client.Connect(ctx, stdioTransport)
	if err != nil {
		slog.Error("Failed to connect the client", "command", *server, "rootDir", *rootDir, "error", err)
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

	root := fmt.Sprintf("file://%s", *rootDir)
	client.AddRoots(&mcp.Root{Name: "note-folder", URI: root})

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_notes",
		Arguments: map[string]any{"path": "00-inbox"},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(res.Content[0].Text)

	session.Wait()
}
