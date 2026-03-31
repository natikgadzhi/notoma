@README.md

# Multi-agent Work Environment

## How It Works

1. The user edits tasks and discusses the plan with the lead agent
2. The lead agent decomposes work into a phased task plan
3. The lead agent spawns worker agents via the `Agent` tool to execute tasks in parallel
4. Builder workers build code, write tests, and commit their work, all in independent worktrees
5. Reviewer agents review, request changes, and keep a consistent, high quality bar
6. The lead agent coordinates work, keeps `main` up to date, moves tasks between states, merges pull requests

## Lead Agent Behavior

The user will first discuss the high level plan with you. Once the user confirms we're good to start work:

1. **Read** `README.md` and other top-level docs to understand the project
2. **Plan** — Break the work into phases and tasks:
   - **Phase 0: Bootstrap** — Project scaffolding, directory structure, config files, dependency setup
   - **Phase 1: Core** — Core data models, types, interfaces
   - **Phase 2: Implementation** — Feature implementation (parallelize heavily here)
   - **Phase 3: Integration** — Wire components together, integration tests
   - **Phase 4: Polish** — Error handling, edge cases, documentation, final tests
3. **Create tasks** using `TaskCreate` for each work item, with:
   - Clear subject (imperative: "Implement user authentication endpoint")
   - Detailed description with acceptance criteria, file paths, and dependencies
   - `activeForm` in present continuous ("Implementing user authentication")
   - Dependency chains via `addBlockedBy` (Phase 1 tasks block on Phase 0, etc.)
   - List of edge cases that are covered, and edge cases that are not important
4. **Spawn workers** using the `Agent` tool with `subagent_type: "general-purpose"`
   - Spawn 3-6 workers depending on project complexity
   - Use the best model available to you with the maximum effort
   - Pass the task ID and full task details explicitly in each worker's `prompt`
   - Name them descriptively: `builder-1`, `builder-2`, `reviewer-1`, `reviewer-2`, etc.
   - Generally prefer reviewers to have larger context and bigger models
   - Keep the main checkout on `main` branch and `git pull --ff` after each task is completed and merged
5. **Assign tasks** to idle workers as they become available
6. **Track task file state** — move task files between directories to reflect their status:
   - When a worker claims a task: `mv tasks/backlog/<task>.md tasks/in-progress/`
   - When a task's PR is merged and verified: `mv tasks/in-progress/<task>.md tasks/done/`
7. **Monitor progress** — poll `TaskList`/`TaskGet` to track worker progress; spawn the next worker when one completes
8. **Assign reviews of pull requests**. Once a worker prepares a task implementation in a pull request, assign a review task to a reviewer agent
   - Reviewers post their feedback on GitHub pull request comments
   - Builder addresses all feedback
   - You ask for a review again
   - After successful and clean review, you make a decision to merge
9. **Handle conflicts** — if workers produce conflicting changes, ask builders to review and resolve conflicts in their respective pull requests
10. **Shut down** when all tasks are complete

## Worker Agent Instructions

Workers are spawned by the lead via the `Agent` tool. The lead passes the task ID and description in the prompt. Each worker MUST:

1. **Read the task** with `TaskGet <task-id>` to get full requirements
2. **Mark it in-progress** with `TaskUpdate`
3. **Create a git worktree** — ALWAYS use a worktree, never work in the main checkout:

   ```bash
   git fetch origin && git pull --ff   # in main checkout first
   git worktree add ../worktrees/notoma-task-N -b task-N-description
   cd ../worktrees/notoma-task-N
   ```

   - Each task gets its own worktree and its own branch
   - Branch off the latest `main`
   - Work exclusively inside the worktree directory

4. **Read existing code** before writing — understand the current state
5. **Implement the task** — write code, tests, configs as needed
6. **Verify the work** — run the project's quality checks:
   - `go build ./...` — builds cleanly
   - `go vet ./...` — no vet errors
   - `golangci-lint run` — no lint errors
   - `go test ./...` — all tests pass
   - `go test -race ./...` — if any concurrency is involved
7. **Commit the work** with a descriptive message (include task ID):
   - Format: `[task-N] <description>`
   - One logical change per commit
8. **Push and create a pull request** — MANDATORY for every task with code changes:

   ```bash
   git push -u origin task-N-description
   gh pr create --title "type(scope): [task-N] description" --body "..."
   ```

   - Every code change goes through a PR. No direct commits to `main`.

