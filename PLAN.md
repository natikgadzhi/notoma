# notion-sync: Notion → Obsidian Continuous Sync Tool

## Goals

1. One-way sync from Notion to Obsidian (Notion is source of truth)
2. Incremental updates — only sync pages modified since last run
3. Database → Bases conversion — Notion databases to Obsidian `.base` files
4. Attachment handling — download images/files before URLs expire (1-hour TTL)
5. Configurable scope — sync specific pages/databases, not entire workspace
6. Runs as k8s CronJob in your k3s cluster

## Non-Goals

- Two-way sync, real-time sync, conflict resolution, GUI

---

## Progress

- [x] **Phase 1:** Project scaffolding, CLI, config, Notion API connection, rate limiting
- [x] **Phase 2:** Block → Markdown transformation (all block types)
- [x] **Phase 3:** Database → Obsidian Bases conversion
- [ ] **Phase 4:** Attachments & incremental sync state
- [ ] **Phase 5:** CI/CD, Docker, k8s manifests
- [ ] **Phase 6:** `status` command
- [ ] **Phase 7:** `validate` command

---

## Tech Stack

| Component | Choice | Version |
|-----------|--------|---------|
| Language | Go | 1.24.5+ |
| Notion Client | `github.com/jomei/notionapi` | v1.13.3 |
| CLI | `github.com/spf13/cobra` | v1.9.1 |
| YAML | `gopkg.in/yaml.v3` | latest |
| Env loading | `github.com/joho/godotenv` | latest |
| Logging | `log/slog` (stdlib) | — |
| Linter | golangci-lint | v2.8.0 |

---

## Secrets Management

### Development

Use environment variables or `.env` file. Never commit secrets to the repo.

**.gitignore** (must include):
```
.env
*.env
!.env.sample
```

**.env.sample** (commit this as a template):
```bash
# Notion integration token (create at https://www.notion.so/my-integrations)
NOTION_TOKEN=ntn_xxx

# Docker Hub credentials (for pushing images)
DOCKER_HUB_USERNAME=yourusername
DOCKER_HUB_TOKEN=dckr_pat_xxx
```

The tool loads secrets from:
1. Environment variables (highest priority)
2. `.env` file in current directory (for local dev)

Use `github.com/joho/godotenv` to load `.env` files.

### Config File

The config file contains NO secrets. `NOTION_TOKEN` is loaded from environment only.

### CI/CD

- Store `NOTION_TOKEN` as a GitHub repository secret
- Store `DOCKER_HUB_USERNAME` and `DOCKER_HUB_TOKEN` as GitHub secrets
- Never echo or log secrets

### Kubernetes

- Create a k8s Secret for `NOTION_TOKEN`
- Reference via `secretKeyRef` in CronJob, not in ConfigMap

Study these codebases for transformation logic:

### obsidian-importer
- **Repo:** https://github.com/obsidianmd/obsidian-importer
- **Version:** 1.8.2
- **Key files:**
  - `src/formats/notion/convert-to-md.ts` — Block → Markdown
  - `src/formats/notion/notion-databases.ts` — Database → `.base` files
  - `src/formats/notion/notion-api.ts` — API client patterns

### notion2obsidian
- **Repo:** https://github.com/bitbonsai/notion2obsidian
- **Version:** 2.6.0
- **Key files:**
  - `src/lib/links.js` — Wiki-link conversion
  - `src/lib/callouts.js` — Callout icon mapping
  - `src/lib/frontmatter.js` — YAML frontmatter

---

## Project Structure

```
notion-sync/
├── cmd/notion-sync/main.go
├── internal/
│   ├── config/config.go
│   ├── notion/
│   │   ├── client.go
│   │   └── ratelimit.go
│   ├── transform/
│   │   ├── markdown.go
│   │   ├── richtext.go
│   │   ├── base.go
│   │   └── attachments.go
│   ├── sync/
│   │   ├── sync.go
│   │   └── state.go
│   └── writer/writer.go
├── .env.sample
├── .gitignore
├── .golangci.yml
├── Dockerfile
├── .github/workflows/
├── deploy/cronjob.yaml
└── go.mod
```

