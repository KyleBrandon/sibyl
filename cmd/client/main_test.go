package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Test fixtures
var (
	validConfigYAML = `mcpServers:
  note-server:
    type: "local"
    command: ["./bin/note-server"]
    args:
      - "--logLevel"
      - "INFO"
      - "--notesFolder"
      - "/tmp/notes"
  gcp-server:
    type: "local"
    command: ["./bin/gcp-server"]
    args:
      - "--logLevel"
      - "INFO"
      - "--gcpFolderID"
      - "test-folder-id"
`

	minimalConfigYAML = `mcpServers:
  note-server:
    type: "local"
    command: ["./bin/note-server"]
  gcp-server:
    type: "local"
    command: ["./bin/gcp-server"]
`

	invalidYAML = `mcpServers:
  note-server:
    type: "local"
    command: ["./bin/note-server"
    # Missing closing bracket - invalid YAML
`

	missingServerConfigYAML = `mcpServers:
  note-server:
    type: "local"
    command: ["./bin/note-server"]
  # Missing gcp-server
`

	wrongTypeConfigYAML = `mcpServers:
  note-server:
    type: "local"
    command: ["./bin/note-server"]
  gcp-server:
    type: "remote"
    command: ["./bin/gcp-server"]
`

	emptyCommandConfigYAML = `mcpServers:
  note-server:
    type: "local"
    command: []
  gcp-server:
    type: "local"
    command: ["./bin/gcp-server"]
`
)

// Helper functions
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "mcphost-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	return tmpFile.Name()
}

func setupTempDir(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "mcphost-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	return tmpDir
}

func cleanupTempDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.RemoveAll(dir); err != nil {
		t.Errorf("Failed to cleanup temp directory %s: %v", dir, err)
	}
}

func createConfigInDir(t *testing.T, dir, filename, content string) string {
	t.Helper()
	configPath := filepath.Join(dir, filename)
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file %s: %v", configPath, err)
	}
	return configPath
}

