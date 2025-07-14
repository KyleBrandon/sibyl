package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/KyleBrandon/sibyl/pkg/notes-mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	logLevel := flag.String("logLevel", "INFO", "Default logging level to use")
	logFile := flag.String("logFile", "notes-server.log", "Default log file to log to")
	configureLogging(*logLevel, *logFile)

	ctx := context.Background()
	notesServer := notes.NewNotesServer(ctx)

	if err := notesServer.McpServer.Run(ctx, mcp.NewStdioTransport()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func configureLogging(logLevel, logFile string) {
	level := parseLevel(logLevel)

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		slog.Error("Could not open the server log")
		panic(err)
	}

	defer f.Close()

	handler := slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}

func parseLevel(logLevel string) slog.Leveler {
	switch logLevel {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "ERROR":
		return slog.LevelError
	case "WARN":
		return slog.LevelWarn
	default:
		return slog.LevelWarn
	}
}
