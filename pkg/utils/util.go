// Package utils contains file system abstraction methods for easier testing
package utils

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath ensures the path is within the vault directory
func ValidatePath(vaultDir, inputPath string) (string, error) {
	if vaultDir == "" {
		slog.Error("Vault folder has not been set")
		return "", fmt.Errorf("vault folder has not been set as a root by the client")
	}

	if inputPath == "" {
		return vaultDir, nil
	}

	// Convert relative path to absolute path within vault
	var fullPath string
	if filepath.IsAbs(inputPath) {
		fullPath = inputPath
	} else {
		fullPath = filepath.Join(vaultDir, inputPath)
	}

	// Clean the path
	fullPath = filepath.Clean(fullPath)

	// Ensure the path is within the vault directory
	if !strings.HasPrefix(fullPath, vaultDir) {
		slog.Error("Path is outside the vault directory", "vaultDir", vaultDir, "inputPath", inputPath, "fullPath", fullPath)
		return "", fmt.Errorf("path is outside vault directory")
	}

	return fullPath, nil
}

func Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func AppendFile(path string, data []byte, perm fs.FileMode) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

func WalkDir(root string, walkFn filepath.WalkFunc) error {
	return filepath.Walk(root, walkFn)
}

func FileURIToPath(fileURI string) (string, error) {
	u, err := url.Parse(fileURI)
	if err != nil {
		return "", err
	}

	if u.Scheme != "file" {
		return "", fmt.Errorf("unsupported URI scheme: %s", u.Scheme)
	}

	// `u.Path` will be percent-decoded automatically.
	// On macOS, `u.Path` usually starts with a `/` even if it points to `/Users/...`
	// so we can just clean it with filepath.
	path := filepath.Clean(u.Path)

	return path, nil
}

func ConfigureLogging(logLevel string, logFile *os.File) {
	level := parseLevel(logLevel)

	handler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
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
		return slog.LevelInfo
	}
}
