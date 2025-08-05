# Agent Guidelines for Sibyl MCP Project

## Build Commands

- `make all` - Build all MCP servers (pdf_server, notes_server)
- `make pdf_server` - Build PDF server to ./bin/pdf-server
- `make notes_server` - Build notes server to ./bin/notes-server
- `make clean` - Remove all binaries from ./bin/
- `go test ./...` - Run all tests (11 test packages, 4,852+ lines of test code)
- `go test ./pkg/notes` - Run tests for specific package
- `go test -v -run TestMergeStrategies ./pkg/notes` - Run single test function
- `go test ./tests/integration/...` - Run integration tests
- `go test ./cmd/...` - Run CLI tests

## Code Style Guidelines

- Package comments: Use `// Package name description` format
- Imports: Group standard library, third-party, then local packages with blank lines
- Types: Use PascalCase for exported types, camelCase for unexported
- Functions: Use PascalCase for exported functions, camelCase for unexported
- Variables: Use camelCase, descriptive names (e.g., `driveService`, `vaultDir`)
- Error handling: Always check errors, use `fmt.Errorf` with `%w` for wrapping
- Logging: Use `log/slog` with structured logging (e.g., `slog.Error("message", "key", value)`)
- JSON tags: Use snake_case for JSON field names
- MCP tools: Use descriptive names with underscores (e.g., `read_note`, `search_pdfs`)

## Project Structure

- `cmd/` - Main applications (pdf-server, note_server)
- `pkg/` - Reusable packages (dto, pdf-mcp, notes, utils)
- `pkg/dto/` - Data transfer objects
- `pkg/pdf-mcp/` - PDF processing MCP server implementation with OCR support
- `pkg/notes/` - Notes management MCP server implementation
- `tests/` - Integration and performance tests
- `tests/testutils/` - Test utilities and helpers
- Create `*_test.go` files following Go conventions for testing

## Test Coverage

- **11 test packages** with comprehensive coverage
- **4,852+ lines** of test code
- **Unit tests** for all core functionality
- **Integration tests** for full workflow testing
- **Performance tests** for benchmarking
- **CLI tests** for command-line argument parsing
- **GitHub Actions CI/CD** pipeline for automated testing

## OCR Integration

The PDF server now includes comprehensive OCR support for hybrid text extraction:

### OCR Engines

- **Tesseract** - Local OCR engine, good for typed documents
- **Google Vision** - Cloud OCR engine, excellent for handwritten content
- **Mock OCR** - Testing and fallback engine

### Key Features

- **Hybrid Processing** - Combines OCR text extraction with LLM vision analysis
- **Smart Engine Selection** - Automatically chooses best OCR engine based on document type
- **Document Analysis** - Analyzes PDFs to recommend optimal processing approach
- **Multi-language Support** - Configurable language support for OCR engines
- **Confidence Scoring** - OCR results include confidence metrics for quality assessment

### New MCP Tools

- `extract_text_from_pdf` - Pure OCR text extraction
- `extract_structured_text` - OCR with layout and structure information
- `convert_pdf_hybrid` - **Primary tool** - Combines OCR + LLM vision for best results
- `analyze_document` - Document analysis and OCR recommendations
- `list_ocr_engines` - List available OCR engines and capabilities
