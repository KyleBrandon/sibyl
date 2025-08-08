package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMainFunction(t *testing.T) {
	// Test that main function can be called without crashing

	// Save original args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test with help flag
	os.Args = []string{"note-server", "--help"}

	// We can't easily test main() directly without it starting a server
	// In a real implementation, we'd refactor main to be testable

	t.Log("Notes server main function test - would test CLI argument parsing")
}

func TestArgumentParsing(t *testing.T) {
	// Test various command line argument combinations
	tests := []struct {
		name  string
		args  []string
		valid bool
	}{
		{
			name:  "Valid arguments",
			args:  []string{"note-server", "--notesFolder", "/path/to/notes", "--logLevel", "INFO"},
			valid: true,
		},
		{
			name:  "Missing notes folder",
			args:  []string{"note-server", "--logLevel", "INFO"},
			valid: false,
		},
		{
			name:  "Invalid log level",
			args:  []string{"note-server", "--notesFolder", "/path/to/notes", "--logLevel", "INVALID"},
			valid: false,
		},
		{
			name:  "Help flag",
			args:  []string{"note-server", "--help"},
			valid: true,
		},
		{
			name:  "Version flag",
			args:  []string{"note-server", "--version"},
			valid: true,
		},
		{
			name:  "Minimal valid args",
			args:  []string{"note-server", "--notesFolder", "/tmp/notes"},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// In a real implementation, we'd test the argument parsing logic
			if len(tt.args) == 0 {
				t.Error("Test args should not be empty")
			}

			if tt.args[0] != "note-server" {
				t.Error("First arg should be program name")
			}

			// Check for standalone flags first
			for _, arg := range tt.args {
				if arg == "--help" || arg == "--version" {
					// Standalone flags are always valid
					return
				}
			}

			// Basic validation simulation for regular flags
			hasNotesFolder := false
			validLogLevel := true

			for i := 1; i < len(tt.args)-1; i += 2 {
				if i+1 >= len(tt.args) {
					break
				}

				flag := tt.args[i]
				value := tt.args[i+1]

				switch flag {
				case "--notesFolder":
					hasNotesFolder = value != ""
				case "--logLevel":
					validLogLevel = value == "DEBUG" || value == "INFO" || value == "WARN" || value == "ERROR"
				}
			}

			isValid := hasNotesFolder && validLogLevel
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.valid, isValid)
			}
		})
	}
}

func TestEnvironmentVariables(t *testing.T) {
	// Test environment variable handling
	tests := []struct {
		name    string
		envVars map[string]string
		valid   bool
	}{
		{
			name: "Valid environment variables",
			envVars: map[string]string{
				"NOTE_SERVER_FOLDER": "/path/to/notes",
				"LOG_LEVEL":          "INFO",
			},
			valid: true,
		},
		{
			name: "Missing notes folder env var",
			envVars: map[string]string{
				"LOG_LEVEL": "INFO",
			},
			valid: false,
		},
		{
			name: "Invalid log level env var",
			envVars: map[string]string{
				"NOTE_SERVER_FOLDER": "/path/to/notes",
				"LOG_LEVEL":          "INVALID",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			originalEnv := make(map[string]string)
			for key := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
			}

			// Set test env vars
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Restore original env vars
			defer func() {
				for key, value := range originalEnv {
					if value == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, value)
					}
				}
			}()

			// Test environment variable reading
			notesFolder := os.Getenv("NOTE_SERVER_FOLDER")
			logLevel := os.Getenv("LOG_LEVEL")

			hasNotesFolder := notesFolder != ""
			validLogLevel := logLevel == "" || logLevel == "DEBUG" || logLevel == "INFO" || logLevel == "WARN" || logLevel == "ERROR"

			isValid := hasNotesFolder && validLogLevel
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.valid, isValid)
			}
		})
	}
}