---

## Agent Instructions

### General Rules

**For every code change:**
1. State the goal clearly
2. Make the change
3. Write tests for the change
4. Run tests: `go test ./...` — all must pass
5. Run formatter: `go fmt ./...`
6. Run linter: `golangci-lint run`
7. Build Docker image: `docker build -t notion-sync .` — must succeed
8. Run in Docker: `docker run --rm notion-sync --help` — must show help
9. Fix any issues before proceeding

**Never:**
- Comment out failing tests
- Skip linter errors
- Proceed with broken tests
- Proceed if Docker build fails
- Commit secrets or .env files

**Concurrency warning:**
- Be extremely cautious with goroutines
- Prefer sequential code unless parallelism is explicitly needed
- If using goroutines: use `sync.WaitGroup`, proper channel closing, context cancellation
- Always test concurrent code with `-race` flag: `go test -race ./...`

### Dockerfile Requirement

A working Dockerfile must exist from Phase 1 onward. Use this minimal version initially:

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o notion-sync ./cmd/notion-sync

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/notion-sync /notion-sync
ENTRYPOINT ["/notion-sync"]
```

### PR Workflow (After CI Setup)

Once GitHub Actions CI is configured and the agent has `gh` authorization:

1. Create a feature branch for each phase: `git checkout -b phase-N-description`
2. Make changes, verify locally (tests, lint, docker build)
3. Commit with clear message: `git commit -m "feat: implement block transformation"`
4. Push and create PR: `gh pr create --fill`
5. Wait for CI checks to pass: `gh pr checks --watch`
6. If CI fails, fix locally, push again
7. Once CI passes, merge: `gh pr merge --auto --squash`

**CI must pass before merging. No exceptions.**

---

## Phase 1: Project Scaffolding & API Connection

**Goal:** Verify we can connect to Notion and fetch pages.

### Agent Prompt
```
Initialize a Go 1.24 module called notion-sync with the project structure from the plan.

Set up:
1. Cobra CLI with `sync` command
2. Create .env.sample with NOTION_TOKEN, DOCKER_HUB_USERNAME, DOCKER_HUB_TOKEN
3. Add .env to .gitignore
4. Use github.com/joho/godotenv to load .env file (fallback, env vars take priority)
5. Load NOTION_TOKEN from environment (not from config file)
6. YAML config loading for sync roots, output paths, options
7. URL parser to extract page/database IDs from Notion share URLs
   - Handle: https://www.notion.so/{workspace}/{title}-{id}
   - Handle: https://www.notion.so/{workspace}/{id}?v={view_id}
   - Extract 32-char hex ID, format as UUID (8-4-4-4-12) if needed
7. Notion client wrapper using github.com/jomei/notionapi
8. Rate limiter (3 req/s average, handle 429 with Retry-After header)
9. Auto-detect resource type (page vs database) by trying API endpoints
10. Create minimal Dockerfile (see Agent Instructions)

Clone and study https://github.com/obsidianmd/obsidian-importer 
(specifically src/formats/notion/notion-api.ts) for rate limiting patterns.

Verification:
- Write tests for URL parsing (various URL formats)
- Write tests for config loading
- Write tests for NOTION_TOKEN loading from env
- Write tests for rate limiter
- Run: go test ./... (must pass)
- Run: go fmt ./...
- Run: golangci-lint run
- Run: docker build -t notion-sync . (must succeed)
- Run: docker run --rm notion-sync --help (must show help)
- Manual test: NOTION_TOKEN=xxx ./notion-sync sync --config config.yaml (fetch a page title)
```

---

## Phase 2: Block → Markdown Transformation

**Goal:** Convert Notion blocks to Obsidian-flavored markdown.

### Agent Prompt
```
Implement block-to-markdown transformation in internal/transform/markdown.go.

