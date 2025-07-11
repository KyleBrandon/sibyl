// Package utils contains file system abstraction methods for easier testing
package utils

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath ensures the path is within the vault directory
func ValidatePath(vaultDir, inputPath string) (string, error) {
	if inputPath == "" {
		slog.Info("input path is empty, use vault root")
		return vaultDir, nil
	}

	// Convert relative path to absolute path within vault
	var fullPath string
	if filepath.IsAbs(inputPath) {
		slog.Info("input path is absolute path", "inputPath", inputPath)
		fullPath = inputPath
	} else {
		fullPath = filepath.Join(vaultDir, inputPath)
		slog.Info("input path is relative", "inputPath", inputPath, "fullPath", fullPath)
	}

	// Clean the path
	fullPath = filepath.Clean(fullPath)

	// Ensure the path is within the vault directory
	if !strings.HasPrefix(fullPath, vaultDir) {
		return "", fmt.Errorf("path is outside vault directory: %s", inputPath)
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

// Utility functions
// func Max(a, b int) int {
// 	if a > b {
// 		return a
// 	}
// 	return b
// }
//
// func min(a, b int) int {
// 	if a < b {
// 		return a
// 	}
// 	return b
// }
