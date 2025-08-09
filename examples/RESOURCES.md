# MCP Resources Reference

Technical reference for Sibyl's MCP resources - the structured data endpoints that enable LLM exploration and discovery.

## PDF Server Resources

### `pdf://documents/`
**Type**: Collection  
**MIME Type**: `application/json`  
**Description**: Complete PDF document catalog

**Schema**:
```json
[
  {
    "id": "string",           // Google Drive file ID
    "name": "string",         // Filename
    "size": number,           // File size in bytes  
    "created": "string",      // ISO 8601 timestamp
    "modified": "string",     // ISO 8601 timestamp
    "webViewLink": "string",  // Direct Google Drive link
    "thumbnailLink": "string", // Preview image URL
    "uri": "string"           // MCP resource URI
  }
]
```

**Example Response**:
```json
[
  {
    "id": "1BxYz4cR9tF3k2mN8qW7sV6uA5bP",
    "name": "machine-learning-paper.pdf", 
    "size": 2048576,
    "created": "2025-01-15T10:30:00Z",
    "modified": "2025-01-15T14:22:00Z",
    "webViewLink": "https://drive.google.com/file/d/1BxYz.../view",
    "thumbnailLink": "https://drive.google.com/thumbnail?id=1BxYz...",
    "uri": "pdf://documents/1BxYz4cR9tF3k2mN8qW7sV6uA5bP"
  }
]
```

## Notes Server Resources

### `notes://files/`
**Type**: Collection  
**MIME Type**: `application/json`  
**Description**: Complete notes collection with metadata

**Schema**:
```json
[
  {
    "path": "string",         // Relative file path
    "name": "string",         // Filename
    "size": number,           // File size in bytes
    "modified": "string",     // ISO 8601 timestamp  
    "uri": "string",          // MCP resource URI
    "tags": ["string"],       // Extracted YAML frontmatter tags
    "preview": "string"       // First 200 chars of content
  }
]
```

### `notes://templates/`
**Type**: Dictionary  
**MIME Type**: `application/json`  
**Description**: Available note templates

**Schema**:
```json
{
  "template_id": {
    "name": "string",         // Display name
    "description": "string",  // Purpose description
    "use_case": "string",     // When to use this template  
    "uri": "string",          // MCP resource URI
    "content": "string"       // Template content with {{VARIABLES}}
  }
}
```

**Example Response**:
```json
{
  "daily": {
    "name": "Daily Note",
    "description": "Template for daily notes with focus areas and tasks",
    "use_case": "Daily planning, journaling, task tracking",
    "uri": "notes://templates/daily",
    "content": "# Daily Note - {{DATE}}\n\n## Today's Focus\n- {{FOCUS_1}}\n- {{FOCUS_2}}\n\n## Tasks\n- [ ] {{TASK_1}}\n- [ ] {{TASK_2}}\n\n## Notes\n{{NOTES}}\n\n## Reflection\n{{REFLECTION}}"
  },
  "meeting": {
    "name": "Meeting Notes",
    "description": "Structured meeting documentation with action items",
    "use_case": "Meeting documentation, action item tracking",
    "uri": "notes://templates/meeting",
    "content": "# Meeting Notes - {{TITLE}}\n\n**Date:** {{DATE}}  \n**Attendees:** {{ATTENDEES}}  \n**Duration:** {{DURATION}}\n\n## Agenda\n{{AGENDA}}\n\n## Discussion\n{{DISCUSSION}}\n\n## Decisions\n{{DECISIONS}}\n\n## Action Items\n- [ ] {{ACTION_1}} - {{ASSIGNEE_1}}\n- [ ] {{ACTION_2}} - {{ASSIGNEE_2}}\n\n## Next Steps\n{{NEXT_STEPS}}"
  }
}
```

### `notes://collections/`
**Type**: Dictionary  
**MIME Type**: `application/json`  
**Description**: Notes organized by folders and tags

**Schema**:
```json
{
  "folders": {
    "folder_name": ["file_path"]  // Files grouped by directory
  },
  "tags": {
    "tag_name": ["file_path"]     // Files grouped by YAML tags
  }
}
```

## Resource Usage Patterns

### Discovery Pattern
```javascript
// LLM explores available content
const docs = await mcp.readResource('pdf://documents/');
const notes = await mcp.readResource('notes://files/');

// Filter by metadata
const recentDocs = docs.filter(d => 
  new Date(d.modified) > new Date('2025-01-01')
);
```

### Navigation Pattern  
```javascript
// Browse organized collections
const collections = await mcp.readResource('notes://collections/');

// Find all project-related notes
const projectNotes = collections.tags.project || [];
const projectFolderNotes = collections.folders.projects || [];
```

### Template Selection Pattern
```javascript
// Get available templates
const templates = await mcp.readResource('notes://templates/');

// Select appropriate template
const meetingTemplate = templates.meeting;
const variables = {
  TITLE: "Sprint Planning",
  DATE: "2025-08-10", 
  ATTENDEES: "Team Alpha"
};
```

## HTTP Response Codes

| Code | Meaning | Action |
|------|---------|--------|
| 200 | Success | Resource data returned |
| 404 | Not Found | Resource URI invalid |
| 403 | Forbidden | Access denied (credentials) |
| 500 | Server Error | Check server logs |

## Error Responses

```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Resource 'pdf://invalid/' not found",
    "details": "Check available resource URIs"
  }
}
```

## Performance Notes

- **Caching**: Resources are cached for 60 seconds
- **Pagination**: Large collections (>1000 items) are paginated
- **Lazy Loading**: Preview content is truncated to 200 characters
- **Filtering**: Client-side filtering recommended for performance

## Integration Examples

### Claude Desktop
Resources appear automatically in the MCP resource panel.

### Custom Clients
```python
import mcp_client

client = mcp_client.connect('stdio', ['./bin/pdf-server'])
documents = client.read_resource('pdf://documents/')
```

### REST API Wrapper
```bash
curl -X POST http://localhost:8080/mcp \
  -d '{"method": "resources/read", "params": {"uri": "pdf://documents/"}}'
```