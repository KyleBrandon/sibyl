package notes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// NoteTemplate represents a note template
type NoteTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	UseCase     string `json:"use_case"`
}

// GetTemplatesRequest represents a request for note templates
type GetTemplatesRequest struct {
	TemplateType string `json:"template_type,omitempty" mcp:"Type of template: daily, meeting, research, or all"`
}

// CreateFromTemplateRequest represents a request to create a note from a template
type CreateFromTemplateRequest struct {
	Path         string            `json:"path" mcp:"Path for the new note"`
	TemplateType string            `json:"template_type" mcp:"Template type to use"`
	Variables    map[string]string `json:"variables,omitempty" mcp:"Variables to substitute in template"`
}

func (ns *NotesServer) NewGetTemplatesTools() {
	tool := mcp.NewTool(
		"get_note_templates",
		mcp.WithDescription("Get available note templates"),
		mcp.WithString("template_type", mcp.Description("Template type: daily, meeting, research, or all")),
	)

	ns.McpServer.AddTool(tool, mcp.NewTypedToolHandler(ns.GetNoteTemplates))
}

func (ns *NotesServer) NewCreateFromTemplateTool() {
	tool := mcp.NewTool(
		"create_note_from_template",
		mcp.WithDescription("Create a new note from a template"),
		mcp.WithString("path", mcp.Description("Path for the new note"), mcp.Required()),
		mcp.WithString("template_type", mcp.Description("Template type to use"), mcp.Required()),
		mcp.WithObject("variables", mcp.Description("Variables to substitute in template")),
	)

	ns.McpServer.AddTool(tool, mcp.NewTypedToolHandler(ns.CreateNoteFromTemplate))
}

func (ns *NotesServer) GetNoteTemplates(ctx context.Context, req mcp.CallToolRequest, params GetTemplatesRequest) (*mcp.CallToolResult, error) {
	templates := ns.getBuiltinTemplates()

	if params.TemplateType != "" && params.TemplateType != "all" {
		// Filter to specific template type
		if template, exists := templates[params.TemplateType]; exists {
			filteredTemplates := map[string]NoteTemplate{
				params.TemplateType: template,
			}
			templatesJSON, _ := json.MarshalIndent(filteredTemplates, "", "  ")
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(string(templatesJSON)),
				},
			}, nil
		} else {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Template type '%s' not found", params.TemplateType)),
				},
			}, nil
		}
	}

	// Return all templates
	templatesJSON, _ := json.MarshalIndent(templates, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(templatesJSON)),
		},
	}, nil
}

func (ns *NotesServer) CreateNoteFromTemplate(ctx context.Context, req mcp.CallToolRequest, params CreateFromTemplateRequest) (*mcp.CallToolResult, error) {
	templates := ns.getBuiltinTemplates()

	template, exists := templates[params.TemplateType]
	if !exists {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Template type '%s' not found", params.TemplateType)),
			},
		}, nil
	}

	// Substitute variables in template
	content := ns.substituteVariables(template.Content, params.Variables)

	// Create the note using existing write functionality
	writeParams := WriteNoteRequest{
		Path:    params.Path,
		Content: content,
	}

	return ns.WriteNote(ctx, req, writeParams)
}

func (ns *NotesServer) getBuiltinTemplates() map[string]NoteTemplate {
	now := time.Now()

	return map[string]NoteTemplate{
		"daily": {
			Name:        "Daily Note",
			Description: "Template for daily notes with common sections",
			UseCase:     "Daily journaling, task tracking, quick notes",
			Content: fmt.Sprintf(`# Daily Note - %s

## Today's Focus
- 

## Tasks
- [ ] 
- [ ] 
- [ ] 

## Notes


## Reflections


## Tomorrow's Priorities
- 
- 
- 

---
*Created: %s*`, now.Format("2006-01-02"), now.Format("2006-01-02 15:04")),
		},

		"meeting": {
			Name:        "Meeting Notes",
			Description: "Template for meeting notes with agenda and action items",
			UseCase:     "Meeting documentation, action item tracking",
			Content: fmt.Sprintf(`# Meeting Notes - {{TITLE}}

**Date:** %s  
**Attendees:** {{ATTENDEES}}  
**Duration:** {{DURATION}}

## Agenda
1. 
2. 
3. 

## Discussion


## Decisions Made
- 

## Action Items
- [ ] {{ACTION_ITEM}} - {{ASSIGNEE}} - {{DUE_DATE}}
- [ ] 
- [ ] 

## Next Steps


---
*Meeting notes created: %s*`, now.Format("2006-01-02"), now.Format("2006-01-02 15:04")),
		},

		"research": {
			Name:        "Research Notes",
			Description: "Template for research and study notes",
			UseCase:     "Academic research, learning notes, literature review",
			Content: fmt.Sprintf(`# Research Notes - {{TOPIC}}

**Source:** {{SOURCE}}  
**Date:** %s  
**Tags:** {{TAGS}}

## Summary


## Key Points
- 
- 
- 

## Quotes & References
> 

## My Thoughts


## Related Topics
- 
- 

## Follow-up Questions
- 
- 

---
*Research notes created: %s*`, now.Format("2006-01-02"), now.Format("2006-01-02 15:04")),
		},

		"project": {
			Name:        "Project Notes",
			Description: "Template for project planning and tracking",
			UseCase:     "Project management, planning, progress tracking",
			Content: fmt.Sprintf(`# Project: {{PROJECT_NAME}}

**Status:** {{STATUS}}  
**Start Date:** {{START_DATE}}  
**Due Date:** {{DUE_DATE}}  
**Owner:** {{OWNER}}

## Objective


## Scope
### In Scope
- 

### Out of Scope
- 

## Milestones
- [ ] {{MILESTONE}} - {{DATE}}
- [ ] 
- [ ] 

## Tasks
- [ ] 
- [ ] 
- [ ] 

## Resources Needed


## Risks & Mitigation


## Progress Log
### %s


---
*Project notes created: %s*`, now.Format("2006-01-02"), now.Format("2006-01-02 15:04")),
		},
	}
}

func (ns *NotesServer) substituteVariables(content string, variables map[string]string) string {
	if variables == nil {
		return content
	}

	result := content
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}
