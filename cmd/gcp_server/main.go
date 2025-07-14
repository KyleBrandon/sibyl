package main

import (
	"context"
	"flag"
	"log"

	notes "github.com/KyleBrandon/sibyl/pkg/notes_mcp"
	"github.com/KyleBrandon/sibyl/pkg/utils"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	logLevel := flag.String("logLevel", "INFO", "Default logging level to use")
	logFile := flag.String("logFile", "notes-server.log", "Default log file to log to")
	utils.ConfigureLogging(*logLevel, *logFile)

	ctx := context.Background()
	notesServer := notes.NewNotesServer(ctx)

	if err := notesServer.McpServer.Run(ctx, mcp.NewStdioTransport()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
