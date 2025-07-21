package notes

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/KyleBrandon/sibyl/pkg/dto"
	"github.com/KyleBrandon/sibyl/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"
)

type SearchNotesRequest struct {
	Path          string `json:"path,omitempty" mcp:"Directory path (optional, defaults to vault root)"`
	Query         string `json:"query,omitempty" mcp:"Search query to perform on the notes"`
	CaseSensitive bool   `json:"case_sensitive,omitempty" mcp:"Whether search should be case sensitive"`
}

func (ns *NotesServer) NewSearchNotesTool() {
	tool := mcp.NewTool(
		"search_notes",
		mcp.WithDescription("Find all notes that contain the given text"),
		mcp.WithString("path", mcp.Description("Directory path (option, defaults to vault root)")),
		mcp.WithString("query", mcp.Description("Search query to perform on the notes"), mcp.Required()),
		mcp.WithBoolean("case_sensitive", mcp.Description("Whether search should be case sensitive")),
	)

	ns.McpServer.AddTool(tool, mcp.NewTypedToolHandler(ns.SearchNotes))
}

// SearchNotes searches for text within notes
func (ns *NotesServer) SearchNotes(ctx context.Context, req mcp.CallToolRequest, params SearchNotesRequest) (*mcp.CallToolResult, error) {
	path := params.Path
	if path == "" {
		path = ns.vaultDir
	}

	query := params.Query
	caseSensitive := params.CaseSensitive

	fullPath, err := utils.ValidatePath(ns.vaultDir, path)
	if err != nil {
		return nil, err
	}

	var results []dto.SearchResult
	if !caseSensitive {
		query = strings.ToLower(query)
	}

	// Compile regex for search
	var pattern *regexp.Regexp
	if caseSensitive {
		pattern = regexp.MustCompile(regexp.QuoteMeta(query))
	} else {
		pattern = regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))
	}

	err = utils.WalkDir(fullPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only search in markdown files
		if info.IsDir() || (!strings.HasSuffix(strings.ToLower(info.Name()), ".md") && !strings.HasSuffix(strings.ToLower(info.Name()), ".markdown")) {
			return nil
		}

		content, err := utils.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			if pattern.MatchString(line) {
				relativePath, _ := filepath.Rel(ns.vaultDir, path)

				// Get context (3 lines before and after)
				contextStart := max(0, lineNum-3)
				contextEnd := min(len(lines), lineNum+4)
				context := strings.Join(lines[contextStart:contextEnd], "\n")

				results = append(results, dto.SearchResult{
					Path:    relativePath,
					Line:    lineNum + 1,
					Content: strings.TrimSpace(line),
					Context: context,
				})
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search notes: %w", err)
	}

	result, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search results: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(result)),
		},
	}, nil
}
