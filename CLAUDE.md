# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Build the binary
make build                    # or: go build -o loki-mcp .

# Run all tests
make test                     # or: go test -v ./...

# Run a single test
go test -v ./internal/handlers -run TestFormatLokiResults_TimestampParsing

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

# Format code
make fmt                      # or: gofmt -w .

# Lint
make lint                     # or: golangci-lint run ./...

# Tidy dependencies
make tidy                     # or: go mod tidy
```

## Local Development with Docker

```bash
# Start Loki + Grafana + MCP server environment
docker-compose up -d

# Insert test logs
./docker/insert-loki-logs.sh --num 20 --job "test-job"

# View Grafana at http://localhost:3000
# Loki API at http://localhost:3100
```

## Architecture

This is an MCP (Model Context Protocol) server that provides Grafana Loki query tools to AI assistants like Claude Desktop, Cursor, and n8n.

### Server Entry Point
`main.go` - Creates an MCP server using `github.com/mark3labs/mcp-go` that:
- Serves via stdio (for Claude Desktop integration)
- Serves via HTTP with SSE endpoints (`/sse`, `/mcp`) for legacy clients
- Serves via Streamable HTTP (`/stream`) for modern clients

### Tool Handlers
`internal/handlers/loki.go` - All Loki API integration:
- **loki_query** - Run LogQL queries against Loki
- **loki_label_names** - Get all available label names
- **loki_label_values** - Get values for a specific label

Each tool supports authentication (basic auth or bearer token) and multi-tenancy via `X-Scope-OrgID` header.

### Environment Variables
- `LOKI_URL` - Loki server URL (default: http://localhost:3100)
- `LOKI_ORG_ID` - Default org ID for multi-tenant setups
- `LOKI_USERNAME` / `LOKI_PASSWORD` - Basic auth credentials
- `LOKI_TOKEN` - Bearer token authentication
- `LOKI_QUERY_FILTER` - LogQL stream selector to restrict all queries (e.g., `{namespace="prod"}`)
- `LOKI_SECRET_FILTER` - Enable returned log secret filtering (default: true; set false to opt out)
- `LOKI_OTEL_STDERR` - Enable OpenTelemetry console exporters on stderr
- Standard `OTEL_*` env vars configure OTLP traces, metrics, and logs
- `PORT` - HTTP server port (default: 8080)

### CLI Flags
- `--query-filter` - LogQL stream selector to restrict all queries (overrides `LOKI_QUERY_FILTER`)
- `--secret-filter=false` - Opt out of default secret filtering
- `--otel-stderr` - Emit OpenTelemetry traces, metrics, and logs to stderr for local debugging

### Query Filter
The query filter restricts all Loki queries to logs matching the specified LogQL stream selector. This is useful for multi-tenant environments where clients should only access logs from specific namespaces or jobs.

Example:
```bash
# Restrict all queries to logs with namespace="prod"
./loki-mcp --query-filter='{namespace="prod"}'

# Or via environment variable
LOKI_QUERY_FILTER='{namespace="prod"}' ./loki-mcp
```

When a filter is active, it is ANDed with client queries. For example, if the filter is `{namespace="prod"}` and a client queries `{job="api"}`, the actual query becomes `{namespace="prod", job="api"}`.

### Secret Filter

Returned log lines are scanned by betterleaks using its default ruleset. Matching lines are omitted entirely, and the output includes a note such as `1 line omitted due to detected secrets`. The filter is enabled by default and can be disabled with `--secret-filter=false` or `LOKI_SECRET_FILTER=false`.

### OpenTelemetry

OpenTelemetry is configured with standard `OTEL_*` variables for OTLP exporters. `--otel-stderr` / `LOKI_OTEL_STDERR=true` enables console exporters that write to stderr, never stdout, so stdio transport remains protocol-safe. Logrus output continues to stderr and is bridged into OTel logs when providers are configured.

### Output Formats
The `format` parameter controls output: `raw` (default), `json`, or `text`.

## Module Path

The module is `github.com/kaiko-ai/loki-mcp`. Binary installs as `loki-mcp`:
```bash
go install github.com/kaiko-ai/loki-mcp@latest
```
