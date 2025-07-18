package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/KyleBrandon/sibyl/pkg/gcp-mcp"
	"github.com/KyleBrandon/sibyl/pkg/utils"
	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	logLevel := flag.String("logLevel", "INFO", "Default logging level to use")
	logFile := flag.String("logFile", "gcp-server.log", "Default log file to log to")
	flag.Parse()

	f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		slog.Error("Could not open the server log")
		os.Exit(1)
	}

	defer f.Close()

	utils.ConfigureLogging(*logLevel, f)

	// load the environment
	err = godotenv.Load()
	if err != nil {
		slog.Error("could not load .env file", "error", err)
		os.Exit(1)
	}

	credentialsFile := os.Getenv("GOOGLE_SERVICE_KEY_FILE")
	if len(credentialsFile) == 0 {
		slog.Error("Environment GOOGLE_SERVICE_KEY_FILE is not set")
		os.Exit(1)
	}

	ctx := context.Background()
	gcpServer, err := gcp.NewGCPServer(ctx, credentialsFile)
	if err != nil {
		slog.Error("Failed to configure the GCP server connection", "error", err)
		os.Exit(1)
	}

	if err := gcpServer.McpServer.Run(ctx, mcp.NewStdioTransport()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
