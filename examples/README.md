# Sibyl MCP Configuration Examples

This directory contains configuration files and technical references for integrating Sibyl MCP servers with various LLM hosts.

## Configuration Files

### [`mcp-config.json`](mcp-config.json)
Complete MCP server configuration with all required parameters for:
- **Claude Desktop** - macOS/Windows configuration
- **VS Code MCP Extension** - Development environment setup  
- **Custom MCP Hosts** - Generic configuration template

## MCP Host-Specific Setup

### Claude Desktop

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`  
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```bash
# Copy and customize the configuration
cp examples/mcp-config.json ~/Library/Application\ Support/Claude/claude_desktop_config.json
```

### VS Code MCP Extension

1. Install the MCP extension from the marketplace
2. Add server configurations to VS Code settings:
   ```json
   "mcp.servers": {
     // Use content from mcp-config.json
   }
   ```

### Custom Hosts

For implementing custom MCP hosts, see:
- [MCP Protocol Specification](https://spec.modelcontextprotocol.io/)
- [MCP Go SDK Documentation](https://github.com/mark3labs/mcp-go)

## Environment Variables

All configuration parameters can be set via environment variables instead of command-line arguments:

```bash
# PDF Server
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/credentials.json"
export GCP_FOLDER_ID="your-folder-id"
export MATHPIX_APP_ID="your-app-id" 
export MATHPIX_APP_KEY="your-app-key"

# Notes Server  
export NOTE_SERVER_FOLDER="/path/to/notes"
```

## Testing Configuration

Verify your setup:

```bash
# Test PDF server
echo '{"method": "tools/list"}' | ./bin/pdf-server

# Test Notes server
echo '{"method": "tools/list"}' | ./bin/notes-server
```

## Advanced Configuration

### Custom Log Levels
```bash
--log-level DEBUG    # Detailed debugging
--log-level INFO     # Standard operation (default)
--log-level WARN     # Warnings only
--log-level ERROR    # Errors only
```

### Multi-Language OCR
```bash
--ocr-languages "en,fr,de,es,zh,ja,ko"  # Multiple languages
--ocr-languages "auto"                   # Auto-detect (if supported)
```

### Custom Notes Organization
```bash
--notesFolder "/path/to/vault"           # Obsidian vault
--notesFolder "/path/to/notes"           # Generic markdown folder
--notesFolder "/path/to/docs"            # Documentation folder
```

## Technical Resources

See [`RESOURCES.md`](RESOURCES.md) for detailed information about:
- MCP Resource schemas and endpoints
- JSON response formats
- Resource URI patterns
- Advanced MCP client integration