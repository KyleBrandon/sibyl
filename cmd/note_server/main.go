package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/KyleBrandon/sibyl/pkg/notes"
	"github.com/KyleBrandon/sibyl/pkg/utils"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	logLevel := flag.String("logLevel", "INFO", "Default logging level to use")
	logFile := flag.String("logFile", "notes-server.log", "Default log file to log to")

	f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		slog.Error("Could not open the server log")
		panic(err)
	}

	defer f.Close()

	utils.ConfigureLogging(*logLevel, f)

	ctx := context.Background()
	notesServer := notes.NewNotesServer(ctx)

	if err := notesServer.McpServer.Run(ctx, mcp.NewStdioTransport()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
