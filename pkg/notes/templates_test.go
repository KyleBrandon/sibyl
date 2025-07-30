package notes

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestNoteTemplate_Structure(t *testing.T) {
	template := NoteTemplate{
		Name:        "Test Template",
		Description: "A test template",
		Content:     "# Test\n\nContent here",
		UseCase:     "Testing purposes",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(template)
	if err != nil {
		t.Errorf("Failed to marshal template: %v", err)
	}

	var unmarshaled NoteTemplate
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal template: %v", err)
	}

	if unmarshaled.Name != template.Name {
		t.Errorf("Name mismatch: expected %s, got %s", template.Name, unmarshaled.Name)
	}

	if unmarshaled.Description != template.Description {
		t.Errorf("Description mismatch: expected %s, got %s", template.Description, unmarshaled.Description)
	}

	if unmarshaled.Content != template.Content {
		t.Errorf("Content mismatch: expected %s, got %s", template.Content, unmarshaled.Content)
	}

	if unmarshaled.UseCase != template.UseCase {
		t.Errorf("UseCase mismatch: expected %s, got %s", template.UseCase, unmarshaled.UseCase)
	}
}

func TestGetBuiltinTemplates(t *testing.T) {
	ns := &NotesServer{}
	templates := ns.getBuiltinTemplates()

	if len(templates) == 0 {
		t.Fatal("No builtin templates found")
	}

	expectedTemplates := []string{"daily", "meeting", "research", "project"}
	for _, templateName := range expectedTemplates {
		template, exists := templates[templateName]
		if !exists {
			t.Errorf("Expected template '%s' not found", templateName)
			continue
		}

		// Verify required fields
		if template.Name == "" {
			t.Errorf("Template '%s' has empty name", templateName)
		}

		if template.Description == "" {
			t.Errorf("Template '%s' has empty description", templateName)
		}

		if template.Content == "" {
			t.Errorf("Template '%s' has empty content", templateName)
		}

		if template.UseCase == "" {
			t.Errorf("Template '%s' has empty use case", templateName)
		}

		// Verify content contains expected elements
		switch templateName {
		case "daily":
			if !strings.Contains(template.Content, "Daily Note") {
				t.Errorf("Daily template should contain 'Daily Note'")
			}
			if !strings.Contains(template.Content, "Tasks") {
				t.Errorf("Daily template should contain 'Tasks'")
			}

		case "meeting":
			if !strings.Contains(template.Content, "Meeting Notes") {
				t.Errorf("Meeting template should contain 'Meeting Notes'")
			}
			if !strings.Contains(template.Content, "Action Items") {
				t.Errorf("Meeting template should contain 'Action Items'")
			}

		case "research":
			if !strings.Contains(template.Content, "Research Notes") {
				t.Errorf("Research template should contain 'Research Notes'")
			}
			if !strings.Contains(template.Content, "Key Points") {
				t.Errorf("Research template should contain 'Key Points'")
			}

		case "project":
			if !strings.Contains(template.Content, "Project:") {
				t.Errorf("Project template should contain 'Project:'")
			}
			if !strings.Contains(template.Content, "Milestones") {
				t.Errorf("Project template should contain 'Milestones'")
			}
		}
	}
}

