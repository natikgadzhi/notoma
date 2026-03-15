# notoma

One-way sync tool from Notion to Obsidian via Notion API. Notoma is designed for regular ongoing incremental updates. If you need a one-time migration, the Obsidian-native Notion importer will work faster.

## Features

- Sync Notion pages and databases to Obsidian-flavored markdown
- Incremental updates — only sync pages modified since last run
- Database → Obsidian Bases (`.base` files) conversion
- Attachment handling with automatic download
- Rate limiting to respect Notion API limits

## Prerequisites

- Go 1.24+ (for local development)
- A Notion integration token ([create one here](https://www.notion.so/my-integrations))

## Install

**Homebrew:**

```sh
brew install natikgadzhi/homebrew-taps/notoma
```

**From source:**

```sh
go install github.com/natikgadzhi/notoma/cmd/notoma@latest
```

**Or build from a local checkout:**

```sh
go build -o build/notoma ./cmd/notoma
```

## Setup

### 1. Set up your Notion token

```bash
cp .env.sample .env
# Edit .env and add your NOTION_TOKEN
```

Or set it as an environment variable:

```bash
export NOTION_TOKEN="secret_..."
```

### 2. Create a config file

```bash
cp config.yaml.sample config.yaml
# Edit config.yaml with your Notion roots and vault path
```

Example `config.yaml`:

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

---

## Global flags

```
notoma [--config config.yaml] [--debug] <command>
```

| Flag | Description |
|------|-------------|
| `--config` / `-c` | Path to config file (default: `config.yaml`) |
| `--debug` | Print verbose debug logs to stderr |

---

## Commands

### `sync`

```sh
notoma sync [--dry-run] [--force] [--quiet]
```

Fetches pages and databases from Notion and converts them to Obsidian-flavored markdown files in your vault. By default, performs an incremental sync — only fetching pages modified since the last sync.

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` / `-n` | false | Preview changes without writing files |
| `--force` / `-f` | false | Ignore state and perform full resync |
| `--quiet` / `-q` | false | Disable TUI, use plain log output |

When running in a terminal, a TUI progress display is shown by default. Use `--quiet` to disable the TUI and show plain log output instead.

---

### `status`

```sh
notoma status
```

Displays information about the current sync state:
- When the last sync occurred
- Number of tracked resources (pages and databases)
- Number of database entries

---

### `validate`

```sh
notoma validate
```

Checks that the configuration file is valid and that notoma can connect to Notion and access the configured resources. Performs the following checks:

1. Config file exists and is valid YAML
2. All required config fields are present
3. `NOTION_TOKEN` environment variable is set
4. Notion API is accessible (validates token)
5. All configured roots are accessible
6. Workspace discovery works (if enabled)
7. Output vault path exists and is writable
8. State file directory exists or can be created

---

### `version`

```sh
notoma version
```

Prints version, commit, and build date information.

---

## With Docker

```bash
docker build -t notoma .

docker run --rm \
  -e NOTION_TOKEN="secret_..." \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  -v /path/to/vault:/vault \
  notoma sync --config /config.yaml
```

---

## Development

```bash
# Build binary to build/
make build

# Run tests
go test ./...

# Run tests with race detection
go test -race ./...

# Format code
go fmt ./...

# Run linter (requires golangci-lint)
golangci-lint run
```

## Make Targets

```bash
make build   # Build binary to build/
make test    # Run tests
make clean   # Remove build artifacts
make docker  # Build Docker image
```

## License

MIT
