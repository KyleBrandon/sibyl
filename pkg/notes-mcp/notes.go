// Package notes contains the MCP tool implementations for processing Markdown notes
package notes

import (
	"github.com/KyleBrandon/sibyl/pkg/utils"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type NotesServer struct {
	vaultDir  string
	mcpServer *mcp.Server
}

func NewNotesServer(mcpServer *mcp.Server) *NotesServer {
	ns := &NotesServer{}
	ns.mcpServer = mcpServer

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
	ns.mcpServer.AddTools(ns.NewReadNoteTool())
	ns.mcpServer.AddTools(ns.NewWriteNoteTool())
	ns.mcpServer.AddTools(ns.NewAppendNoteTool())
	ns.mcpServer.AddTools(ns.NewCreateFolderTool())
	ns.mcpServer.AddTools(ns.NewListNotesTool())
	ns.mcpServer.AddTools(ns.NewListFoldersTool())
	ns.mcpServer.AddTools(ns.NewSearchNotesTool())
}