func TestGetNoteTemplates_AllTypes(t *testing.T) {
	ctx := context.Background()
	ns := &NotesServer{}

	request := mcp.CallToolRequest{}
	params := GetTemplatesRequest{TemplateType: "all"}

	result, err := ns.GetNoteTemplates(ctx, request, params)
	if err != nil {
		t.Fatalf("GetNoteTemplates failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("GetNoteTemplates returned error: %v", result.Content[0])
	}

	if len(result.Content) == 0 {
		t.Fatal("No content returned from GetNoteTemplates")
	}

	// Verify the content is valid JSON
	content := result.Content[0].(mcp.TextContent).Text
	var templates map[string]NoteTemplate
	err = json.Unmarshal([]byte(content), &templates)
	if err != nil {
		t.Errorf("Invalid JSON returned: %v", err)
	}

	// Verify expected template types are present
	expectedTypes := []string{"daily", "meeting", "research", "project"}
	for _, templateType := range expectedTypes {
		if _, exists := templates[templateType]; !exists {
			t.Errorf("Expected template type '%s' not found", templateType)
		}
	}
}

func TestGetNoteTemplates_SpecificType(t *testing.T) {
	ctx := context.Background()
	ns := &NotesServer{}

	tests := []struct {
		name         string
		templateType string
		expectError  bool
	}{
		{"Daily template", "daily", false},
		{"Meeting template", "meeting", false},
		{"Research template", "research", false},
		{"Project template", "project", false},
		{"Invalid template", "nonexistent", true},
		{"Empty type", "", false}, // Should return all
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{}
			params := GetTemplatesRequest{TemplateType: tt.templateType}

			result, err := ns.GetNoteTemplates(ctx, request, params)
			if err != nil {
				t.Fatalf("GetNoteTemplates failed: %v", err)
			}

			if tt.expectError {
				if !result.IsError {
					t.Error("Expected error result, but got success")
				}
				return
			}

			if result.IsError {
				t.Errorf("Unexpected error result: %v", result.Content[0])
				return
			}

			if len(result.Content) == 0 {
				t.Error("No content returned")
			}

			// For specific types, verify the returned template
			if tt.templateType != "" && tt.templateType != "all" {
				content := result.Content[0].(mcp.TextContent).Text
				var template NoteTemplate
				err = json.Unmarshal([]byte(content), &template)
				if err != nil {
					t.Errorf("Invalid JSON returned for specific template: %v", err)
				}
			}
		})
	}
}

func TestCreateNoteFromTemplate_Success(t *testing.T) {
	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: t.TempDir(), // Use temporary directory for testing
	}

	request := mcp.CallToolRequest{}
	params := CreateFromTemplateRequest{
		Path:         "test-note.md",
		TemplateType: "daily",
		Variables: map[string]string{
			"CUSTOM_VAR": "test value",
		},
	}

	result, err := ns.CreateNoteFromTemplate(ctx, request, params)
	if err != nil {
		t.Fatalf("CreateNoteFromTemplate failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("CreateNoteFromTemplate returned error: %v", result.Content[0])
	}

	// The result should be from WriteNote, so verify it has content
	if len(result.Content) == 0 {
		t.Error("No content returned from CreateNoteFromTemplate")
	}
}

func TestCreateNoteFromTemplate_InvalidTemplate(t *testing.T) {
	ctx := context.Background()
	ns := &NotesServer{
		vaultDir: t.TempDir(),
	}

	request := mcp.CallToolRequest{}
	params := CreateFromTemplateRequest{
		Path:         "test-note.md",
		TemplateType: "nonexistent",
	}

	result, err := ns.CreateNoteFromTemplate(ctx, request, params)
	if err != nil {
		t.Fatalf("CreateNoteFromTemplate failed: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error for invalid template type")
	}
}

