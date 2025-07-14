// Package notes_mcp contains the MCP tool implementations for processing Markdown notes
package notes

import (
	"context"
	"log/slog"

	"github.com/KyleBrandon/sibyl/pkg/utils"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type NotesServer struct {
	ctx       context.Context
	McpServer *mcp.Server
	vaultDir  string
}

func NewNotesServer(ctx context.Context) *NotesServer {
	ns := &NotesServer{}

	serverOptions := mcp.ServerOptions{
		InitializedHandler:      ns.handleInitialized,
		RootsListChangedHandler: ns.handleRootsListChanged,
	}

	ns.ctx = ctx
	ns.McpServer = mcp.NewServer("note-server", "v1.0.0", &serverOptions)
	ns.addTools()

	return ns
}

func (ns *NotesServer) SetVaultFolder(vaultDir string) {
	localPath, err := utils.FileURIToPath(vaultDir)
	if err != nil {
		ns.vaultDir = ""
	} else {
		ns.vaultDir = localPath
	}
}

// addTools adds all the tools to the server

func (ns *NotesServer) addTools() {
	ns.McpServer.AddTools(ns.NewReadNoteTool())
	ns.McpServer.AddTools(ns.NewWriteNoteTool())
	ns.McpServer.AddTools(ns.NewAppendNoteTool())
	ns.McpServer.AddTools(ns.NewCreateFolderTool())
	ns.McpServer.AddTools(ns.NewListNotesTool())
	ns.McpServer.AddTools(ns.NewListFoldersTool())
	ns.McpServer.AddTools(ns.NewSearchNotesTool())
}

func (sn *NotesServer) handleInitialized(ctx context.Context, session *mcp.ServerSession, params *mcp.InitializedParams) {
	slog.Info("Initialized", "params", params)
}

// handleRootsListChanged will receive a "root changed" event from the client and update the note server to use the new root folder
func (ns *NotesServer) handleRootsListChanged(ctx context.Context, session *mcp.ServerSession, params *mcp.RootsListChangedParams) {
	result, err := session.ListRoots(ctx, &mcp.ListRootsParams{})
	if err != nil {
		slog.Error("Failed to get the roots", "error", err)
		return
	}

	if len(result.Roots) != 1 {
		slog.Error("We only support a single root at this time")
		return
	}

	ns.SetVaultFolder(result.Roots[0].URI)
}
