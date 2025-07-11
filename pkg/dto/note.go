package dto

import "time"

// NoteMetadata represents metadata about a note
type NoteMetadata struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	IsDir    bool      `json:"is_dir"`
}

// SearchResult represents a search result
type SearchResult struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Content string `json:"content"`
	Context string `json:"context"`
}
