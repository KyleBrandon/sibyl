package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/KyleBrandon/sibyl/pkg/notes-mcp"
	"github.com/mark3labs/mcp-go/server"
)

// SibylServer represents our MCP server for Markdown notes
type SibylServer struct {
	mcpServer   *server.MCPServer
	notesServer *notes.NotesServer
	vaultDir    string
}

// InitializeMCPServers creates a new Markdown MCP server
func InitializeMCPServers(vaultDir string) (*SibylServer, error) {
	// Verify vault directory exists
	if _, err := os.Stat(vaultDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("vault directory does not exist: %s", vaultDir)
	}

	// Clean the vault directory path
	vaultDir = filepath.Clean(vaultDir)

	mcpServer := server.NewMCPServer("note-server", "1.0.0")

	ns := notes.NewNotesServer(vaultDir, mcpServer)

	s := &SibylServer{
		mcpServer:   mcpServer,
		notesServer: ns,
	}

	return s, nil
}

// Run starts the server
func (s *SibylServer) Run(ctx context.Context) error {
	log.Println("Starting sampling example server...")
	if err := server.ServeStdio(s.mcpServer); err != nil {
		slog.Error("Server error", "error", err)
		return err
	}

	return nil
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: note-server <vault-directory>")
	}

	vaultDir := os.Args[1]

	server, err := InitializeMCPServers(vaultDir)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	ctx := context.Background()
	if err := server.Run(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