9. **Update the task** via `TaskUpdate` with PR URL and status
10. **Respond to review comments**, push fixes, then merge the PR
11. **Wait for the Leader to confirm successful Merge**
12. **Clean up the worktree** after merge:
    ```bash
    cd /Users/natik.gadzhi/src/natikgadzhi/notoma
    git worktree remove ../worktrees/notoma-task-N
    ```
13. **Mark the task completed** with `TaskUpdate` (status: completed)

## Reviewer Agent Instructions

Reviewers are spawned by the Lead agent with the PR number and task ID in their prompt.

1. **Check out the PR branch** in a worktree:
   ```bash
   git worktree add ../worktrees/notoma-review-N origin/task-N-description
   cd ../worktrees/notoma-review-N
   ```
2. **Check for conflicts** — ensure the branch rebases cleanly on latest `main`. If conflicts exist, flag them and request the worker to rebase.
3. **Run all quality checks**:
   - `go build ./...` — builds cleanly
   - `go vet ./...` — no vet errors
   - `golangci-lint run` — no lint errors
   - `go test ./...` — all tests pass (including new tests for the feature)
4. **Verify the feature works** — read the task description and acceptance criteria, confirm the implementation satisfies each criterion
5. **Code quality review**:
   - Read every changed file in the diff (`gh pr diff`)
   - Check for dead code, unnecessary complexity, or missing error handling
   - Verify naming conventions and code style match the rest of the codebase
   - Ensure no commented-out code was left behind
6. **Security review**:
   - Ensure secrets and tokens are not logged or exposed
   - Check that user input is validated at API/CLI boundaries
   - Flag any command injection risks
7. **Post a PR review** via `gh pr review` with:
   - Summary of what was reviewed
   - Edge cases that ARE covered
   - Edge cases NOT covered (note whether they should be addressed now or later)
   - Any security concerns
   - Approve, request changes, or comment accordingly
8. **Clean up the worktree** after review:
   ```bash
   cd /Users/natik.gadzhi/src/natikgadzhi/notoma
   git worktree remove ../worktrees/notoma-review-N
   ```
9. **Update the task** via `TaskUpdate` with the review outcome

## Git Conventions

- The main checkout stays on `main`. Never switch branches here — workers use worktrees.
- Do `git pull --ff` after every task is completed and merged.
- Each task branch is based off the latest `main`.
- Every code change goes through a PR — no direct commits to `main`.
- Workers commit with messages: `[task-N] <description>`
- One logical change per commit — don't bundle unrelated work
- Workers should `git pull --rebase` before pushing to avoid conflicts

## Task File System

Tasks are stored as markdown files in `tasks/` with three subdirectories representing status:

```
tasks/
├── backlog/      # Not yet started
├── in-progress/  # Currently being worked on
└── done/         # Completed and merged
```

- Each task is a numbered markdown file (e.g. `01-bootstrap-go-project.md`)
- The lead agent creates task files in `backlog/` during planning
- Workers move their task file to `in-progress/` when they start work
- Workers move their task file to `done/` after the PR is merged and the task is complete
- Task files contain the full specification: objective, acceptance criteria, dependencies, and notes
- Moving task files between directories should be committed as part of the worker's branch

## Important Rules

- **Always fetch before checking task status** — run `git fetch` and `git pull --ff` before answering questions about whether a task is done, what's been merged, or what state the codebase is in
- **Always read before writing** — understand existing code before changing it
- **Test everything** — write tests for every task. If ambiguous what kind, ask the user
- **Small, focused tasks** — each task should be completable in one agent session
- **Explicit dependencies** — if task B needs task A's output, declare it with `addBlockedBy`
- **No premature abstraction** — build what's needed, not what might be needed
- **Commit early and often** — small atomic commits, not monolithic ones
- **Always verify with end-to-end tests** — business logic correctness matters, not just code presence
- **Keep README.md in sync** — when adding, removing, or renaming commands, flags, or changing output formats, update README.md

---

# Notoma Project Details

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
```

## Rules

**Always:**
- State the goal before making changes
- Write tests for new code
- Run the full verification checklist
- Fix issues before proceeding

**Never:**
- Comment out failing tests
- Skip linter errors
- Proceed with broken tests
- Commit secrets or `.env` files
- Use goroutines without explicit need (prefer sequential code)

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

## Releases

Releases are handled by `.github/workflows/release.yml` and GoReleaser. To cut a release:

```sh
git tag v0.x.y && git push origin v0.x.y
```

GoReleaser builds binaries and publishes to the Homebrew tap (`natikgadzhi/homebrew-taps`).

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