// Tests for loadMCPHostsConfig
func TestLoadMCPHostsConfig_ValidConfig(t *testing.T) {
	configFile := createTempConfigFile(t, validConfigYAML)
	defer os.Remove(configFile)

	config, err := loadMCPHostsConfig(configFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	// Check note-server
	noteServer, exists := config.MCPServers["note-server"]
	if !exists {
		t.Fatal("Expected note-server to exist in config")
	}
	if noteServer.Type != "local" {
		t.Errorf("Expected note-server type to be 'local', got: %s", noteServer.Type)
	}
	if len(noteServer.Command) != 1 || noteServer.Command[0] != "./bin/note-server" {
		t.Errorf("Expected note-server command to be ['./bin/note-server'], got: %v", noteServer.Command)
	}
	if len(noteServer.Args) != 4 {
		t.Errorf("Expected note-server to have 4 args, got: %d", len(noteServer.Args))
	}

	// Check gcp-server
	gcpServer, exists := config.MCPServers["gcp-server"]
	if !exists {
		t.Fatal("Expected gcp-server to exist in config")
	}
	if gcpServer.Type != "local" {
		t.Errorf("Expected gcp-server type to be 'local', got: %s", gcpServer.Type)
	}
	if len(gcpServer.Command) != 1 || gcpServer.Command[0] != "./bin/gcp-server" {
		t.Errorf("Expected gcp-server command to be ['./bin/gcp-server'], got: %v", gcpServer.Command)
	}
}

func TestLoadMCPHostsConfig_MinimalConfig(t *testing.T) {
	configFile := createTempConfigFile(t, minimalConfigYAML)
	defer os.Remove(configFile)

	config, err := loadMCPHostsConfig(configFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that servers exist with minimal configuration
	noteServer, exists := config.MCPServers["note-server"]
	if !exists {
		t.Fatal("Expected note-server to exist in config")
	}
	if len(noteServer.Args) != 0 {
		t.Errorf("Expected note-server to have no args, got: %v", noteServer.Args)
	}
}

func TestLoadMCPHostsConfig_InvalidYAML(t *testing.T) {
	configFile := createTempConfigFile(t, invalidYAML)
	defer os.Remove(configFile)

	_, err := loadMCPHostsConfig(configFile)
	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
	if !contains(err.Error(), "failed to parse YAML config") {
		t.Errorf("Expected YAML parse error, got: %v", err)
	}
}

func TestLoadMCPHostsConfig_NonExistentFile(t *testing.T) {
	_, err := loadMCPHostsConfig("/non/existent/file.yml")
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}
	if !contains(err.Error(), "failed to read config file") {
		t.Errorf("Expected file read error, got: %v", err)
	}
}

func TestLoadMCPHostsConfig_EmptyPath_SearchesPrecedence(t *testing.T) {
	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Create temp directories
	currentDir := setupTempDir(t)
	defer cleanupTempDir(t, currentDir)

	homeDir := setupTempDir(t)
	defer cleanupTempDir(t, homeDir)

	// Set HOME to our temp directory
	os.Setenv("HOME", homeDir)

	// Change to current directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(currentDir)

	// Test 1: No config files found
	_, err := loadMCPHostsConfig("")
	if err == nil {
		t.Fatal("Expected error when no config files found")
	}
	if !contains(err.Error(), "no .mcphost.yml file found") {
		t.Errorf("Expected 'no config file found' error, got: %v", err)
	}

	// Test 2: Config in home directory only
	createConfigInDir(t, homeDir, ".mcphost.yml", validConfigYAML)
	config, err := loadMCPHostsConfig("")
	if err != nil {
		t.Fatalf("Expected no error when config in home dir, got: %v", err)
	}
	if config == nil {
		t.Fatal("Expected config to be loaded from home directory")
	}

	// Test 3: Config in current directory takes precedence
	createConfigInDir(t, currentDir, ".mcphost.yml", minimalConfigYAML)
	config, err = loadMCPHostsConfig("")
	if err != nil {
		t.Fatalf("Expected no error when config in current dir, got: %v", err)
	}
	// Verify it loaded the current directory config (minimal) not home config (full)
	noteServer := config.MCPServers["note-server"]
	if len(noteServer.Args) != 0 {
		t.Error("Expected current directory config to be loaded (minimal), but got home config (with args)")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Tests for initializeMCPClient - focusing on command preparation and error handling
func TestInitializeMCPClient_ServerNotFound(t *testing.T) {
	config := &MCPHostsConfig{
		MCPServers: map[string]MCPServer{
			"note-server": {
				Type:    "local",
				Command: []string{"./bin/note-server"},
			},
		},
	}

	ctx := context.Background()
	_, err := initializeMCPClient(ctx, config, "non-existent-server", "test-client")
	if err == nil {
		t.Fatal("Expected error for non-existent server")
	}
	if !contains(err.Error(), "server 'non-existent-server' not found") {
		t.Errorf("Expected server not found error, got: %v", err)
	}
}

func TestInitializeMCPClient_EmptyCommand(t *testing.T) {
	config := &MCPHostsConfig{
		MCPServers: map[string]MCPServer{
			"test-server": {
				Type:    "local",
				Command: []string{}, // Empty command
			},
		},
	}

	ctx := context.Background()
	_, err := initializeMCPClient(ctx, config, "test-server", "test-client")
	if err == nil {
		t.Fatal("Expected error for empty command")
	}
	if !contains(err.Error(), "no command specified") {
		t.Errorf("Expected no command error, got: %v", err)
	}
}

// Test command preparation logic by creating a helper function
func TestCommandPreparation(t *testing.T) {
	testCases := []struct {
		name            string
		server          MCPServer
		expectedCommand string
		expectedArgs    []string
	}{
		{
			name: "command only",
			server: MCPServer{
				Type:    "local",
				Command: []string{"./bin/server"},
			},
			expectedCommand: "./bin/server",
			expectedArgs:    nil,
		},
		{
			name: "command with args",
			server: MCPServer{
				Type:    "local",
				Command: []string{"./bin/server"},
				Args:    []string{"--logLevel", "INFO", "--port", "8080"},
			},
			expectedCommand: "./bin/server",
			expectedArgs:    []string{"--logLevel", "INFO", "--port", "8080"},
		},
		{
			name: "command with multiple parts",
			server: MCPServer{
				Type:    "local",
				Command: []string{"python", "-m", "server"},
				Args:    []string{"--config", "test.yml"},
			},
			expectedCommand: "python",
			expectedArgs:    []string{"-m", "server", "--config", "test.yml"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the command preparation logic
			var fullCmd []string
			fullCmd = append(fullCmd, tc.server.Command...)
			fullCmd = append(fullCmd, tc.server.Args...)

			if len(fullCmd) == 0 {
				t.Fatal("No command specified")
			}

			command := fullCmd[0]
			var args []string
			if len(fullCmd) > 1 {
				args = fullCmd[1:]
			}

			if command != tc.expectedCommand {
				t.Errorf("Expected command '%s', got '%s'", tc.expectedCommand, command)
			}

			if len(args) != len(tc.expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(tc.expectedArgs), len(args))
			}

			for i, expectedArg := range tc.expectedArgs {
				if i >= len(args) || args[i] != expectedArg {
					t.Errorf("Expected arg[%d] to be '%s', got '%s'", i, expectedArg, args[i])
				}
			}
		})
	}
}

// Integration test for the complete config loading and validation workflow
func TestConfigLoadingWorkflow(t *testing.T) {
	// Create a temporary config file
	configFile := createTempConfigFile(t, validConfigYAML)
	defer os.Remove(configFile)

	// Test the complete workflow
	config, err := loadMCPHostsConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	err = validateRequiredServers(config)
	if err != nil {
		t.Fatalf("Failed to validate config: %v", err)
	}

	// Verify we can prepare commands for both servers
	for _, serverName := range []string{"note-server", "gcp-server"} {
		server, exists := config.MCPServers[serverName]
		if !exists {
			t.Fatalf("Server %s not found in config", serverName)
		}

		var fullCmd []string
		fullCmd = append(fullCmd, server.Command...)
		fullCmd = append(fullCmd, server.Args...)

		if len(fullCmd) == 0 {
			t.Fatalf("No command specified for server %s", serverName)
		}

		// Verify command structure
		if fullCmd[0] != "./bin/"+serverName {
			t.Errorf("Expected command for %s to start with './bin/%s', got: %s",
				serverName, serverName, fullCmd[0])
		}
	}
}

// Test error scenarios with different config files
func TestConfigErrorScenarios(t *testing.T) {
	testCases := []struct {
		name        string
		configYAML  string
		expectError string
	}{
		{
			name:        "missing gcp-server",
			configYAML:  missingServerConfigYAML,
			expectError: "required server 'gcp-server' not found",
		},
		{
			name:        "wrong server type",
			configYAML:  wrongTypeConfigYAML,
			expectError: "must have type 'local'",
		},
		{
			name:        "empty command",
			configYAML:  emptyCommandConfigYAML,
			expectError: "must have a command specified",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configFile := createTempConfigFile(t, tc.configYAML)
			defer os.Remove(configFile)

			config, err := loadMCPHostsConfig(configFile)
			if err != nil {
				// If loading fails, that's also a valid test result
				if !contains(err.Error(), tc.expectError) {
					t.Errorf("Expected error containing '%s', got: %v", tc.expectError, err)
				}
				return
			}

			// If loading succeeds, validation should fail
			err = validateRequiredServers(config)
			if err == nil {
				t.Fatalf("Expected validation error for %s", tc.name)
			}
			if !contains(err.Error(), tc.expectError) {
				t.Errorf("Expected error containing '%s', got: %v", tc.expectError, err)
			}
		})
	}
}

// Tests for validateRequiredServers
func TestValidateRequiredServers_ValidConfig(t *testing.T) {
	config := &MCPHostsConfig{
		MCPServers: map[string]MCPServer{
			"note-server": {
				Type:    "local",
				Command: []string{"./bin/note-server"},
				Args:    []string{"--logLevel", "INFO"},
			},
			"gcp-server": {
				Type:    "local",
				Command: []string{"./bin/gcp-server"},
				Args:    []string{"--logLevel", "INFO"},
			},
		},
	}

	err := validateRequiredServers(config)
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}
}

