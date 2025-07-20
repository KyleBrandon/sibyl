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

var (
	logLevel             string
	logFileName          string
	googleFolderID       string
	googleServiceKeyFile string
)

type GoogleDriveConfig struct {
	folderID       string
	serviceKeyFile string
}

func init() {
	flag.StringVar(&logLevel, "logLevel", "INFO", "Default logging level to use")
	flag.StringVar(&logFileName, "logFile", "gcp-server.log", "Default log file to log to")
	flag.StringVar(&googleFolderID, "gcpFolderID", "", "Folder ID on Google Drive where notes are stored")
	flag.StringVar(&googleServiceKeyFile, "gcpServiceKeyPath", "", "Path to the service key file for Google Drive")
}

func main() {
	flag.Parse()

	f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		slog.Error("Could not open the server log")
		os.Exit(1)
	}

	defer f.Close()

	ctx := context.Background()

	utils.ConfigureLogging(logLevel, f)

	config := loadGoogleDriveConfig()

	gcpServer, err := gcp.NewGCPServer(ctx, config.serviceKeyFile, config.folderID)
	if err != nil {
		slog.Error("Failed to configure the GCP server connection", "error", err)
		os.Exit(1)
	}

	if err := gcpServer.McpServer.Run(ctx, mcp.NewStdioTransport()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func loadGoogleDriveConfig() *GoogleDriveConfig {
	// load the environment
	// Attempt to load environment file, nonfatal
	if err := godotenv.Load(); err != nil {
		slog.Warn("could not load .env file, proceeding without environment file", "error", err)
	}

	// folder ID is optional as the client can override with 'Roots'
	folderID := googleFolderID
	if folderID == "" {
		folderID = os.Getenv("GOOGLE_NOTES_FOLDER_ID")
	}

	serviceKeyFile := googleServiceKeyFile
	if serviceKeyFile == "" {
		serviceKeyFile = os.Getenv("GOOGLE_SERVICE_KEY_FILE")
		if serviceKeyFile == "" {
			slog.Error("Environment GOOGLE_SERVICE_KEY_FILE is not set")
			os.Exit(1)
		}
	}

	return &GoogleDriveConfig{
		folderID,
		serviceKeyFile,
	}
}
