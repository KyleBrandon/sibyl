package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/KyleBrandon/sibyl/pkg/notes-mcp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// SibylServer represents our MCP server for Markdown notes
type SibylServer struct {
	ctx         context.Context
	session     *notes.MySession
	mcpServer   *server.MCPServer
	notesServer *notes.NotesServer
	vaultDir    string
}

// InitializeMCPServers creates a new Markdown MCP server
func InitializeMCPServers(ctx context.Context, vaultDir string) (*SibylServer, error) {
	// Verify vault directory exists
	if _, err := os.Stat(vaultDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("vault directory does not exist: %s", vaultDir)
	}

	// Clean the vault directory path
	vaultDir = filepath.Clean(vaultDir)

	mcpServer := server.NewMCPServer("note-server", "1.0.0")

	// Register a session
	session := &notes.MySession{
		ID:           "user-123",
		NotifChannel: make(chan mcp.JSONRPCNotification, 10),
	}

	ns := notes.NewNotesServer(vaultDir, mcpServer)

	if err := mcpServer.RegisterSession(ctx, session); err != nil {
		log.Printf("Failed to register session: %v", err)
	}

	s := &SibylServer{
		ctx:         ctx,
		session:     session,
		mcpServer:   mcpServer,
		notesServer: ns,
		vaultDir:    vaultDir,
	}

	mcpServer.AddNotificationHandler("notifications/initialized", s.handleNotifications)
	mcpServer.AddNotificationHandler("notifications/roots/list_changed", s.handleNotifications)

	return s, nil
}

func (s *SibylServer) handleNotifications(ctx context.Context, notification mcp.JSONRPCNotification) {
	slog.Info("handleNotifications", "method", notification.Method, "params", notification.Params)

	slog.Info("session", "clientInfo", s.session.ClientInfo)
}

// Run starts the server
func (s *SibylServer) Run() error {
	log.Println("Starting server...")
	if err := server.ServeStdio(s.mcpServer); err != nil {
		slog.Error("Server error", "error", err)
		return err
	}

	s.mcpServer.UnregisterSession(s.ctx, s.session.ID)

	return nil
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: note-server <vault-directory>")
	}

	vaultDir := os.Args[1]

	ctx := context.Background()
	server, err := InitializeMCPServers(ctx, vaultDir)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