func TestValidateRequiredServers_ValidConfigWithExtraServers(t *testing.T) {
	config := &MCPHostsConfig{
		MCPServers: map[string]MCPServer{
			"note-server": {
				Type:    "local",
				Command: []string{"./bin/note-server"},
			},
			"gcp-server": {
				Type:    "local",
				Command: []string{"./bin/gcp-server"},
			},
			"extra-server": {
				Type:    "local",
				Command: []string{"./bin/extra-server"},
			},
		},
	}

	err := validateRequiredServers(config)
	if err != nil {
		t.Errorf("Expected no error for valid config with extra servers, got: %v", err)
	}
}

func TestValidateRequiredServers_MissingNoteServer(t *testing.T) {
	config := &MCPHostsConfig{
		MCPServers: map[string]MCPServer{
			"gcp-server": {
				Type:    "local",
				Command: []string{"./bin/gcp-server"},
			},
		},
	}

	err := validateRequiredServers(config)
	if err == nil {
		t.Fatal("Expected error for missing note-server")
	}
	if !contains(err.Error(), "required server 'note-server' not found") {
		t.Errorf("Expected missing note-server error, got: %v", err)
	}
}

func TestValidateRequiredServers_MissingGCPServer(t *testing.T) {
	config := &MCPHostsConfig{
		MCPServers: map[string]MCPServer{
			"note-server": {
				Type:    "local",
				Command: []string{"./bin/note-server"},
			},
		},
	}

	err := validateRequiredServers(config)
	if err == nil {
		t.Fatal("Expected error for missing gcp-server")
	}
	if !contains(err.Error(), "required server 'gcp-server' not found") {
		t.Errorf("Expected missing gcp-server error, got: %v", err)
	}
}

