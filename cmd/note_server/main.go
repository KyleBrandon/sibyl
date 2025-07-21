package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/KyleBrandon/sibyl/pkg/notes"
	"github.com/KyleBrandon/sibyl/pkg/utils"
	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/server"
)

var (
	logLevel        string
	logFileName     string
	notesFileFolder string
)

func init() {
	flag.StringVar(&logLevel, "logLevel", "INFO", "Default logging level to use")
	flag.StringVar(&logFileName, "logFile", "notes-server.log", "Default log file to log to")
	flag.StringVar(&notesFileFolder, "notesFolder", "", "Folder containing the notes")
}

func main() {
	flag.Parse()

	f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		slog.Error(fmt.Sprintf("Could not open the log file: %s", logFileName))
		os.Exit(1)
	}

	defer f.Close()

	ctx := context.Background()

	utils.ConfigureLogging(logLevel, f)
	notesRootFolder := parseRootFolder()

	notesServer := notes.NewNotesServer(ctx, notesRootFolder)

	if err := server.ServeStdio(notesServer.McpServer); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func parseRootFolder() string {
	// if there is no commandline argument for the notes folder, see if there is an environment
	if notesFileFolder == "" {

		// load the environment
		err := godotenv.Load()
		if err != nil {
			slog.Warn("Could not load .env file", "error", err)
		}

		// read the port from the environment settings
		return os.Getenv("NOTE_SERVER_FOLDER")
	}

	return notesFileFolder
}
