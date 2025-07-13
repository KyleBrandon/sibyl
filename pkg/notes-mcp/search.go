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
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SearchNotesRequest struct {
	Path          string `json:"path,omitempty" mcp:"Directory path (optional, defaults to vault root)"`
	Query         string `json:"query,omitempty" mcp:"Search query to perform on the notes"`
	CaseSensitive bool   `json:"case_sensitive,omitempty" mcp:"Whether search should be case sensitive"`
}

func (ns *NotesServer) NewSearchNotesTool() *mcp.ServerTool {
	return mcp.NewServerTool(
		"search_notes",
		"Find all notes that contain the given text",
		ns.SearchNotes,
		mcp.Input(
			mcp.Property("path", mcp.Description("Directory path (option, defaults to vault root)"), mcp.Required(false)),
			mcp.Property("query", mcp.Description("Search query to perform on the notes"), mcp.Required(true)),
			mcp.Property("case_sensitive", mcp.Description("Whether search should be case sensitive"), mcp.Required(false)),
		),
	)
}

// SearchNotes searches for text within notes
func (ns *NotesServer) SearchNotes(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[SearchNotesRequest]) (*mcp.CallToolResultFor[any], error) {
	path := req.Arguments.Path
	if path == "" {
		path = ns.vaultDir
	}

	query := req.Arguments.Query
	caseSensitive := req.Arguments.CaseSensitive

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

	return &mcp.CallToolResultFor[any]{
		Content: []*mcp.Content{
			mcp.NewTextContent(string(result)),
		},
	}, nil
}
