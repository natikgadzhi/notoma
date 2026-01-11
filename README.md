# notoma

One-way sync tool from Notion to Obsidian. Notion is the source of truth.

## Features

- Sync Notion pages and databases to Obsidian-flavored markdown
- Incremental updates — only sync pages modified since last run
- Database → Obsidian Bases (`.base` files) conversion
- Attachment handling with automatic download
- Rate limiting to respect Notion API limits
- Runs as CLI or Docker container

## Prerequisites

- Go 1.24+ (for local development)
- Docker (optional, for containerized builds)
- A Notion integration token ([create one here](https://www.notion.so/my-integrations))

## Building Locally

```bash
# Clone the repository
git clone https://github.com/natikgadzhi/notion-based.git
cd notion-based

# Build using Make (outputs to build/)
make build

# Verify it works
./build/notoma --help
```

Or build manually:

```bash
go mod download
go build -o build/notoma ./cmd/notoma
```

## Building with Docker

```bash
make docker

# Or manually:
docker build -t notoma .

# Run it
docker run --rm notoma --help
```

## Make Targets

```bash
make build   # Build binary to build/
make test    # Run tests
make clean   # Remove build artifacts
```

## Configuration

### 1. Set up your Notion token

Create a `.env` file (or set environment variables):

```bash
cp .env.sample .env
# Edit .env and add your NOTION_TOKEN
```

### 2. Create a config file

Create `config.yaml`:

```yaml
sync:
  roots:
    - url: "https://www.notion.so/myworkspace/Personal-Wiki-abc123..."
      name: "Personal Wiki"
    - url: "https://www.notion.so/myworkspace/def456...?v=xyz789..."
      name: "Reading List"

output:
  vault_path: "/path/to/obsidian-vault"
  attachment_folder: "_attachments"

state:
  file: "/path/to/notoma-state.json"

options:
  download_attachments: true
```

### 3. Share pages with your integration

In Notion, share each page/database with your integration:
1. Open the page in Notion
2. Click "Share" → "Invite"
3. Select your integration

## Usage

```bash
# Full sync
./notoma sync --config config.yaml

# Preview changes without writing files
./notoma sync --config config.yaml --dry-run

# Force full resync (ignore state)
./notoma sync --config config.yaml --force

# Show version
./notoma version
```

### With Docker

```bash
docker run --rm \
  -e NOTION_TOKEN="your-token" \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  -v /path/to/vault:/vault \
  notoma sync --config /config.yaml
```

## Development

```bash
# Run tests
go test ./...

# Run tests with race detection
go test -race ./...

# Format code
go fmt ./...

# Run linter (requires golangci-lint)
golangci-lint run
```

## Project Status

This project is under active development. Current status:

- [x] Phase 1: Project scaffolding, CLI, config, Notion API connection
- [x] Phase 2: Block → Markdown transformation
- [x] Phase 3: Database → Obsidian Bases conversion
- [ ] Phase 4: Attachment handling
- [ ] Phase 5: Incremental sync
- [ ] Phase 6: CI/CD, Docker, k8s manifests
- [ ] Phase 7: `status` command
- [ ] Phase 8: `validate` command

## License

MIT