IMPORTANT: Before writing any code, clone and thoroughly review the actual 
implementation in obsidian-importer. The mapping table below is a reference,
but the obsidian-importer code contains edge cases and details not captured here.

1. Clone https://github.com/obsidianmd/obsidian-importer
2. Read src/formats/notion/convert-to-md.ts line by line
3. Note how each block type is handled, especially:
   - Nested blocks and recursion
   - Rich text with multiple annotations
   - Table rendering with headers
   - Synced blocks (need to resolve original)
   - Toggle headings (heading + toggleable)
4. Also review https://github.com/bitbonsai/notion2obsidian src/lib/callouts.js
   for callout icon → Obsidian callout type mapping

Implement converters for ALL Notion API block types (see table below).
Handle nested blocks recursively via has_children flag.

Verification:
- Write table-driven tests for each block type
- Write tests for rich text annotation combinations
- Test deeply nested block structures (3+ levels)
- Run: go test ./... (must pass)
- Run: go test -race ./... (if any concurrency)
- Run: go fmt ./...
- Run: golangci-lint run
- Run: docker build -t notion-sync . (must succeed)
- Run: docker run --rm notion-sync --help (must show help)
```

### Notion Block Types (from API)

| Notion Block | Markdown Output | Notes |
|--------------|-----------------|-------|
| `paragraph` | `text\n\n` | May have children |
| `heading_1` | `# Title\n\n` | `is_toggleable` → foldable callout |
| `heading_2` | `## Title\n\n` | `is_toggleable` → foldable callout |
| `heading_3` | `### Title\n\n` | `is_toggleable` → foldable callout |
| `bulleted_list_item` | `- item\n` | Handle nesting with indentation |
| `numbered_list_item` | `1. item\n` | Handle nesting, reset numbering |
| `to_do` | `- [ ] task` / `- [x] task` | `checked` field |
| `toggle` | `> [!faq]- Title\n> content` | Foldable callout |
| `code` | ` ```lang\ncode\n``` ` | `language` field, `caption` |
| `quote` | `> quote\n` | May have children |
| `callout` | `> [!type] Title\n> content` | Map `icon.emoji` to callout type |
| `divider` | `---\n\n` | |
| `table` | Markdown table | `has_column_header`, `has_row_header` |
| `table_row` | `\| cell \| cell \|` | Child of table |
| `column_list` | Flatten contents | Obsidian has no columns |
| `column` | Flatten contents | Child of column_list |
| `image` | `![caption](path)` | Download, update path |
| `video` | `![video](url)` | External URL or embed |
| `file` | `[filename](path)` | Download, update path |
| `pdf` | `![pdf](path)` | Download, update path |
| `bookmark` | `[title](url)` | `url`, `caption` |
| `embed` | Platform-specific | YouTube, Twitter, etc. |
| `equation` | `$$latex$$` | `expression` field |
| `child_page` | `[[Page Name]]` | Wiki-link |
| `child_database` | `[[Database Name]]` | Wiki-link to .base |
| `link_to_page` | `[[Page Name]]` | Resolve page title |
| `synced_block` | Resolve original | If `synced_from` not null, fetch original |
| `template` | Skip | Notion-specific |
| `breadcrumb` | Skip | Not meaningful in Obsidian |
| `table_of_contents` | Skip | Obsidian generates automatically |
| `link_preview` | `[title](url)` | Like bookmark |
| `audio` | `![audio](url)` | External URL |

### Rich Text Annotations

| Annotation | Markdown | Notes |
|------------|----------|-------|
| `bold` | `**text**` | |
| `italic` | `*text*` | |
| `strikethrough` | `~~text~~` | |
| `underline` | `<u>text</u>` | HTML, or ignore |
| `code` | `` `text` `` | |
| `color` | Ignore or `==highlight==` | For `*_background` colors |
| `link.url` | `[text](url)` | Convert internal to wiki-link |
| `mention.page` | `[[Page Name]]` | Resolve page title |
| `mention.user` | `@Name` | Or just name |
| `mention.date` | Date string | Format as ISO |
| `mention.database` | `[[Database]]` | |
| `equation` | `$latex$` | Inline equation |

