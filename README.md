# Loki MCP Server

[![CI](https://github.com/grafana/loki-mcp/workflows/CI/badge.svg)](https://github.com/grafana/loki-mcp/actions/workflows/ci.yml)

A Go-based server implementation for the Model Context Protocol (MCP) with Grafana Loki integration.

## Getting Started

### Prerequisites

- Go 1.21 or higher

### Building and Running

Build and run the server:

```bash
# Build the server
go build -o loki-mcp ./cmd/loki-mcp

# Run the server
./loki-mcp
```

Or run directly with Go:

```bash
go run ./cmd/loki-mcp
```

The server communicates using stdin/stdout and SSE following the Model Context Protocol (MCP). This makes it suitable for use with Claude Desktop and other MCP-compatible clients.

## Project Structure

```
.
├── cmd/
│   └── loki-mcp/         # MCP server implementation
├── internal/
│   └── handlers/         # Tool handlers
├── docker/               # Docker-related files (promtail, grafana, scripts)
└── go.mod                # Go module definition
```

## MCP Server

The Loki MCP Server implements the Model Context Protocol (MCP) and provides the following tools:

### Loki Query Tool

The `loki_query` tool allows you to query Grafana Loki log data:

- Required parameters:
  - `query`: LogQL query string

- Optional parameters:
  - `url`: The Loki server URL (default: from LOKI_URL environment variable or http://localhost:3100)
  - `start`: Start time for the query (default: 1h ago)
  - `end`: End time for the query (default: now)
  - `limit`: Maximum number of entries to return (default: 100)
  - `org`: Organization ID for the query (sent as X-Scope-OrgID header)

### Loki Label Names Tool

The `loki_label_names` tool returns all available label names from Loki.

### Loki Label Values Tool

The `loki_label_values` tool returns all values for a specific label.

#### Environment Variables

The Loki tools support the following environment variables:

- `LOKI_URL`: Default Loki server URL to use if not specified in the request
- `LOKI_ORG_ID`: Default organization ID to use if not specified in the request
- `LOKI_USERNAME`: Default username for basic authentication if not specified in the request
- `LOKI_PASSWORD`: Default password for basic authentication if not specified in the request
- `LOKI_TOKEN`: Default bearer token for authentication if not specified in the request

**Security Note**: When using authentication environment variables, be careful not to expose sensitive credentials in logs or configuration files. Consider using token-based authentication over username/password when possible.

## Docker Support

You can build and run the MCP server using Docker:

```bash
# Build the Docker image
docker build -t loki-mcp .

# Run the server
docker run --rm -i loki-mcp
```

Alternatively, you can use Docker Compose:

```bash
# Build and run with Docker Compose
docker-compose up --build
```

### Local Testing with Loki

The project includes a complete Docker Compose setup to test Loki queries locally:

1. Start the Docker Compose environment:
   ```bash
   docker-compose up -d
   ```

   This will start:
   - A Loki server on port 3100
   - A Grafana instance on port 3000 (pre-configured with Loki as a data source)
   - A log generator container that sends sample logs to Loki
   - The Loki MCP server

2. Insert dummy logs for testing:
   ```bash
   # Insert 10 dummy logs with default settings
   ./docker/insert-loki-logs.sh

   # Insert 20 logs with custom job and app name
   ./docker/insert-loki-logs.sh --num 20 --job "custom-job" --app "my-app"

   # Insert logs with custom environment and interval
   ./docker/insert-loki-logs.sh --env "production" --interval 0.5

   # Show help message
   ./docker/insert-loki-logs.sh --help
   ```

3. Access the Grafana UI at http://localhost:3000 to explore logs visually.

## Server-Sent Events (SSE) Support

The server now supports multiple modes of communication:
1. Standard input/output (stdin/stdout) following the Model Context Protocol (MCP)
2. HTTP Server with Server-Sent Events (SSE) endpoint for integration with tools like n8n
3. Streamable HTTP endpoint for modern MCP clients

The default port for the HTTP server is 8080, but can be configured using the `PORT` environment variable.

### Server Endpoints

When running in HTTP mode, the server exposes the following endpoints:

- SSE Endpoint: `http://localhost:8080/sse` - For real-time event streaming (legacy)
- MCP Endpoint: `http://localhost:8080/mcp` - For MCP protocol messaging (legacy)
- Streamable HTTP: `http://localhost:8080/stream` - For modern MCP clients

### Using Docker with SSE

When running the server with Docker, make sure to expose port 8080:

```bash
# Build the Docker image
docker build -t loki-mcp .

# Run the server with port mapping
docker run -p 8080:8080 --rm -i loki-mcp
```

### n8n Integration

You can integrate the Loki MCP Server with n8n workflows:

1. Install the MCP Client Tools node in n8n

2. Configure the node with these parameters:
   - **SSE Endpoint**: `http://your-server-address:8080/sse` (replace with your actual server address)
   - **Authentication**: Choose appropriate authentication if needed
   - **Tools to Include**: Choose which Loki tools to expose to the AI Agent

3. Connect the MCP Client Tool node to an AI Agent node that will use the Loki querying capabilities

Example workflow:
Trigger → MCP Client Tool (Loki server) → AI Agent (Claude)

## Architecture

The Loki MCP Server uses a modular architecture:

- **Server**: The main MCP server implementation in `cmd/loki-mcp/main.go`
- **Handlers**: Individual tool handlers in `internal/handlers/`
  - `loki.go`: Grafana Loki query functionality

## Using with Claude Desktop

You can use this MCP server with Claude Desktop to add Loki query tools. Follow these steps:

### Option 1: Using the Compiled Binary

1. Build the server:
```bash
go build -o loki-mcp ./cmd/loki-mcp
```

2. Add the configuration to your Claude Desktop configuration file using `claude_desktop_config_binary.json`.

### Option 2: Using Docker (Recommended)

1. Build the Docker image:
```bash
docker build -t loki-mcp .
```

2. Add the configuration to your Claude Desktop configuration file using `claude_desktop_config_docker.json`.

### Configuration Details

The Claude Desktop configuration file is located at:
- On macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- On Windows: `%APPDATA%\Claude\claude_desktop_config.json`
- On Linux: `~/.config/Claude/claude_desktop_config.json`

You can use one of the example configurations provided in this repository:
- `claude_desktop_config.json`: Generic template using `go run`
- `claude_desktop_config_binary.json`: Example using the compiled binary
- `claude_desktop_config_docker.json`: Example using Docker (most reliable)

**Notes**:
- When using `go run` with Claude Desktop, you may need to set several environment variables in the configuration file:
  - `HOME`: The user's home directory
  - `GOPATH`: The Go workspace directory
  - `GOMODCACHE`: The Go module cache directory
  - `GOCACHE`: The Go build cache directory

  These are required to ensure Go can find its modules and build cache when run from Claude Desktop.

- Using Docker is the most reliable approach as it packages all dependencies and environment variables in a container.

Or create your own configuration:

```json
{
  "mcpServers": {
    "lokiserver": {
      "command": "path/to/loki-mcp",
      "args": [],
      "env": {
        "LOKI_URL": "http://localhost:3100",
        "LOKI_ORG_ID": "your-default-org-id",
        "LOKI_USERNAME": "your-username",
        "LOKI_PASSWORD": "your-password",
        "LOKI_TOKEN": "your-bearer-token"
      },
      "disabled": false,
      "autoApprove": ["loki_query"]
    }
  }
}
```

Make sure to replace `path/to/loki-mcp` with the absolute path to the built binary.

4. Restart Claude Desktop.

5. You can now use the tools in Claude:
   - Loki query examples:
     - "Query Loki for logs with the query {job=\"varlogs\"}"
     - "Find error logs from the last hour in Loki using query {job=\"varlogs\"} |= \"ERROR\""
     - "Show me the most recent 50 logs from Loki with job=varlogs"
     - "Query Loki for logs with org 'tenant-123' using query {job=\"varlogs\"}"

### Using Organization ID in Natural Language Prompts

When using this MCP server with Claude Desktop or other AI assistants, users can naturally mention the organization ID in their prompts in several ways:

#### Direct Organization Reference
- **"Query Loki for logs from organization 'tenant-123' with the query {job=\"varlogs\"}"**
- **"Search Loki logs for org 'production-env' using {job=\"web\"}"**
- **"Get logs from organization ID 'client-abc' matching {service=\"api\"}"**

#### Contextual Organization Mentions
- **"Check the error logs for our production tenant (org: prod-001) using query {level=\"error\"}"**
- **"Find all logs from customer organization 'customer-xyz' for the last hour"**
- **"Query Loki with org tenant-456 to find logs matching {job=\"backend\"}"**

#### Multi-tenant Scenarios
- **"Switch to organization 'dev-team' and query {job=\"logs\"} for debugging"**
- **"Use org 'staging-env' to search for warning logs in the last 2 hours"**
- **"Search logs in tenant 'qa-environment' for any error messages"**

#### Combined with Other Parameters
- **"Query Loki for organization 'prod-cluster' from 2 hours ago to now with limit 50"**
- **"Get the last 100 logs from org 'microservice-team' for query {app=\"payment\"}"**

When you mention any of these natural language prompts, the AI assistant will automatically map terms like "organization", "org", "tenant", or "organization ID" to the `org` parameter in the Loki query tool, which gets sent as the `X-Scope-OrgID` header to your Loki server for proper multi-tenant filtering.

The key is to naturally mention any specific parameters in your request - the AI will understand how to map them to the appropriate Loki query tool parameters. When parameters are not explicitly mentioned, the system will automatically use defaults from environment variables:
- `LOKI_URL` for the Loki server URL
- `LOKI_ORG_ID` for the organization ID
- `LOKI_USERNAME` and `LOKI_PASSWORD` for basic authentication
- `LOKI_TOKEN` for bearer token authentication

This makes it very convenient to set up default connection parameters once and then use natural language queries without having to specify authentication details every time.

## Using with Cursor

You can also integrate the Loki MCP server with the Cursor editor. To do this, add the following configuration to your Cursor settings:

Docker configuration:

```json
{
  "mcpServers": {
    "loki-mcp": {
      "command": "docker",
      "args": ["run", "--rm", "-i",
               "-e", "LOKI_URL=http://host.docker.internal:3100",
               "-e", "LOKI_ORG_ID=your-default-org-id",
               "-e", "LOKI_USERNAME=your-username",
               "-e", "LOKI_PASSWORD=your-password",
               "-e", "LOKI_TOKEN=your-bearer-token",
               "loki-mcp:latest"]
    }
  }
}
```

After adding this configuration, restart Cursor, and you'll be able to use the Loki query tool directly within the editor.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Running Tests

The project includes comprehensive unit tests and CI/CD workflows to ensure reliability:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Run tests with race detection
go test -race ./...
```
