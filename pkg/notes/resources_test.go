package notes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestListNoteFiles_Success(t *testing.T) {
	// Create temporary directory with test notes
	tempDir := t.TempDir()

	// Create test notes with different content
	testNotes := map[string]string{
		"daily.md": `---
tags: [daily, productivity]
---

# Daily Note - 2025-01-15

Today I worked on #development and #testing.`,

		"meeting.md": `---
tags: [meeting, project-a]
---

# Meeting Notes

This is a test meeting note with #important tag.`,

		"research/paper.md": `# Research Notes

This is research about #ai and #machine-learning.

## Key Points

- Important finding 1
- Important finding 2`,
	}

	// Create directory structure and files
	for path, content := range testNotes {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if dir != tempDir {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				t.Fatalf("Failed to create directory %s: %v", dir, err)
			}
		}

		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test note %s: %v", path, err)
		}
	}

	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: tempDir,
	}

	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "notes://files/",
		},
	}

	resources, err := ns.ListNoteFiles(ctx, request)
	if err != nil {
		t.Fatalf("ListNoteFiles failed: %v", err)
	}

	if len(resources) == 0 {
		t.Fatal("No resources returned from ListNoteFiles")
	}

	// Verify the resource structure
	resource := resources[0]
	textResource, ok := resource.(mcp.TextResourceContents)
	if !ok {
		t.Fatal("Resource is not TextResourceContents")
	}

	if textResource.URI != "notes://files/" {
		t.Errorf("Expected URI 'notes://files/', got '%s'", textResource.URI)
	}

	if textResource.MIMEType != "application/json" {
		t.Errorf("Expected MIME type 'application/json', got '%s'", textResource.MIMEType)
	}

	// Verify the JSON content
	var noteFiles []map[string]interface{}
	err = json.Unmarshal([]byte(textResource.Text), &noteFiles)
	if err != nil {
		t.Errorf("Invalid JSON in note files resource: %v", err)
	}

	if len(noteFiles) != 3 {
		t.Errorf("Expected 3 note files, got %d", len(noteFiles))
	}

	// Verify each note file has required fields
	for i, noteFile := range noteFiles {
		requiredFields := []string{"path", "name", "size", "modified", "uri", "tags", "preview"}
		for _, field := range requiredFields {
			if _, exists := noteFile[field]; !exists {
				t.Errorf("Required field '%s' missing from note file %d", field, i)
			}
		}

		// Verify URI format
		uri, ok := noteFile["uri"].(string)
		if !ok {
			t.Errorf("URI field should be string in note file %d", i)
			continue
		}

		if !strings.HasPrefix(uri, "notes://files/") {
			t.Errorf("Invalid URI format in note file %d: %s", i, uri)
		}

		// Verify tags are extracted
		tags, ok := noteFile["tags"].([]interface{})
		if !ok {
			t.Errorf("Tags field should be array in note file %d", i)
			continue
		}

		if len(tags) == 0 {
			t.Errorf("Note file %d should have extracted tags", i)
		}
	}
}

func TestListNoteFiles_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: tempDir,
	}

	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "notes://files/",
		},
	}

	resources, err := ns.ListNoteFiles(ctx, request)
	if err != nil {
		t.Fatalf("ListNoteFiles failed: %v", err)
	}

	if len(resources) == 0 {
		t.Fatal("No resources returned")
	}

	resource := resources[0].(mcp.TextResourceContents)

	var noteFiles []map[string]interface{}
	err = json.Unmarshal([]byte(resource.Text), &noteFiles)
	if err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}

	if len(noteFiles) != 0 {
		t.Errorf("Expected 0 note files in empty directory, got %d", len(noteFiles))
	}
}