### Callout Icon Mapping

Map Notion callout icons to Obsidian callout types:
- 💡 → `tip`
- ⚠️ → `warning`  
- ❗ → `danger`
- ℹ️ → `info`
- ✅ → `success`
- ❌ → `failure`
- ❓ → `question`
- 📝 → `note`
- Default → `note`

---

## Phase 3: Database → Bases Conversion

**Goal:** Transform Notion databases into Obsidian `.base` files.

### Agent Prompt
```
Implement database-to-bases conversion in internal/transform/base.go.

Study https://github.com/obsidianmd/obsidian-importer 
(src/formats/notion/notion-databases.ts) for the conversion logic.

Implement:
1. Fetch database schema (properties, types)
2. Generate .base YAML file with:
   - filters (file.inFolder for the database folder)
   - properties mapping
   - default table view
3. Generate entry notes with frontmatter from row properties
4. Handle relations as wiki-links: `related: ["[[Page A]]", "[[Page B]]"]`
5. Store formula/rollup computed values (don't convert formulas)

Property type mapping:
- title, rich_text, number, select, checkbox, url, email → direct
- multi_select → list
- date → ISO string or range
- relation → list of wiki-links
- formula, rollup → computed value only
- files → list of local paths (after download)

Verification:
- Write tests for schema parsing
- Write tests for .base YAML generation
- Write tests for frontmatter generation
- Write tests for relation → wiki-link conversion
- Run: go test ./... (must pass)
- Run: go fmt ./...
- Run: golangci-lint run
- Run: docker build -t notion-sync . (must succeed)
- Run: docker run --rm notion-sync --help (must show help)
```

---

## Phase 4: Attachments & Incremental Sync

**Goal:** Handle files and implement efficient incremental sync.

### Agent Prompt
```
Implement attachment handling and state management.

1. Attachment downloader (internal/transform/attachments.go):
   - Detect Notion-hosted URLs (contain secure.notion-static.com or prod-files-secure)
   - Download immediately when page is fetched (URLs expire in 1 hour!)
   - Generate unique filenames, store in configured attachment folder
   - Update markdown references to local paths

2. State management (internal/sync/state.go):
   - Track last sync time
   - Track page versions (last_edited_time per page ID)
   - Load/save state to JSON file
   - Query API for pages modified since last sync

3. Sync orchestration (internal/sync/sync.go):
   - Load state
   - Fetch changed pages
   - For each page: fetch blocks, transform, download attachments, write
   - Update state

Verification:
- Write tests for URL detection
- Write tests for state persistence
- Write tests for incremental sync logic
- Mock HTTP for attachment download tests
- Run: go test ./... (must pass)
- Run: go test -race ./... (required for sync code)
- Run: go fmt ./...
- Run: golangci-lint run
- Run: docker build -t notion-sync . (must succeed)
- Run: docker run --rm notion-sync --help (must show help)
```

---

## Phase 5: Deployment & CI/CD

**Goal:** Containerize and deploy to k8s.

### Agent Prompt
```
Set up deployment infrastructure.

1. Dockerfile:
   - Multi-stage build (golang:1.24-alpine → scratch)
   - Copy CA certs for HTTPS
   - Build with CGO_ENABLED=0

2. GitHub Actions (.github/workflows/build.yaml):
   - Run tests on PR and push
   - Build and push to GHCR on main
   - Tag with git SHA and semver

3. Homelab integration (.github/workflows/bump-homelab.yaml):
   - Trigger repository_dispatch to homelab repo on successful build
   - Pass image SHA in payload

4. Kubernetes manifests (deploy/):
   - CronJob (configurable schedule, default every 6 hours)
   - ConfigMap for config.yaml
   - Secret for NOTION_TOKEN
   - PVC reference for vault storage

Verification:
- Dockerfile builds locally: docker build -t notion-sync .
- Container runs: docker run notion-sync --help
- GitHub Actions syntax is valid (use actionlint if available)
- k8s manifests are valid YAML
```

