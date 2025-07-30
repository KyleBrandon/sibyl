# MCP Resources Guide

This document describes the MCP resources available in the Sibyl servers, which provide structured access to documents and data for LLM exploration.

## PDF Server Resources

The PDF server provides resources for exploring and working with PDF documents stored in Google Drive.

### Available Resources

#### 1. PDF Documents Collection (`pdf://documents/`)
Lists all PDF documents in the configured Google Drive folder.

**Resource URI:** `pdf://documents/`  
**MIME Type:** `application/json`  
**Description:** Collection of PDF documents in Google Drive folder

**Example Response:**
```json
[
  {
    "id": "1BxYz...",
    "name": "research-paper.pdf",
    "size": 2048576,
    "created": "2025-01-15T10:30:00Z",
    "modified": "2025-01-15T10:30:00Z",
    "webViewLink": "https://drive.google.com/file/d/1BxYz.../view",
    "thumbnailLink": "https://drive.google.com/thumbnail?id=1BxYz...",
    "uri": "pdf://documents/1BxYz..."
  }
]
```

#### 2. Conversion Templates (`pdf://templates/`)
Available PDF to Markdown conversion templates with specialized prompts.

**Resource URI:** `pdf://templates/`  
**MIME Type:** `application/json`  
**Description:** Available PDF to Markdown conversion templates

**Example Response:**
```json
{
  "handwritten": {
    "name": "Handwritten Notes",
    "description": "Optimized for converting handwritten notes and sketches",
    "uri": "pdf://templates/handwritten",
    "type": "handwritten",
    "use_case": "Personal notes, meeting notes, sketches, diagrams",
    "prompt": "You are an expert at converting handwritten notes..."
  },
  "typed": {
    "name": "Typed Documents",
    "description": "Optimized for typed documents and papers",
    "uri": "pdf://templates/typed",
    "type": "typed", 
    "use_case": "Academic papers, reports, articles",
    "prompt": "You are an expert at converting typed documents..."
  }
}
```

## Notes Server Resources

The Notes server provides resources for exploring and organizing markdown notes in your vault.

### Available Resources

#### 1. Note Files Collection (`notes://files/`)
Lists all markdown files in the notes vault with metadata and previews.

**Resource URI:** `notes://files/`  
**MIME Type:** `application/json`  
**Description:** Collection of markdown note files in the vault

**Example Response:**
```json
[
  {
    "path": "projects/sibyl.md",
    "name": "sibyl.md",
    "size": 1024,
    "modified": "2025-01-15T14:30:00Z",
    "uri": "notes://files/projects/sibyl.md",
    "tags": ["development", "mcp", "ai"],
    "preview": "# Sibyl MCP Project This project implements MCP servers for PDF processing..."
  }
]
```

#### 2. Note Templates (`notes://templates/`)
Available note templates for different purposes and workflows.

**Resource URI:** `notes://templates/`  
**MIME Type:** `application/json`  
**Description:** Available note templates for different purposes

**Example Response:**
```json
{
  "daily": {
    "name": "Daily Note",
    "description": "Template for daily notes with common sections",
    "use_case": "Daily journaling, task tracking, quick notes",
    "uri": "notes://templates/daily",
    "content": "# Daily Note - {{DATE}}\n\n## Today's Focus\n..."
  },
  "meeting": {
    "name": "Meeting Notes", 
    "description": "Template for meeting notes with agenda and action items",
    "use_case": "Meeting documentation, action item tracking",
    "uri": "notes://templates/meeting",
    "content": "# Meeting Notes - {{TITLE}}\n\n**Date:** {{DATE}}..."
  }
}
```

#### 3. Note Collections (`notes://collections/`)
Organized collections of notes grouped by folders and tags.

**Resource URI:** `notes://collections/`  
**MIME Type:** `application/json`  
**Description:** Grouped collections of notes by tags and folders

**Example Response:**
```json
{
  "folders": {
    "projects": ["projects/sibyl.md", "projects/other.md"],
    "meetings": ["meetings/2025-01-15.md"],
    "root": ["daily.md", "inbox.md"]
  },
  "tags": {
    "development": ["projects/sibyl.md", "daily.md"],
    "meeting": ["meetings/2025-01-15.md"],
    "important": ["inbox.md"]
  }
}
```

## Using Resources in MCP Clients

### Discovery
Resources enable LLMs to discover available content without knowing specific file paths:

```
LLM: "What PDF documents are available?"
→ Reads pdf://documents/ resource
→ Gets list of all PDFs with metadata

LLM: "Show me all notes tagged with 'project'"  
→ Reads notes://collections/ resource
→ Finds notes in tags.project array
```

### Navigation
Resources provide structured navigation through content:

```
LLM: "What note templates can I use?"
→ Reads notes://templates/ resource  
→ Shows available templates with descriptions

LLM: "What's the best way to convert this handwritten PDF?"
→ Reads pdf://templates/ resource
→ Suggests handwritten template with specialized prompt
```

### Metadata-Rich Exploration
Resources include rich metadata for intelligent decision making:

- **File sizes** - Help determine processing approach
- **Modification dates** - Find recent or outdated content  
- **Tags and folders** - Understand content organization
- **Previews** - Quick content assessment without full reads
- **URIs** - Direct links to specific resources

## Benefits Over Tool-Only Approach

1. **Discoverability** - LLMs can explore available content
2. **Efficiency** - Batch metadata retrieval vs individual tool calls
3. **Context** - Rich metadata helps LLMs make better decisions
4. **Navigation** - Structured browsing of content hierarchies
5. **Real-time** - Resources reflect current state of files

## Configuration

Resources are automatically enabled when you start the MCP servers. No additional configuration is required beyond the standard server setup.

The resources will reflect the current state of your:
- Google Drive folder (for PDF server)
- Notes vault directory (for Notes server)