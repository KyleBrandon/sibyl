# Agent Guidelines for Sibyl MCP Project

## Build Commands
- `make all` - Build all binaries (client, note_server, gcp_server)
- `make client` - Build client binary to ./bin/client
- `make note_server` - Build note server to ./bin/note-server  
- `make gcp_server` - Build GCP server to ./bin/gcp-server
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
- MCP tools: Use descriptive names with underscores (e.g., `read_note`, `search_drive_files`)

## Project Structure
- `cmd/` - Main applications (client, note_server, gcp_server)
- `pkg/` - Reusable packages (dto, gcp-mcp, notes, utils)
- `pkg/dto/` - Data transfer objects
- No existing test files found - create `*_test.go` files following Go conventions