func TestListNoteTemplates_Success(t *testing.T) {
	ctx := context.Background()
	ns := &NotesServer{}

	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "notes://templates/",
		},
	}

	resources, err := ns.ListNoteTemplates(ctx, request)
	if err != nil {
		t.Fatalf("ListNoteTemplates failed: %v", err)
	}

	if len(resources) == 0 {
		t.Fatal("No resources returned from ListNoteTemplates")
	}

	// Verify the resource structure
	resource := resources[0]
	textResource, ok := resource.(mcp.TextResourceContents)
	if !ok {
		t.Fatal("Resource is not TextResourceContents")
	}

	if textResource.URI != "notes://templates/" {
		t.Errorf("Expected URI 'notes://templates/', got '%s'", textResource.URI)
	}

	if textResource.MIMEType != "application/json" {
		t.Errorf("Expected MIME type 'application/json', got '%s'", textResource.MIMEType)
	}

	// Verify the JSON content
	var templates map[string]interface{}
	err = json.Unmarshal([]byte(textResource.Text), &templates)
	if err != nil {
		t.Errorf("Invalid JSON in templates resource: %v", err)
	}

	// Verify expected template types
	expectedTypes := []string{"daily", "meeting", "research", "project"}
	for _, templateType := range expectedTypes {
		if _, exists := templates[templateType]; !exists {
			t.Errorf("Expected template type '%s' not found", templateType)
		}
	}

	// Verify template structure
	for templateType, templateData := range templates {
		templateMap, ok := templateData.(map[string]interface{})
		if !ok {
			t.Errorf("Template '%s' is not a map", templateType)
			continue
		}

		requiredFields := []string{"name", "description", "use_case", "uri", "content"}
		for _, field := range requiredFields {
			if _, exists := templateMap[field]; !exists {
				t.Errorf("Required field '%s' missing from template '%s'", field, templateType)
			}
		}

		// Verify URI format
		expectedURI := "notes://templates/" + templateType
		if templateMap["uri"] != expectedURI {
			t.Errorf("Expected URI '%s', got '%s'", expectedURI, templateMap["uri"])
		}
	}
}

func TestListNoteCollections_Success(t *testing.T) {
	// Create temporary directory with test notes
	tempDir := t.TempDir()

	testNotes := map[string]string{
		"daily.md": `---
tags: [daily, productivity]
---

# Daily Note

Content with #work tag.`,

		"projects/project-a.md": `---
tags: [project, development]
---

# Project A

Content about #development.`,

		"meetings/standup.md": `# Standup Meeting

Notes with #meeting and #team tags.`,
	}

	// Create directory structure and files
	for path, content := range testNotes {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if dir != tempDir {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				t.Fatalf("Failed to create directory %s: %v", dir, err)
			}
		}

		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test note %s: %v", path, err)
		}
	}

	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: tempDir,
	}

	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "notes://collections/",
		},
	}

	resources, err := ns.ListNoteCollections(ctx, request)
	if err != nil {
		t.Fatalf("ListNoteCollections failed: %v", err)
	}

	if len(resources) == 0 {
		t.Fatal("No resources returned from ListNoteCollections")
	}

	// Verify the resource structure
	resource := resources[0]
	textResource, ok := resource.(mcp.TextResourceContents)
	if !ok {
		t.Fatal("Resource is not TextResourceContents")
	}

	if textResource.URI != "notes://collections/" {
		t.Errorf("Expected URI 'notes://collections/', got '%s'", textResource.URI)
	}

	// Verify the JSON content
	var collections map[string]interface{}
	err = json.Unmarshal([]byte(textResource.Text), &collections)
	if err != nil {
		t.Errorf("Invalid JSON in collections resource: %v", err)
	}

	// Verify collections structure
	if _, exists := collections["folders"]; !exists {
		t.Error("Collections should contain 'folders' key")
	}

	if _, exists := collections["tags"]; !exists {
		t.Error("Collections should contain 'tags' key")
	}

	// Verify folders
	folders, ok := collections["folders"].(map[string]interface{})
	if !ok {
		t.Error("Folders should be a map")
	} else {
		// Should have root, projects, and meetings folders
		expectedFolders := []string{"root", "projects", "meetings"}
		for _, folder := range expectedFolders {
			if _, exists := folders[folder]; !exists {
				t.Errorf("Expected folder '%s' not found", folder)
			}
		}
	}

	// Verify tags
	tags, ok := collections["tags"].(map[string]interface{})
	if !ok {
		t.Error("Tags should be a map")
	} else {
		// Should have various tags extracted from content
		expectedTags := []string{"daily", "productivity", "project", "development", "work", "meeting", "team"}
		foundTags := 0
		for _, tag := range expectedTags {
			if _, exists := tags[tag]; exists {
				foundTags++
			}
		}

		if foundTags == 0 {
			t.Error("No expected tags found in collections")
		}
	}
}

