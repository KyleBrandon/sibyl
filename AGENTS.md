# Agent Guidelines for Sibyl MCP Project

## Build Commands
- `make all` - Build all MCP servers (pdf_server, notes_server)
- `make pdf_server` - Build PDF server to ./bin/pdf-server  
- `make notes_server` - Build notes server to ./bin/notes-server
- `make clean` - Remove all binaries from ./bin/
- `go test ./...` - Run all tests
- `go test ./pkg/notes` - Run tests for specific package

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
- `pkg/pdf-mcp/` - PDF processing MCP server implementation
- `pkg/notes/` - Notes management MCP server implementation
- Create `*_test.go` files following Go conventions for testing