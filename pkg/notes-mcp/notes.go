// Package notes contains the MCP tool implementations for processing Markdown notes
package notes

import "github.com/mark3labs/mcp-go/server"

type NotesServer struct {
	vaultDir  string
	mcpServer *server.MCPServer
}

func NewNotesServer(vaultDir string, mcpServer *server.MCPServer) *NotesServer {
	ns := &NotesServer{
		vaultDir,
		mcpServer,
	}

	ns.addTools()

	return ns
}

// addTools adds all the tools to the server

func (ns *NotesServer) addTools() {
	ns.mcpServer.AddTool(
		ns.NewReadNoteTool(),
		ns.ReadNote)

	ns.mcpServer.AddTool(
		ns.NewWriteNoteTool(),
		ns.WriteNote,
	)

	ns.mcpServer.AddTool(
		ns.NewAppendNoteTool(),
		ns.AppendNote,
	)

	ns.mcpServer.AddTool(
		ns.NewCreateFolderTool(),
		ns.CreateFolder,
	)

	ns.mcpServer.AddTool(
		ns.NewListNotesTool(),
		ns.ListNotes,
	)

	ns.mcpServer.AddTool(
		ns.NewListFoldersTool(),
		ns.ListFolders,
	)

	ns.mcpServer.AddTool(

		ns.NewSearchNotesTool(),
		ns.SearchNotes,
	)
}