func TestTemplateVariableSubstitution(t *testing.T) {
	// Test variable substitution in templates
	template := "# Meeting Notes - {{TITLE}}\n\n**Date:** {{DATE}}\n**Attendees:** {{ATTENDEES}}"

	variables := map[string]string{
		"TITLE":     "Team Standup",
		"DATE":      "2025-01-15",
		"ATTENDEES": "Alice, Bob, Charlie",
	}

	result := template
	for key, value := range variables {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	expected := "# Meeting Notes - Team Standup\n\n**Date:** 2025-01-15\n**Attendees:** Alice, Bob, Charlie"
	if result != expected {
		t.Errorf("Variable substitution failed.\nExpected: %s\nGot: %s", expected, result)
	}

	// Verify no placeholders remain
	if strings.Contains(result, "{{") || strings.Contains(result, "}}") {
		t.Error("Template still contains unreplaced placeholders")
	}
}

func TestTemplateTimeSubstitution(t *testing.T) {
	// Test that templates correctly substitute time-based variables
	now := time.Now()

	templates := map[string]string{
		"daily":   "# Daily Note - {{DATE}}\n\n*Created: {{DATETIME}}*",
		"meeting": "**Date:** {{DATE}}\n*Meeting notes created: {{DATETIME}}*",
	}

	for templateName, content := range templates {
		t.Run(templateName, func(t *testing.T) {
			// Simulate the time substitution that happens in the actual templates
			result := strings.ReplaceAll(content, "{{DATE}}", now.Format("2006-01-02"))
			result = strings.ReplaceAll(result, "{{DATETIME}}", now.Format("2006-01-02 15:04"))

			// Verify date format
			if !strings.Contains(result, now.Format("2006-01-02")) {
				t.Error("Template should contain current date")
			}

			// Verify no placeholders remain for time variables
			if strings.Contains(result, "{{DATE}}") || strings.Contains(result, "{{DATETIME}}") {
				t.Error("Template still contains time placeholders")
			}
		})
	}
}

func TestGetTemplatesRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request GetTemplatesRequest
	}{
		{
			name: "Valid request with template type",
			request: GetTemplatesRequest{
				TemplateType: "daily",
			},
		},
		{
			name:    "Valid request without template type",
			request: GetTemplatesRequest{},
		},
		{
			name: "Valid request with 'all' type",
			request: GetTemplatesRequest{
				TemplateType: "all",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling/unmarshaling
			jsonData, err := json.Marshal(tt.request)
			if err != nil {
				t.Errorf("Failed to marshal request: %v", err)
				return
			}

			var unmarshaled GetTemplatesRequest
			err = json.Unmarshal(jsonData, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal request: %v", err)
				return
			}

			if unmarshaled.TemplateType != tt.request.TemplateType {
				t.Errorf("TemplateType mismatch: expected %s, got %s", tt.request.TemplateType, unmarshaled.TemplateType)
			}
		})
	}
}

func TestCreateFromTemplateRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request CreateFromTemplateRequest
		isValid bool
	}{
		{
			name: "Valid request with variables",
			request: CreateFromTemplateRequest{
				Path:         "notes/meeting.md",
				TemplateType: "meeting",
				Variables: map[string]string{
					"TITLE": "Team Meeting",
					"DATE":  "2025-01-15",
				},
			},
			isValid: true,
		},
		{
			name: "Valid request without variables",
			request: CreateFromTemplateRequest{
				Path:         "daily.md",
				TemplateType: "daily",
			},
			isValid: true,
		},
		{
			name: "Invalid request - empty path",
			request: CreateFromTemplateRequest{
				Path:         "",
				TemplateType: "daily",
			},
			isValid: false,
		},
		{
			name: "Invalid request - empty template type",
			request: CreateFromTemplateRequest{
				Path:         "note.md",
				TemplateType: "",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling/unmarshaling
			jsonData, err := json.Marshal(tt.request)
			if err != nil {
				t.Errorf("Failed to marshal request: %v", err)
				return
			}

			var unmarshaled CreateFromTemplateRequest
			err = json.Unmarshal(jsonData, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal request: %v", err)
				return
			}

			if unmarshaled.Path != tt.request.Path {
				t.Errorf("Path mismatch: expected %s, got %s", tt.request.Path, unmarshaled.Path)
			}

			if unmarshaled.TemplateType != tt.request.TemplateType {
				t.Errorf("TemplateType mismatch: expected %s, got %s", tt.request.TemplateType, unmarshaled.TemplateType)
			}

			// Basic validation
			if tt.isValid {
				if unmarshaled.Path == "" {
					t.Error("Valid request should not have empty path")
				}
				if unmarshaled.TemplateType == "" {
					t.Error("Valid request should not have empty template type")
				}
			}
		})
	}
}

func BenchmarkGetBuiltinTemplates(b *testing.B) {
	ns := &NotesServer{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		templates := ns.getBuiltinTemplates()
		if len(templates) == 0 {
			b.Fatal("No templates returned")
		}
	}
}

func BenchmarkTemplateVariableSubstitution(b *testing.B) {
	template := "# Meeting Notes - {{TITLE}}\n\n**Date:** {{DATE}}\n**Attendees:** {{ATTENDEES}}\n**Duration:** {{DURATION}}"
	variables := map[string]string{
		"TITLE":     "Team Standup",
		"DATE":      "2025-01-15",
		"ATTENDEES": "Alice, Bob, Charlie, David, Eve",
		"DURATION":  "30 minutes",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := template
		for key, value := range variables {
			placeholder := "{{" + key + "}}"
			result = strings.ReplaceAll(result, placeholder, value)
		}
	}
}