---

## Phase 6: Status Command

**Goal:** Implement `notion-sync status` to show sync state.

### Agent Prompt
```
Implement the `status` command in cmd/notion-sync/status.go.

The command should:
1. Load state file from config
2. Display:
   - Last sync time (human-readable, e.g., "2 hours ago")
   - Number of tracked pages
   - Number of tracked databases
   - List of recently synced items (last 10)
   - Any pages that failed in last sync (if tracking errors)
3. Handle missing state file gracefully ("No sync has been performed yet")
4. Support --json flag for machine-readable output

Verification:
- Write tests for state display formatting
- Write tests for missing state file handling
- Write tests for JSON output format
- Run: go test ./... (must pass)
- Run: go fmt ./...
- Run: golangci-lint run
- Run: docker build -t notion-sync . (must succeed)
- Run: docker run --rm notion-sync status --help (must show help)
```

---

## Phase 7: Validate Command

**Goal:** Implement `notion-sync validate` to check config and connectivity.

### Agent Prompt
```
Implement the `validate` command in cmd/notion-sync/validate.go.

The command should:
1. Parse and validate config file syntax
2. Check required fields are present (at least one root, output path)
3. Check NOTION_TOKEN env var is set
4. Validate Notion token by making a test API call (users.me endpoint)
5. For each root URL:
   - Parse URL and extract ID
   - Verify the resource exists and is accessible
   - Report resource type (page or database) and title
5. Check output path exists and is writable
6. Check state file path is writable
7. Report all issues found, don't stop at first error

Output format:
✓ Config syntax valid
✓ Notion token valid (workspace: "My Workspace")
✓ Root 1: Page "Personal Wiki" (abc123...)
✗ Root 2: Database "Reading List" - not shared with integration
✓ Output path /data/vault exists and writable
✗ State path /data/state.json - directory does not exist

Exit code: 0 if all checks pass, 1 if any fail

Verification:
- Write tests for each validation check
- Write tests for error aggregation
- Mock API calls for testing
- Run: go test ./... (must pass)
- Run: go fmt ./...
- Run: golangci-lint run
- Run: docker build -t notion-sync . (must succeed)
- Run: docker run --rm notion-sync validate --help (must show help)
```

---

## CLI Interface

```bash
notion-sync sync --config config.yaml           # Full sync
notion-sync sync --config config.yaml --dry-run # Preview changes
notion-sync sync --config config.yaml --force   # Ignore state, full resync
notion-sync status --config config.yaml         # Show sync state
notion-sync status --config config.yaml --json  # Machine-readable state
notion-sync validate --config config.yaml       # Validate config and connectivity
notion-sync version                             # Show version
```

---

## Config File Structure

**Important:** `NOTION_TOKEN` is loaded from environment variables or `.env` file only — it is NOT in the config file.

```yaml
# config.yaml
sync:
  # Use Notion share URLs (Copy link from Share menu)
  roots:
    - url: "https://www.notion.so/myworkspace/Personal-Wiki-abc123def456..."
      name: "Personal Wiki"  # Optional, for logging
    - url: "https://www.notion.so/myworkspace/def456...?v=xyz789..."
      name: "Reading List"

output:
  vault_path: "/data/obsidian-vault"
  attachment_folder: "_attachments"

state:
  file: "/data/notion-sync-state.json"

options:
  download_attachments: true
```

### URL Parsing

Notion URLs follow these formats:
- **Pages:** `https://www.notion.so/{workspace}/{title}-{id}` or `https://www.notion.so/{workspace}/{id}`
- **Databases:** `https://www.notion.so/{workspace}/{id}?v={view_id}`

The ID is a 32-character hex string (last segment before `?`). The tool should:
1. Extract the 32-char ID from the URL
2. Auto-detect type (page vs database) by querying the API
3. Format as UUID if needed (8-4-4-4-12 pattern)
