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

func NewNotesServer(ctx context.Context, notesFolder string) *NotesServer {
	ns := &NotesServer{}

	serverOptions := mcp.ServerOptions{
		InitializedHandler:      ns.handleInitialized,
		RootsListChangedHandler: ns.handleRootsListChanged,
	}

	ns.ctx = ctx
	ns.vaultDir = notesFolder
	ns.McpServer = mcp.NewServer("note-server", "v1.0.0", &serverOptions)
	ns.addTools()

	return ns
}

func (ns *NotesServer) setVaultFolder(vaultDir string) {
	localPath, err := utils.FileURIToPath(vaultDir)
	if err != nil {
		ns.vaultDir = ""
	} else {
		ns.vaultDir = localPath
	}
}

// addTools adds all the tools to the server

func (ns *NotesServer) addTools() {
	ns.McpServer.AddTools(
		ns.NewReadNoteTool(),
		ns.NewWriteNoteTool(),
		ns.NewAppendNoteTool(),
		ns.NewCreateFolderTool(),
		ns.NewListNotesTool(),
		ns.NewListFoldersTool(),
		ns.NewSearchNotesTool())
}

func (ns *NotesServer) handleInitialized(ctx context.Context, session *mcp.ServerSession, params *mcp.InitializedParams) {
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

	ns.setVaultFolder(result.Roots[0].URI)
}
