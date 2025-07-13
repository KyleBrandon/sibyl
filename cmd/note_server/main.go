package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/KyleBrandon/sibyl/pkg/notes-mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SibylServer represents our MCP server for Markdown notes
type SibylServer struct {
	ctx context.Context
	// session     *notes.MySession
	mcpServer   *mcp.Server
	notesServer *notes.NotesServer
}

// InitializeMCPServers creates a new Markdown MCP server
func InitializeMCPServers(ctx context.Context) (*SibylServer, error) {
	s := &SibylServer{
		ctx: ctx,
	}

	serverOptions := mcp.ServerOptions{
		InitializedHandler:      s.handleInitialized,
		RootsListChangedHandler: s.handleRootsListChanged,
	}

	s.mcpServer = mcp.NewServer("note-server", "v1.0.0", &serverOptions)
	s.notesServer = notes.NewNotesServer(s.mcpServer)

	return s, nil
}

func (s *SibylServer) handleInitialized(ctx context.Context, session *mcp.ServerSession, params *mcp.InitializedParams) {
	slog.Info("Initialized", "params", params)
}

func (s *SibylServer) handleRootsListChanged(ctx context.Context, session *mcp.ServerSession, params *mcp.RootsListChangedParams) {
	result, err := session.ListRoots(ctx, &mcp.ListRootsParams{})
	if err != nil {
		slog.Error("Failed to get the roots", "error", err)
		return
	}

	if len(result.Roots) != 1 {
		slog.Error("We only support a single root at this time")
		return
	}

	s.notesServer.SetVaultFolder(result.Roots[0].URI)
}

func main() {
	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		slog.Error("Could not open the server log")
		panic(err)
	}

	defer logFile.Close()

	handler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))

	ctx := context.Background()
	server, err := InitializeMCPServers(ctx)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.mcpServer.Run(ctx, mcp.NewStdioTransport()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