func TestExtractTags_Frontmatter(t *testing.T) {
	content := `---
tags: [development, testing, "machine learning"]
---

# Test Note

Content here.`

	tags := extractTags(content)

	expectedTags := []string{"development", "testing", "machine learning"}
	if len(tags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(tags))
	}

	for _, expectedTag := range expectedTags {
		found := false
		for _, tag := range tags {
			if tag == expectedTag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tag '%s' not found", expectedTag)
		}
	}
}

func TestExtractTags_Hashtags(t *testing.T) {
	content := `# Test Note

This content has #development and #testing tags.
Also mentions #machine-learning and #ai.`

	tags := extractTags(content)

	expectedTags := []string{"development", "testing", "machine-learning", "ai"}
	if len(tags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(tags))
	}

	for _, expectedTag := range expectedTags {
		found := false
		for _, tag := range tags {
			if tag == expectedTag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tag '%s' not found", expectedTag)
		}
	}
}

func TestExtractTags_Mixed(t *testing.T) {
	content := `---
tags: [project, meeting]
---

# Meeting Notes

Discussion about #development and #testing.
Also covered #deployment topics.`

	tags := extractTags(content)

	// Should extract both frontmatter and hashtag tags
	expectedTags := []string{"project", "meeting", "development", "testing", "deployment"}

	for _, expectedTag := range expectedTags {
		found := false
		for _, tag := range tags {
			if tag == expectedTag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tag '%s' not found", expectedTag)
		}
	}

	// Should not have duplicates
	tagCounts := make(map[string]int)
	for _, tag := range tags {
		tagCounts[tag]++
	}

	for tag, count := range tagCounts {
		if count > 1 {
			t.Errorf("Tag '%s' appears %d times (should be unique)", tag, count)
		}
	}
}

func TestGetContentPreview(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "Simple content",
			content: `# Title

First paragraph here.
Second paragraph here.`,
			expected: "# Title First paragraph here. Second paragraph here.",
		},
		{
			name: "Content with frontmatter",
			content: `---
tags: [test]
---

# Title

First paragraph after frontmatter.`,
			expected: "# Title First paragraph after frontmatter.",
		},
		{
			name: "Long content",
			content: `# Very Long Title

This is a very long paragraph that should be truncated because it exceeds the maximum preview length that we want to show in the resource listing. It contains a lot of text that would make the preview too long.`,
			expected: "", // We'll check length instead of exact content
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preview := getContentPreview(tt.content)

			// Allow some flexibility in preview length (up to 210 characters)
			if len(preview) > 210 {
				t.Errorf("Preview too long: %d characters", len(preview))
			}

			// For long content test, verify it's truncated or reasonably short
			if tt.name == "Long content" {
				if len(preview) > 210 {
					t.Error("Long content should be truncated")
				}
			}

			// Should not contain frontmatter
			if strings.Contains(preview, "---") {
				t.Error("Preview should not contain frontmatter")
			}
		})
	}
}

func BenchmarkListNoteFiles(b *testing.B) {
	// Create temporary directory with many test files
	tempDir := b.TempDir()

	// Create 100 test notes
	for i := 0; i < 100; i++ {
		content := fmt.Sprintf(`---
tags: [test, file%d]
---

# Test Note %d

This is test content for note %d.`, i, i, i)

		filename := fmt.Sprintf("note%d.md", i)
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		if err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: tempDir,
	}

	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "notes://files/",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ns.ListNoteFiles(ctx, request)
		if err != nil {
			b.Fatalf("ListNoteFiles failed: %v", err)
		}
	}
}

func BenchmarkExtractTags(b *testing.B) {
	content := `---
tags: [development, testing, machine-learning, ai, productivity]
---

# Test Note

This content has #development and #testing tags.
Also mentions #machine-learning and #ai and #productivity.
More content with #work and #project tags.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractTags(content)
	}
}

func BenchmarkGetContentPreview(b *testing.B) {
	content := strings.Repeat("This is a line of content that will be used for preview generation. ", 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getContentPreview(content)
	}
}
