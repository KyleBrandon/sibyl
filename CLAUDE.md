# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Sibyl is a dual MCP (Model Context Protocol) server application providing PDF processing and notes management capabilities. The project implements two separate MCP servers that can be used interactively by any MCP-compatible LLM host.

### Architecture

```
┌─────────────────┐    ┌───────────────────┐    ┌──────────────────┐
│   LLM HOST      │    │  PDF-MCP-SERVER   │    │  NOTES-MCP-SERVER│
│ (Claude/ChatGPT)│◄──►│                   │    │                  │
│                 │    │ • search_pdfs     │    │ • merge_note     │
│                 │    │ • convert_to_img  │    │ • preview_merge  │
│                 │    │ • get_prompts     │    │ • templates      │
│                 │    │ • suggest_approach│    │ • smart_create   │
└─────────────────┘    └───────────────────┘    └──────────────────┘
```

## Development Commands

### Build Commands
- `make all` - Build both MCP servers (pdf_server, notes_server)
- `make pdf_server` - Build PDF server to ./bin/pdf-server  
- `make notes_server` - Build notes server to ./bin/notes-server
- `make clean` - Remove all binaries from ./bin/

### Testing Commands
- `go test ./...` - Run all tests (11 test packages, 4,852+ lines of test code)
- `go test ./pkg/notes` - Run tests for specific package
- `go test -v -run TestMergeStrategies ./pkg/notes` - Run single test function
- `go test ./tests/integration/...` - Run integration tests
- `go test ./cmd/...` - Run CLI tests
- `go test -race -coverprofile=coverage.out -covermode=atomic ./pkg/...` - Run tests with coverage
- `go tool cover -html=coverage.out -o coverage.html` - Generate coverage report

### Quality Assurance Commands
- `go vet ./...` - Run go vet static analysis
- `staticcheck ./...` - Run staticcheck linter (install with `go install honnef.co/go/tools/cmd/staticcheck@latest`)
- `golangci-lint run --timeout=5m` - Run comprehensive linting
- `gosec ./...` - Run security scanner
- `go list -json -deps ./... | nancy sleuth` - Run vulnerability scanner

### Server Execution
- `./bin/pdf-server --help` - View PDF server options
- `./bin/notes-server --help` - View notes server options

## Project Structure

### Core Directories
- `cmd/` - Main applications
  - `cmd/pdf-server/` - PDF processing MCP server entry point
  - `cmd/note_server/` - Notes management MCP server entry point
- `pkg/` - Reusable packages
  - `pkg/dto/` - Data transfer objects (note.go, gcp.go)
  - `pkg/pdf-mcp/` - PDF processing MCP server implementation with OCR support
  - `pkg/notes/` - Notes management MCP server implementation
  - `pkg/utils/` - Shared utilities
- `tests/` - Testing infrastructure
  - `tests/integration/` - Integration tests
  - `tests/testutils/` - Test utilities and helpers
- `examples/` - Configuration examples and documentation

### Key Implementation Files
- `pkg/pdf-mcp/server.go` - PDF MCP server implementation
- `pkg/notes/server.go` - Notes MCP server implementation  
- `pkg/notes/merge.go` - Note merging strategies (5 different approaches)
- `pkg/notes/templates.go` - Note template system
- `pkg/pdf-mcp/ocr.go` - OCR integration (Mathpix, mock engines)
- `pkg/pdf-mcp/conversion.go` - PDF to image conversion using MuPDF

## MCP Server Configuration

### PDF Server
The PDF server provides document processing capabilities with OCR support:

```bash
./bin/pdf-server \
  --credentials "/path/to/google-credentials.json" \
  --folder-id "google-drive-folder-id" \
  --ocr-engine "mathpix" \
  --ocr-languages "en,fr,de" \
  --mathpix-app-id "your-app-id" \
  --mathpix-app-key "your-app-key" \
  --log-level "INFO"
```

Environment variables: `GOOGLE_APPLICATION_CREDENTIALS`, `GCP_FOLDER_ID`, `MATHPIX_APP_ID`, `MATHPIX_APP_KEY`

### Notes Server
The notes server provides intelligent note management:

```bash
./bin/notes-server \
  --notesFolder "/path/to/notes" \
  --logLevel "INFO" \
  --logFile "notes-server.log"
```

## Key Features

### PDF Server Capabilities
- **Document Search**: Search Google Drive for PDFs by query
- **Image Conversion**: High-quality PDF-to-image conversion using MuPDF (go-fitz)
- **OCR Processing**: Mathpix integration for document text extraction
- **Smart Prompts**: Document-type-specific conversion prompts
- **Hybrid Processing**: Combines OCR with LLM vision analysis

### Notes Server Capabilities  
- **Intelligent Merging**: 5 merge strategies (append, prepend, date_section, topic_merge, replace)
- **Merge Preview**: Preview operations before execution
- **Template System**: Built-in templates for daily, meeting, research, project notes
- **Variable Substitution**: Customizable template parameters

## Code Conventions

### Go Style Guidelines
- Package comments: Use `// Package name description` format
- Imports: Group standard library, third-party, then local packages with blank lines
- Error handling: Always check errors, use `fmt.Errorf` with `%w` for wrapping
- Logging: Use `log/slog` with structured logging (`slog.Error("message", "key", value)`)
- JSON tags: Use snake_case for JSON field names
- MCP tools: Use descriptive names with underscores (`read_note`, `search_pdfs`)

### Dependencies
- **MCP Framework**: `github.com/mark3labs/mcp-go` - Core MCP server implementation
- **PDF Processing**: `github.com/gen2brain/go-fitz` - MuPDF bindings for PDF manipulation
- **Google APIs**: `google.golang.org/api` - Google Drive integration
- **Configuration**: `github.com/joho/godotenv` - Environment variable loading

## Testing Infrastructure

The project maintains comprehensive test coverage with:
- **11 test packages** across all major components
- **4,852+ lines** of dedicated test code
- **Unit tests** for all core functionality
- **Integration tests** for full workflow validation
- **Performance benchmarks** for scalability testing
- **70% minimum coverage threshold** enforced by CI/CD

## CI/CD Pipeline

GitHub Actions workflow (`.github/workflows/test.yml`) provides:
- Multi-version Go testing (1.21, 1.22)
- Comprehensive linting with golangci-lint
- Security scanning with gosec and nancy
- Coverage reporting with codecov
- Build artifact generation
- Benchmark performance tracking

## Environment Setup

The project uses Go 1.24.4 and requires:
1. Google Cloud credentials for Drive API access
2. Mathpix API credentials for OCR functionality
3. MCP-compatible host for server interaction

Sample MCP configuration is available in `examples/mcp-config.json`.