func TestValidateRequiredServers_WrongServerType(t *testing.T) {
	testCases := []struct {
		name       string
		serverName string
		serverType string
	}{
		{"note-server with remote type", "note-server", "remote"},
		{"gcp-server with remote type", "gcp-server", "remote"},
		{"note-server with empty type", "note-server", ""},
		{"gcp-server with invalid type", "gcp-server", "invalid"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &MCPHostsConfig{
				MCPServers: map[string]MCPServer{
					"note-server": {
						Type:    "local",
						Command: []string{"./bin/note-server"},
					},
					"gcp-server": {
						Type:    "local",
						Command: []string{"./bin/gcp-server"},
					},
				},
			}

			// Override the specific server type
			server := config.MCPServers[tc.serverName]
			server.Type = tc.serverType
			config.MCPServers[tc.serverName] = server

			err := validateRequiredServers(config)
			if err == nil {
				t.Fatalf("Expected error for %s with type '%s'", tc.serverName, tc.serverType)
			}
			expectedError := "must have type 'local'"
			if !contains(err.Error(), expectedError) {
				t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
			}
		})
	}
}

func TestValidateRequiredServers_EmptyCommand(t *testing.T) {
	testCases := []struct {
		name       string
		serverName string
	}{
		{"note-server with empty command", "note-server"},
		{"gcp-server with empty command", "gcp-server"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &MCPHostsConfig{
				MCPServers: map[string]MCPServer{
					"note-server": {
						Type:    "local",
						Command: []string{"./bin/note-server"},
					},
					"gcp-server": {
						Type:    "local",
						Command: []string{"./bin/gcp-server"},
					},
				},
			}

			// Override the specific server command
			server := config.MCPServers[tc.serverName]
			server.Command = []string{}
			config.MCPServers[tc.serverName] = server

			err := validateRequiredServers(config)
			if err == nil {
				t.Fatalf("Expected error for %s with empty command", tc.serverName)
			}
			expectedError := "must have a command specified"
			if !contains(err.Error(), expectedError) {
				t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
			}
		})
	}
}

func TestValidateRequiredServers_NilCommand(t *testing.T) {
	config := &MCPHostsConfig{
		MCPServers: map[string]MCPServer{
			"note-server": {
				Type:    "local",
				Command: nil, // nil command
			},
			"gcp-server": {
				Type:    "local",
				Command: []string{"./bin/gcp-server"},
			},
		},
	}

	err := validateRequiredServers(config)
	if err == nil {
		t.Fatal("Expected error for nil command")
	}
	if !contains(err.Error(), "must have a command specified") {
		t.Errorf("Expected command specification error, got: %v", err)
	}
}
