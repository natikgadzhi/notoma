# notoma

One-way sync tool from Notion to Obsidian via Notion API. Notoma is designed for regular ongoing incremental updates. If you need a one-time update, Obsidian-native Notion importer will work faster.

## Features

- Sync Notion pages and databases to Obsidian-flavored markdown
- Incremental updates — only sync pages modified since last run
- Database → Obsidian Bases (`.base` files) conversion
- Attachment handling with automatic download
- Rate limiting to respect Notion API limits

## Prerequisites

- Go 1.24+ (for local development)
- A Notion integration token ([create one here](https://www.notion.so/my-integrations))

## Building Locally

```bash
# Clone the repository
git clone https://github.com/natikgadzhi/notoma.git
cd notoma

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

## License

MIT
