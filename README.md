# Sibyl MCP Servers

**AI-powered PDF processing and note management through the Model Context Protocol (MCP)**

Sibyl provides two specialized MCP servers that enable LLM hosts (Claude Desktop, VS Code, etc.) to intelligently process PDFs and manage markdown notes with advanced capabilities like OCR, intelligent merging, and template-based creation.

## ⚖️ Important: Licensing Notice

**Before you begin:** Sibyl uses MuPDF (via go-fitz) for PDF processing, which is licensed under AGPL v3. This means:

- ✅ **Open source projects**: Free to use under AGPL v3
- ⚠️ **Commercial/proprietary use**: Requires commercial license from Artifex Software
- 📋 **Network deployment**: Must provide source code to users under AGPL v3

[See detailed licensing information below](#️-license--legal-notice) before deploying in production.

## 🚀 Quick Start

### Prerequisites

- **Go 1.21+** - [Install Go](https://golang.org/doc/install)
- **Google Cloud Account** - For Google Drive API access
- **Mathpix Account** - For OCR processing ([Sign up](https://mathpix.com/))
- **MCP-compatible host** - Claude Desktop, VS Code with MCP extension, etc.

### 1. Clone and Build

```bash
git clone https://github.com/your-username/sibyl.git
cd sibyl
make all
```

This creates:
- `./bin/pdf-server` - PDF processing MCP server
- `./bin/notes-server` - Note management MCP server

### 2. Set Up Credentials

#### Google Drive API Setup
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing one
3. Enable the Google Drive API
4. Create service account credentials
5. Download the JSON credentials file
6. Share your Google Drive folder with the service account email

#### Mathpix OCR Setup
1. Sign up at [Mathpix](https://mathpix.com/)
2. Create a new app in your dashboard
3. Note your App ID and App Key

### 3. Configure Environment

Create a `.env` file in the project root:

```bash
# Google Drive Configuration
GOOGLE_APPLICATION_CREDENTIALS=/path/to/your/google-credentials.json
GCP_FOLDER_ID=your-google-drive-folder-id

# Mathpix OCR Configuration  
MATHPIX_APP_ID=your-mathpix-app-id
MATHPIX_APP_KEY=your-mathpix-app-key
```

### 4. Configure Your MCP Host

Add the servers to your MCP host configuration:

**Claude Desktop (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):**

```json
{
  "mcpServers": {
    "pdf-server": {
      "command": "/full/path/to/sibyl/bin/pdf-server",
      "args": [
        "--credentials", "/path/to/google-credentials.json",
        "--folder-id", "your-google-drive-folder-id",
        "--mathpix-app-id", "your-mathpix-app-id",
        "--mathpix-app-key", "your-mathpix-app-key",
        "--ocr-languages", "en,fr,de",
        "--log-level", "INFO"
      ]
    },
    "notes-server": {
      "command": "/full/path/to/sibyl/bin/notes-server",
      "args": [
        "--notesFolder", "/path/to/your/notes",
        "--logLevel", "INFO",
        "--logFile", "notes-server.log"
      ]
    }
  }
}
```

### 5. Test the Setup

1. Restart your MCP host (Claude Desktop, VS Code, etc.)
2. Try asking: *"What PDF documents do I have available?"*
3. Try asking: *"Show me my notes in the projects folder"*

## 📖 Core Concepts

### PDF Server Capabilities

The PDF server connects to your Google Drive and provides:

- **🔍 Document Search**: Find PDFs by name, content, or metadata
- **📄 PDF-to-Markdown Conversion**: High-quality conversion using Mathpix OCR + visual analysis
- **🖼️ Image Processing**: Converts PDFs to 150 DPI PNG images for visual analysis
- **🌍 Multi-language OCR**: Supports 40+ languages through Mathpix
- **📊 MCP Resources**: Structured access to your document library

**How it works:**
1. **PDF → PNG**: Converts PDF pages to high-quality images using MuPDF
2. **PDF → OCR**: Extracts text using Mathpix OCR (handles formulas, tables, handwriting)
3. **PNG + OCR → LLM**: Your LLM analyzes both visual and text data for optimal conversion

### Notes Server Capabilities

The notes server manages your markdown vault with:

- **📝 Intelligent Reading/Writing**: Full CRUD operations on markdown files
- **🔄 Smart Merging**: 5 merge strategies (append, prepend, date_section, topic_merge, replace)
- **👀 Merge Preview**: See exactly what changes before applying them
- **📋 Template System**: Pre-built templates for daily notes, meetings, research, projects
- **🔍 Content Search**: Full-text search across your entire vault
- **📊 MCP Resources**: Structured exploration of your note collection

## 🛠️ Available Tools

### PDF Server Tools

| Tool | Description | Parameters |
|------|-------------|------------|
| `search_pdfs` | Search Google Drive for PDF files | `query` (string), `max_files` (number) |
| `convert_pdf_to_markdown` | Convert PDF to Markdown using OCR + visual analysis | `file_id` (string) |

### Notes Server Tools

| Tool | Description | Parameters |
|------|-------------|------------|
| `read_note` | Read note content | `path` (string) |
| `write_note` | Create or overwrite a note | `path` (string), `content` (string) |
| `merge_note` | Merge content with existing note | `path`, `content`, `strategy`, `title?` |
| `preview_merge` | Preview merge operation | `path`, `content`, `strategy?` |
| `list_notes` | List notes in directory | `path?`, `recursive?` (boolean) |
| `search_notes` | Search note content | `query` (string), `path?` |
| `get_note_templates` | Get available templates | `template_type?` (string) |
| `create_note_from_template` | Create note from template | `path`, `template_type`, `variables?` |

### Merge Strategies

- **`append`** - Add content to end of file
- **`prepend`** - Add content to beginning of file  
- **`date_section`** - Add as new dated section with timestamp
- **`topic_merge`** - Intelligent merging based on content topics
- **`replace`** - Replace entire file content

### Note Templates

- **`daily`** - Daily note with focus areas, tasks, and reflections
- **`meeting`** - Meeting notes with agenda, attendees, and action items
- **`research`** - Research notes with citations, analysis, and conclusions
- **`project`** - Project planning template with goals, milestones, and progress

## 🎯 Usage Examples

### Example 1: PDF Research Workflow

**You**: *"I have a research paper about machine learning in my Drive. Can you convert it to notes and add it to my research collection?"*

**Assistant**: I'll help you find and convert that research paper. Let me search your Google Drive first.

*→ Calls `search_pdfs(query: "machine learning")`*

I found 3 PDFs about machine learning:
1. "Deep Learning Fundamentals.pdf" (2.1MB, modified yesterday)
2. "ML Algorithms Survey.pdf" (896KB, modified last week)  
3. "Neural Networks Intro.pdf" (1.5MB, modified last month)

Which would you like me to convert?

**You**: *"The first one"*

**Assistant**: I'll convert "Deep Learning Fundamentals.pdf" to Markdown using our OCR and visual analysis process.

*→ Calls `convert_pdf_to_markdown(file_id: "1BxYz...")`*

Perfect! The PDF has been processed with Mathpix OCR and converted to high-quality images. I've extracted the text and can see the visual elements including equations, diagrams, and tables.

Here's the converted content... [shows converted markdown]

Would you like me to save this as a new research note?

**You**: *"Yes, save it as 'Deep Learning Fundamentals Summary'"*

**Assistant**: *→ Calls `write_note(path: "research/Deep Learning Fundamentals Summary.md", content: converted_content)`*

Done! I've saved the converted paper as "Deep Learning Fundamentals Summary.md" in your research folder. The conversion includes all the mathematical formulas, diagrams, and structured content from the original PDF.

### Example 2: Smart Note Merging

**You**: *"I just took some meeting notes. Can you merge them with my daily note but keep things organized?"*

**Assistant**: I can help merge your meeting notes. Let me preview how this would look with topic-based merging to keep everything organized.

*→ Calls `preview_merge(path: "daily.md", content: meeting_notes, strategy: "topic_merge")`*

Here's how the merge would look:
- Your existing "Tasks" section would be combined with new action items
- Meeting notes would be added under a new "Meetings" section  
- No duplicate content would be created

Should I proceed with this merge?

**You**: *"Perfect, go ahead"*

**Assistant**: *→ Calls `merge_note(path: "daily.md", content: meeting_notes, strategy: "topic_merge")`*

Done! Your meeting notes have been intelligently merged into your daily note with proper organization and no duplicates.

### Example 3: Template-Based Note Creation

**You**: *"Create a meeting note for tomorrow's sprint planning session"*

**Assistant**: I'll create a meeting note using our meeting template.

*→ Calls `create_note_from_template(path: "meetings/sprint-planning-2025-08-10.md", template_type: "meeting", variables: {"TITLE": "Sprint Planning", "DATE": "2025-08-10", "ATTENDEES": "TBD"})`*

I've created "sprint-planning-2025-08-10.md" with a structured meeting template including:
- Agenda section
- Attendees list (you can fill this in)  
- Discussion points
- Action items tracking
- Next steps

The note is ready for you to add specific agenda items before the meeting.

## 📊 MCP Resources

Both servers provide MCP resources that enable intelligent exploration of your content without knowing specific file paths.

### PDF Server Resources

- **`pdf://documents/`** - Complete catalog of your PDF library with metadata
- **Example**: Lists all PDFs with file sizes, modification dates, and direct links

### Notes Server Resources  

- **`notes://files/`** - Your complete note collection with previews and tags
- **`notes://templates/`** - Available note templates with descriptions  
- **`notes://collections/`** - Notes organized by folders and tags

**Resource Benefits:**
- 🔍 **Discovery**: LLMs can explore without knowing file paths
- ⚡ **Efficiency**: Batch metadata retrieval vs individual queries
- 🧠 **Context**: Rich metadata helps LLMs make smarter decisions
- 🗺️ **Navigation**: Structured browsing of content hierarchies

## ⚙️ Configuration Options

### PDF Server Arguments

```bash
./bin/pdf-server --help
```

| Argument | Required | Description | Environment Variable |
|----------|----------|-------------|---------------------|
| `--credentials` | Yes | Path to Google Cloud credentials JSON | `GOOGLE_APPLICATION_CREDENTIALS` |
| `--folder-id` | Yes | Google Drive folder ID to search | `GCP_FOLDER_ID` |
| `--mathpix-app-id` | Yes | Mathpix API App ID | `MATHPIX_APP_ID` |
| `--mathpix-app-key` | Yes | Mathpix API App Key | `MATHPIX_APP_KEY` |
| `--ocr-languages` | No | Comma-separated language codes (default: "en") | - |
| `--log-level` | No | Log level: DEBUG, INFO, WARN, ERROR (default: INFO) | - |
| `--log-file` | No | Log file path (default: stderr) | - |

### Notes Server Arguments

```bash
./bin/notes-server --help
```

| Argument | Required | Description |
|----------|----------|-------------|
| `--notesFolder` | Yes | Path to your notes directory |
| `--logLevel` | No | Log level: DEBUG, INFO, WARN, ERROR (default: INFO) |
| `--logFile` | No | Log file path (default: stderr) |

## 🧪 Development & Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -race -coverprofile=coverage.out -covermode=atomic ./pkg/...
go tool cover -html=coverage.out -o coverage.html

# Run specific package tests
go test ./pkg/pdfmcp
go test ./pkg/notes

# Run integration tests
go test ./tests/integration/...
```

### Code Quality

```bash
# Static analysis
go vet ./...
staticcheck ./...

# Security scanning  
gosec ./...

# Comprehensive linting
golangci-lint run --timeout=5m
```

### Project Structure

```
sibyl/
├── cmd/                    # Main applications
│   ├── pdfserver/          # PDF MCP server entry point
│   └── noteserver/         # Notes MCP server entry point
├── pkg/                    # Reusable packages
│   ├── pdfmcp/             # PDF server implementation 
│   ├── notes/              # Notes server implementation
│   ├── dto/                # Data transfer objects
│   └── utils/              # Shared utilities
├── tests/                  # Testing infrastructure
│   ├── integration/        # End-to-end tests
│   └── testutils/          # Test helpers
├── examples/               # Usage examples and configs
└── bin/                    # Built binaries
```

## 🔧 Troubleshooting

### Common Issues

**PDF Server won't start:**
- ✅ Check Google Cloud credentials file exists and is valid
- ✅ Verify Google Drive API is enabled in your GCP project
- ✅ Ensure service account has access to your Drive folder
- ✅ Confirm Mathpix credentials are correct

**Notes Server can't find notes:**
- ✅ Verify notes folder path exists and is readable
- ✅ Check that notes are in Markdown format (.md extension)
- ✅ Ensure proper file permissions

**MCP Host can't connect:**
- ✅ Use absolute paths in MCP configuration
- ✅ Restart your MCP host after configuration changes
- ✅ Check server logs for startup errors

### Log Files

Enable debug logging for troubleshooting:

```bash
# PDF Server
./bin/pdf-server --log-level DEBUG --log-file pdf-server.log [other args...]

# Notes Server  
./bin/notes-server --logLevel DEBUG --logFile notes-server.log [other args...]
```

## 📚 Advanced Usage

### Custom OCR Languages

Mathpix supports 40+ languages. Specify multiple languages:

```bash
--ocr-languages "en,fr,de,es,zh,ja,ko"
```

### Batch Processing

Process multiple PDFs efficiently:

1. Use `pdf://documents/` resource to get complete file list
2. Filter by date, size, or name patterns  
3. Process each with `convert_pdf_to_markdown`
4. Use `merge_note` with `topic_merge` strategy for intelligent consolidation

### Custom Templates

Create your own note templates by examining the built-in ones:

```bash
# Get template structure
curl -X POST http://localhost/mcp \
  -d '{"method": "tools/call", "params": {"name": "get_note_templates"}}'
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes and add tests
4. Ensure all tests pass: `make test`
5. Commit your changes: `git commit -m 'Add amazing feature'`
6. Push to the branch: `git push origin feature/amazing-feature`
7. Open a Pull Request

### Code Style

- Follow Go conventions and use `gofmt`
- Add tests for new functionality
- Update documentation for user-facing changes
- Use structured logging with `slog`

## ⚖️ License & Legal Notice

### Primary License
This project is licensed under the **Apache License 2.0** - see the [LICENSE](LICENSE) file for details.

### 🚨 Important: Third-Party Licensing Dependencies

**Sibyl uses MuPDF for PDF processing, which has specific licensing requirements:**

#### **MuPDF Licensing (AGPL v3)**
- **Library**: `github.com/gen2brain/go-fitz` (Go wrapper for MuPDF)
- **Underlying Library**: MuPDF (licensed under AGPL v3)
- **Licensor**: Artifex Software, Inc.

#### **AGPL v3 Requirements**
⚠️ **If you use Sibyl in any of the following ways, your entire application MUST be licensed under AGPL v3:**

1. **Network Services** - Running Sibyl as a web service, API, or SaaS
2. **Modified Distribution** - Distributing modified versions of Sibyl
3. **Integration** - Incorporating Sibyl into other software
4. **Internal Use** - Using Sibyl within an organization over a network

#### **AGPL v3 Obligations**
When AGPL applies, you must:
- ✅ License your entire application under AGPL v3 or compatible
- ✅ Provide complete source code to all users
- ✅ Include license notices and copyright information
- ✅ Ensure users can rebuild your application from source

#### **Commercial Licensing Alternative**
If you cannot comply with AGPL v3 requirements:

🏢 **Contact Artifex Software for a commercial MuPDF license:**
- Website: https://mupdf.com/licensing/
- Removes AGPL obligations
- Enables proprietary/commercial deployment
- Required for closed-source applications

#### **Other Dependencies (Permissively Licensed)**
All other dependencies use permissive licenses:
- `github.com/mark3labs/mcp-go` - MIT License
- `github.com/joho/godotenv` - MIT License  
- `google.golang.org/api` - Apache License 2.0
- Go standard library - BSD-style License

### **License Compatibility Summary**

| Use Case | License Required | Commercial License Needed |
|----------|------------------|---------------------------|
| Open source project (AGPL v3) | ✅ Free | ❌ No |
| Internal company tool | ⚠️ AGPL v3 | ❌ No* |
| SaaS/Web service | ⚠️ AGPL v3 | ❌ No* |
| Proprietary software | ❌ Not possible | ✅ Yes |
| Commercial distribution | ❌ Not possible | ✅ Yes |

*As long as you comply with AGPL v3 source distribution requirements

### **Recommendation for Users**
1. **Open Source Projects**: Use Sibyl freely under AGPL v3
2. **Commercial Projects**: Evaluate if AGPL v3 compliance is feasible
3. **Proprietary Products**: Contact Artifex for commercial licensing
4. **Consulting**: Consider legal review for complex licensing scenarios

For questions about licensing compliance, consult with a qualified legal professional.

## 🙏 Acknowledgments

- [MCP Go SDK](https://github.com/mark3labs/mcp-go) - MCP server implementation
- [Mathpix](https://mathpix.com/) - OCR processing service
- [MuPDF](https://mupdf.com/) - PDF processing and rendering
- [Google Drive API](https://developers.google.com/drive) - Document storage and access