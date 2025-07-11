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

func (ns *NotesServer) NewSearchNotesTool() mcp.Tool {
	return mcp.NewTool("search_notes",
		mcp.WithDescription("Search for text within notes"),
		mcp.WithString("path",
			mcp.Description("Directory path (option, defaults to vault root)"),
		),
		mcp.WithString("query",
			mcp.Description("Search query to perform on the notes"),
			mcp.Required(),
		),
		mcp.WithBoolean("case_sensitive",
			mcp.DefaultBool(false),
			mcp.Description("Whether search should be case sensitive"),
		),
	)
}

// SearchNotes searches for text within notes
func (s *NotesServer) SearchNotes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		// no path specified, default to notes root
		path = ""
	}

	query, err := req.RequireString("query")
	if err != nil {
		return nil, err
	}

	caseSensitive, err := req.RequireBool("case_sensitive")
	if err != nil {
		return nil, err
	}

	fullPath, err := utils.ValidatePath(s.vaultDir, path)
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
				relativePath, _ := filepath.Rel(s.vaultDir, path)

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
