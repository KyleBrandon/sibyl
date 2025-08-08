package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	pdfmcp "github.com/KyleBrandon/sibyl/pkg/pdfmcp"
	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Define command line flags
	credentialsPath := flag.String("credentials", "", "Path to Google Cloud service account credentials JSON file")
	folderID := flag.String("folder-id", "", "Google Drive folder ID to search for PDFs")
	logLevel := flag.String("log-level", "INFO", "Log level (DEBUG, INFO, WARN, ERROR)")
	logFile := flag.String("log-file", "", "Log file path (optional, logs to stderr if not specified)")

	// Mathpix OCR configuration (required)
	ocrLanguages := flag.String("ocr-languages", "en", "OCR languages (comma-separated, e.g., en,fr,de)")
	mathpixAppID := flag.String("mathpix-app-id", "", "Mathpix API App ID (required)")
	mathpixAppKey := flag.String("mathpix-app-key", "", "Mathpix API App Key (required)")

	flag.Parse()

	// Load environment variables if available
	if err := godotenv.Load(); err != nil {
		slog.Debug("No .env file found, using environment variables and command line args")
	}

	// Get values from environment if not provided via flags
	if *credentialsPath == "" {
		*credentialsPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}
	if *folderID == "" {
		*folderID = os.Getenv("GCP_FOLDER_ID")
	}

	if *mathpixAppID == "" {
		*mathpixAppID = os.Getenv("MATHPIX_APP_ID")
	}
	if *mathpixAppKey == "" {
		*mathpixAppKey = os.Getenv("MATHPIX_APP_KEY")
	}

	// Validate required parameters
	if *credentialsPath == "" {
		fmt.Fprintf(os.Stderr, "Error: Google Cloud credentials path is required\n")
		fmt.Fprintf(os.Stderr, "Use --credentials flag or set GOOGLE_APPLICATION_CREDENTIALS environment variable\n")
		os.Exit(1)
	}

	if *folderID == "" {
		fmt.Fprintf(os.Stderr, "Error: Google Drive folder ID is required\n")
		fmt.Fprintf(os.Stderr, "Use --folder-id flag or set GCP_FOLDER_ID environment variable\n")
		os.Exit(1)
	}

	// Configure logging
	var logHandler slog.Handler
	if *logFile != "" {
		file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		logHandler = slog.NewJSONHandler(file, &slog.HandlerOptions{
			Level: parseLogLevel(*logLevel),
		})
	} else {
		logHandler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: parseLogLevel(*logLevel),
		})
	}

	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	// Create context
	ctx := context.Background()

	// Parse OCR languages
	languages := strings.Split(*ocrLanguages, ",")
	for i, lang := range languages {
		languages[i] = strings.TrimSpace(lang)
	}

	// Create OCR configuration
	ocrConfig := pdfmcp.OCRConfig{
		Languages:     languages,
		MathpixAppID:  *mathpixAppID,
		MathpixAppKey: *mathpixAppKey,
	}

	slog.Info("Starting PDF MCP Server",
		"credentials", *credentialsPath,
		"folder_id", *folderID,
		"log_level", *logLevel,
		"ocr_languages", *ocrLanguages,
		"mathpix_configured", *mathpixAppID != "")

	pdfServer, err := pdfmcp.NewPDFServer(ctx, *credentialsPath, *folderID, ocrConfig)
	if err != nil {
		slog.Error("Failed to create PDF server", "error", err)
		os.Exit(1)
	}

	slog.Info("PDF MCP Server initialized successfully")

	// Run the MCP server
	if err := server.ServeStdio(pdfServer.McpServer); err != nil {
		slog.Error("PDF MCP Server failed", "error", err)
		os.Exit(1)
	}
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
