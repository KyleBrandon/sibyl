# Sibyl MCP Servers - Usage Examples

This directory contains examples and documentation for using the Sibyl MCP servers.

## Overview

Sibyl provides two MCP servers for PDF processing and note management:

- **PDF Server**: Search, download, and convert PDFs from Google Drive
- **Notes Server**: Advanced note management with merge capabilities and templates

## Quick Start

### 1. Build the Servers

```bash
make all
```

This creates:
- `./bin/pdf-server` - PDF processing MCP server
- `./bin/notes-server` - Enhanced notes MCP server

### 2. Configure Your MCP Host

Add the servers to your MCP host configuration (e.g., Claude Desktop, VS Code, etc.):

```json
{
  "mcpServers": {
    "pdf-server": {
      "command": "./bin/pdf-server",
      "args": [
        "--credentials", "/path/to/google-credentials.json",
        "--folder-id", "your-google-drive-folder-id",
        "--log-level", "INFO"
      ]
    },
    "notes-server": {
      "command": "./bin/notes-server", 
      "args": [
        "--notesFolder", "/path/to/your/notes",
        "--logLevel", "INFO"
      ]
    }
  }
}
```

### 3. Interactive Workflow Examples

## Example 1: PDF to Markdown Conversion

**User**: "Help me convert a PDF about machine learning to my notes"

**LLM**: I'll help you convert a PDF to notes. Let me search for PDFs first.
→ *Calls `search_pdfs(query: "machine learning")`*

**LLM**: I found 3 PDFs about machine learning:
1. "ML Fundamentals.pdf" (2MB, modified yesterday)  
2. "Deep Learning Notes.pdf" (5MB, modified last week)
3. "ML Algorithms.pdf" (1MB, modified last month)

Which would you like to convert?

**User**: "The first one"

**LLM**: I'll convert "ML Fundamentals.pdf" to images for processing.
→ *Calls `convert_pdf_to_images(file_id: "abc123")`*

**LLM**: The PDF has 8 pages. Let me get appropriate conversion prompts.
→ *Calls `get_conversion_prompts(document_type: "mixed_content")`*

**LLM**: I'll now convert each page using the mixed content prompt optimized for documents with both text and visual elements.

[LLM processes each page with the appropriate prompt]

**LLM**: Here's the complete conversion. Would you like me to save this as a new note or merge it with an existing one?

**User**: "Save it as a new note called 'ML Fundamentals Summary'"

**LLM**: → *Calls `create_note(path: "ML Fundamentals Summary.md", content: markdown)`*

Perfect! I've saved your converted PDF as "ML Fundamentals Summary.md" in your vault.

## Example 2: Iterative Refinement

**User**: "The handwriting in section 2 wasn't converted well. Can you try again with a different approach?"

**LLM**: I'll re-process that section. Let me get a prompt specifically for handwritten content.
→ *Calls `get_conversion_prompts(document_type: "handwritten")`*

**LLM**: [Re-processes with handwriting-focused prompt]

How does this look now? I focused more on the handwritten elements and marked unclear sections.

**User**: "Much better! Now merge this with my existing research notes"

**LLM**: → *Calls `preview_merge(path: "research_notes.md", content: new_content)`*

**LLM**: Here's how the merge would look... Should I proceed?

**User**: "Yes, add it as a new section"

**LLM**: → *Calls `merge_note(path: "research_notes.md", content: new_content, strategy: "date_section")`*

## Available Resources

Both servers provide MCP resources for structured exploration of content:

### PDF Server Resources
- `pdf://documents/` - Collection of PDF documents with metadata
- `pdf://templates/` - Available conversion templates and prompts

### Notes Server Resources  
- `notes://files/` - Collection of note files with tags and previews
- `notes://templates/` - Available note templates
- `notes://collections/` - Notes organized by folders and tags

Resources enable LLMs to discover and explore content without knowing specific paths. See [RESOURCES.md](RESOURCES.md) for detailed documentation.

## Available Tools

### PDF Server Tools

- `search_pdfs(query, max_files)` - Search for PDF files in Google Drive
- `get_pdf_content(file_id)` - Get PDF file content and metadata
- `convert_pdf_to_images(file_id, dpi)` - Convert PDF to images for LLM processing
- `get_conversion_prompts(document_type)` - Get prompt templates for conversion
- `suggest_conversion_approach(file_id)` - Analyze PDF and suggest best approach

### Notes Server Tools

- `read_note(path)` - Read note content
- `write_note(path, content)` - Write/create a note
- `merge_note(path, content, strategy, title)` - Merge content with existing note
- `preview_merge(path, content, strategy)` - Preview merge operation
- `get_note_templates(template_type)` - Get available note templates
- `create_note_from_template(path, template_type, variables)` - Create note from template
- `list_notes(path, recursive)` - List notes in directory
- `search_notes(query, path)` - Search note content

### Merge Strategies

- `append` - Add content to end of file
- `prepend` - Add content to beginning of file  
- `date_section` - Add as new dated section
- `topic_merge` - Intelligent topic-based merging
- `replace` - Replace entire file content

### Note Templates

- `daily` - Daily note with tasks and reflections
- `meeting` - Meeting notes with agenda and action items
- `research` - Research notes with citations and analysis
- `project` - Project planning and tracking template

## Environment Variables

### PDF Server
- `GOOGLE_APPLICATION_CREDENTIALS` - Path to Google Cloud credentials
- `GCP_FOLDER_ID` - Google Drive folder ID to search

### Notes Server  
- `NOTE_SERVER_FOLDER` - Path to notes directory

## Advanced Usage

### Custom Prompts

The PDF server includes built-in prompts optimized for different document types:

- **Handwritten**: Optimized for handwritten notes and sketches
- **Typed**: Optimized for typed documents and printed materials  
- **Mixed**: Optimized for documents with both text and visual elements
- **Research**: Optimized for academic papers and research documents

### Intelligent Merging

The notes server provides sophisticated merging capabilities:

```
User: "Merge this content with my daily note using topic-based merging"
LLM: → merge_note(path: "daily.md", content: content, strategy: "topic_merge")
```

### Template Customization

Create notes from templates with variable substitution:

```
User: "Create a meeting note for the ML team standup"
LLM: → create_note_from_template(
  path: "meetings/ml-standup-2024-01-15.md",
  template_type: "meeting", 
  variables: {
    "TITLE": "ML Team Standup",
    "ATTENDEES": "Alice, Bob, Charlie",
    "DURATION": "30 minutes"
  }
)
```

This creates a properly formatted meeting note with all the standard sections and your specific details filled in.