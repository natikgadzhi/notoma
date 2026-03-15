# Claude Code Instructions for notoma

## Project Overview

One-way sync tool from Notion to Obsidian. Notion is source of truth. Supports incremental updates, database-to-Bases conversion, and attachment handling.

## Tech Stack

- **Language:** Go 1.24+
- **Notion Client:** `github.com/jomei/notionapi` v1.13.3
- **CLI:** `github.com/spf13/cobra` v1.9.1
- **Config:** `gopkg.in/yaml.v3`
- **Env Loading:** `github.com/joho/godotenv`
- **Logging:** `log/slog` (stdlib)
- **Linter:** golangci-lint v2.8.0

## Secrets

- `NOTION_TOKEN` is loaded from environment or `.env` file only — never from config
- Never commit `.env` files
- `.env.sample` is the template (safe to commit)

## Verification Checklist (Every Change)

Run these commands after every code change:

```bash
go test ./...              # All tests must pass
go test -race ./...        # Required if any concurrency
go fmt ./...               # Format code
golangci-lint run          # No linter errors
docker build -t notoma .   # Must succeed
docker run --rm notoma --help  # Must show help
```

## Rules

**Always:**
- State the goal before making changes
- Write tests for new code
- Run the full verification checklist
- Fix issues before proceeding
- Read existing code before writing new code

**Never:**
- Comment out failing tests
- Skip linter errors
- Proceed with broken tests or Docker build
- Commit secrets or `.env` files
- Use goroutines without explicit need (prefer sequential code)
- Modify this file or the project spec

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/):

```
<type>(<scope>): <description>
```

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`

**Examples:**
- `feat(transform): add callout block support`
- `fix(sync): handle rate limit errors gracefully`
- `docs: update CLI usage examples`

## Concurrency

Be extremely cautious. If goroutines are needed:
- Use `sync.WaitGroup`
- Proper channel closing
- Context cancellation
- Always test with `-race` flag

## Project Structure

```
notoma/
├── cmd/notoma/main.go
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

## Implementation Phases

1. **Phase 1:** Project scaffolding, CLI, config, Notion API connection, rate limiting
2. **Phase 2:** Block → Markdown transformation (all block types)
3. **Phase 3:** Database → Obsidian Bases conversion
4. **Phase 4:** Attachment handling (download images/files, update paths)
5. **Phase 5:** Incremental sync (state tracking, change detection)
6. **Phase 6:** CI/CD, Docker, k8s manifests
7. **Phase 7:** `status` command
8. **Phase 8:** `validate` command

## Reference Implementations

Study these before implementing transformation logic:
- `obsidian-importer`: https://github.com/obsidianmd/obsidian-importer
  - `src/formats/notion/convert-to-md.ts` — Block → Markdown
  - `src/formats/notion/notion-databases.ts` — Database → Bases
- `notion2obsidian`: https://github.com/bitbonsai/notion2obsidian
  - `src/lib/callouts.js` — Callout icon mapping

## CLI Commands

```bash
notoma sync --config config.yaml           # Full sync
notoma sync --config config.yaml --dry-run # Preview changes
notoma sync --config config.yaml --force   # Ignore state, full resync
notoma status --config config.yaml         # Show sync state
notoma validate --config config.yaml       # Validate config and connectivity
notoma version                             # Show version
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--config` / `-c` | Path to config file (default: `config.yaml`) |
| `--debug` | Print verbose debug logs to stderr |

## Multi-Agent Work Environment

This section outlines how lead and worker agents collaborate on this project.

### Core Workflow

The lead agent orchestrates by reading project specs, decomposing work into phases, and spawning worker agents to execute tasks concurrently. Workers operate in isolated git worktrees, implement features, and submit pull requests for review.

### Agent Roles

**Lead Agent:**
- Reads `README.md` and `CLAUDE.md` to understand requirements
- Creates a phased task plan with clear dependencies
- Spawns worker agents with explicit task details
- Monitors progress and assigns new work as agents become available
- Resolves conflicts and coordinates integration efforts

**Worker Agents:**
- Execute tasks in dedicated git worktrees on feature branches
- Read existing code before writing new code
- Run quality checks: `go build`, `go vet`, `go test`
- Commit work and push to create pull requests
- Keep commits atomic with clear messages

**Reviewer Agents:**
- Check out PR branches in separate worktrees
- Verify builds, tests, and acceptance criteria
- Conduct code quality and security reviews
- Post structured feedback via `gh pr review`

### Git & Worktree Requirements

Each task gets its own worktree and branch. Workers must never commit directly to main — all code changes require a pull request. Convention: `git worktree add ../notoma-task-N -b task-N-description`.

### Task Organization

Tasks move between three states reflecting their status:
- `backlog/` — unstarted tasks
- `in-progress/` — active tasks
- `done/` — completed tasks

### Quality Standards

Every implementation requires: build succeeds, linting passes, tests run, and features satisfy acceptance criteria. Reviewers conduct security and code quality checks before approving merges.

### Key Rules

- Never modify the project spec document
- Always read existing code before writing
- Write tests for every task
- Keep commits atomic and focused
- Declare explicit dependencies between tasks
- Verify with end-to-end tests where applicable