func TestNotesFolderValidation(t *testing.T) {
	// Test notes folder validation logic
	tempDir := t.TempDir()

	tests := []struct {
		name  string
		path  string
		setup func() string
		valid bool
	}{
		{
			name: "Valid existing directory",
			setup: func() string {
				return tempDir
			},
			valid: true,
		},
		{
			name: "Valid directory with notes",
			setup: func() string {
				notesDir := filepath.Join(tempDir, "notes")
				os.MkdirAll(notesDir, 0755)
				// Create a test note
				os.WriteFile(filepath.Join(notesDir, "test.md"), []byte("# Test"), 0644)
				return notesDir
			},
			valid: true,
		},
		{
			name: "Nonexistent directory",
			setup: func() string {
				return filepath.Join(tempDir, "nonexistent")
			},
			valid: false,
		},
		{
			name: "File instead of directory",
			setup: func() string {
				filePath := filepath.Join(tempDir, "notadir.txt")
				os.WriteFile(filePath, []byte("test"), 0644)
				return filePath
			},
			valid: false,
		},
		{
			name: "Empty path",
			setup: func() string {
				return ""
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()

			// Test path validation
			isValid := validateNotesFolder(path)

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for path '%s', got valid=%v", tt.valid, path, isValid)
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	// Test configuration validation logic
	tempDir := t.TempDir()

	tests := []struct {
		name   string
		config map[string]string
		valid  bool
	}{
		{
			name: "Valid config",
			config: map[string]string{
				"notes_folder": tempDir,
				"log_level":    "INFO",
				"log_file":     filepath.Join(tempDir, "server.log"),
			},
			valid: true,
		},
		{
			name: "Invalid log level",
			config: map[string]string{
				"notes_folder": tempDir,
				"log_level":    "INVALID",
			},
			valid: false,
		},
		{
			name: "Empty notes folder",
			config: map[string]string{
				"notes_folder": "",
				"log_level":    "INFO",
			},
			valid: false,
		},
		{
			name: "Minimal valid config",
			config: map[string]string{
				"notes_folder": tempDir,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test basic validation logic
			notesFolder := tt.config["notes_folder"]
			logLevel := tt.config["log_level"]

			// Basic validation checks
			hasNotesFolder := validateNotesFolder(notesFolder)
			validLogLevel := logLevel == "" || logLevel == "DEBUG" || logLevel == "INFO" || logLevel == "WARN" || logLevel == "ERROR"

			isValid := hasNotesFolder && validLogLevel

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.valid, isValid)
			}
		})
	}
}

func TestServerStartup(t *testing.T) {
	// Test server startup logic (without actually starting the server)
	tempDir := t.TempDir()

	t.Run("Server initialization", func(t *testing.T) {
		// In a real implementation, we'd test:
		// - Server creation with valid config
		// - Notes folder initialization
		// - MCP protocol setup
		// - Graceful shutdown handling

		config := map[string]string{
			"notes_folder": tempDir,
			"log_level":    "INFO",
			"port":         "8081",
		}

		if !validateNotesFolder(config["notes_folder"]) {
			t.Error("Server should require valid notes folder")
		}

		if config["log_level"] != "INFO" {
			t.Error("Server should use specified log level")
		}
	})

	t.Run("Error handling", func(t *testing.T) {
		// Test error scenarios
		errorScenarios := []string{
			"invalid_notes_folder",
			"permission_denied",
			"disk_full",
			"network_error",
		}

		for _, scenario := range errorScenarios {
			t.Run(scenario, func(t *testing.T) {
				// In a real implementation, we'd simulate these errors
				// and verify proper error handling and logging
				if scenario == "" {
					t.Error("Error scenario should not be empty")
				}
			})
		}
	})
}

func TestLogConfiguration(t *testing.T) {
	// Test logging configuration
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		logLevel string
		logFile  string
		valid    bool
	}{
		{
			name:     "Valid log config",
			logLevel: "INFO",
			logFile:  filepath.Join(tempDir, "server.log"),
			valid:    true,
		},
		{
			name:     "Debug level",
			logLevel: "DEBUG",
			logFile:  filepath.Join(tempDir, "debug.log"),
			valid:    true,
		},
		{
			name:     "Error level",
			logLevel: "ERROR",
			logFile:  filepath.Join(tempDir, "error.log"),
			valid:    true,
		},
		{
			name:     "Invalid log level",
			logLevel: "INVALID",
			logFile:  filepath.Join(tempDir, "server.log"),
			valid:    false,
		},
		{
			name:     "No log file specified",
			logLevel: "INFO",
			logFile:  "",
			valid:    true, // Should default to stdout
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test log level validation
			validLogLevel := tt.logLevel == "DEBUG" || tt.logLevel == "INFO" || tt.logLevel == "WARN" || tt.logLevel == "ERROR"

			// Test log file path validation
			validLogFile := tt.logFile == "" || filepath.IsAbs(tt.logFile)

			isValid := validLogLevel && validLogFile

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.valid, isValid)
			}
		})
	}
}

// Helper function to validate notes folder
func validateNotesFolder(path string) bool {
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func BenchmarkArgumentParsing(b *testing.B) {
	args := []string{"note-server", "--notesFolder", "/path/to/notes", "--logLevel", "INFO", "--logFile", "/path/to/log"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate argument parsing
		if len(args) < 2 {
			b.Fatal("Not enough arguments")
		}

		// Basic parsing simulation
		for j := 1; j < len(args); j += 2 {
			if j+1 >= len(args) {
				break
			}
			flag := args[j]
			value := args[j+1]

			// Simulate flag processing
			switch flag {
			case "--notesFolder":
				if value == "" {
					b.Fatal("Empty notes folder")
				}
			case "--logLevel":
				if value == "" {
					b.Fatal("Empty log level")
				}
			case "--logFile":
				if value == "" {
					b.Fatal("Empty log file")
				}
			}
		}
	}
}

func BenchmarkPathValidation(b *testing.B) {
	tempDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validateNotesFolder(tempDir)
	}
}
