// Package notes_mcp contains the MCP tool implementations for processing Markdown notes
package notes

import (
	"context"

	"github.com/mark3labs/mcp-go/server"
)

type NotesServer struct {
	ctx       context.Context
	McpServer *server.MCPServer
	vaultDir  string
}

func NewNotesServer(ctx context.Context, notesFolder string) *NotesServer {
	ns := &NotesServer{}

	// serverOptions := mcp.ServerOptions{
	// 	InitializedHandler:      ns.handleInitialized,
	// 	RootsListChangedHandler: ns.handleRootsListChanged,
	// }

	ns.ctx = ctx
	ns.vaultDir = notesFolder
	ns.McpServer = server.NewMCPServer("note-server", "v1.0.0", server.WithToolCapabilities(true))
	ns.addTools()

	return ns
}

// func (ns *NotesServer) setVaultFolder(vaultDir string) {
// 	localPath, err := utils.FileURIToPath(vaultDir)
// 	if err != nil {
// 		ns.vaultDir = ""
// 	} else {
// 		ns.vaultDir = localPath
// 	}
// }

// addTools adds all the tools to the server

func (ns *NotesServer) addTools() {
	ns.NewReadNoteTool()
	ns.NewWriteNoteTool()
	ns.NewAppendNoteTool()
	ns.NewCreateFolderTool()
	ns.NewListNotesTool()
	ns.NewListFoldersTool()
	ns.NewSearchNotesTool()
}

// func (ns *NotesServer) handleInitialized(ctx context.Context, session *mcp.ServerSession, params *mcp.InitializedParams) {
// 	slog.Info("Initialized", "params", params)
// }
//
// // handleRootsListChanged will receive a "root changed" event from the client and update the note server to use the new root folder
// func (ns *NotesServer) handleRootsListChanged(ctx context.Context, session *mcp.ServerSession, params *mcp.RootsListChangedParams) {
// 	result, err := session.ListRoots(ctx, &mcp.ListRootsParams{})
// 	if err != nil {
// 		slog.Error("Failed to get the roots", "error", err)
// 		return
// 	}
//
// 	if len(result.Roots) != 1 {
// 		slog.Error("We only support a single root at this time")
// 		return
// 	}
//
// 	ns.setVaultFolder(result.Roots[0].URI)
// }
