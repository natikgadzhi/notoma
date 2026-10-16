@README.md

# Multi-agent Work Environment

Builds in this repo run as Workflows, Claude Code's multi-agent mode, not as agent teams. Start build sessions with `/effort ultracode` (or `claude --effort ultracode`) so each phase can run as a workflow without asking first. Without ultracode, propose the phase's workflow with a rough agent count and wait for the user to write `ultracode` or "use a workflow".

## How It Works

1. The user edits tasks and discusses the plan with the lead
2. The lead (you, the main session) agrees on a plan with the user and writes it as phased task files in `tasks/backlog/`
3. Each phase runs as one workflow: builder agents implement the phase's tasks in parallel, each in its own worktree, and open a pull request; reviewer agents review each PR as soon as it opens; builders fix what reviewers flag
4. Between phases the lead stays in the loop: reads the workflow's result, merges approved PRs, keeps `main` up to date, and checks in with the user before starting the next phase

## Lead Behavior

The user will first discuss the high level plan with you. Once the user confirms we're good to start work:

1. **Read** `README.md` and other top-level docs to understand the project
2. **Plan** — Break the work into phases and tasks:
   - **Phase 0: Bootstrap** — Project scaffolding, directory structure, config files, dependency setup
   - **Phase 1: Core** — Core data models, types, interfaces
   - **Phase 2: Implementation** — Feature implementation (parallelize heavily here)
   - **Phase 3: Integration** — Wire components together, integration tests
   - **Phase 4: Polish** — Error handling, edge cases, documentation, final tests
3. **Write task files** in `tasks/backlog/` (see Task File System), each with:
   - Clear imperative title ("Implement user authentication endpoint")
   - Acceptance criteria, file paths, and dependencies
   - Edge cases that are covered, and edge cases that are not important
   - Tasks within one phase must be independent of each other. If task B needs task A's output, B goes in a later phase.
4. **Run the phase as a workflow** (load the `workflow-authoring` skill before writing the script):
   - `pipeline()` over the phase's task files with stages build → review → fix, so each PR goes to review as soon as it is ready. No barrier between build and review.
   - Builders run with `isolation: 'worktree'` and follow Builder Instructions below. They return the branch, PR URL, check results, and edge cases covered and not covered (use a `schema`).
   - Reviewers follow Reviewer Instructions below. Use two per PR with different lenses (correctness and tests; security and simplicity) and a higher `effort` than the builders. They post their review on the PR and return a verdict.
   - On "changes requested", a builder addresses the feedback on the same branch and the PR goes back to review. Allow at most 2 rounds, then return the PR as unresolved instead of looping.
   - Builders and reviewers inherit the session model; only set `model` if the user asks.
5. **Merge** once the workflow returns: merge approved PRs one at a time, rebasing the next branch if an earlier merge conflicts with it, and `git pull --ff` on `main` after each. Report unresolved PRs to the user.
6. **Repeat** for the next phase, and stop when all tasks are done

## Builder Instructions

Builders run inside the phase workflow, each in its own worktree. The lead passes the task file path and full task details in the prompt. Each builder MUST:

1. **Read the task file** and the existing code it touches before writing anything
2. **Branch** off the latest `main` as `task-N-description`, and move the task file from `tasks/backlog/` to `tasks/in-progress/` in the first commit
3. **Implement the task** — write code, tests, configs as needed
4. **Verify the work** — run the project's quality checks:

- `go build ./...` — builds cleanly
- `go vet ./...` — no vet errors
- `golangci-lint run` — no lint errors
- `go test ./...` — all tests pass
- `go test -race ./...` — if any concurrency is involved
- You are _required_ to have a strong suite of tests that follows the best practice for the project language.
- You are _required_ to run the project's linter and fix all lint errors before committing.

5. **Commit the work** with a descriptive message (include task ID):
   - Format: `[task-N] <description>`
   - One logical change per commit
6. **Move the task file** to `tasks/done/` in the PR's last commit, so merging the PR marks the task done
7. **Push and create a pull request** — MANDATORY for every task with code changes:

   ```bash
   git push -u origin task-N-description
   gh pr create --title "type(scope): [task-N] description" --body "..."
   ```

   - Every code change goes through a PR. No direct commits to `main` are allowed.

8. **Return** the PR URL, check results, and the edge cases covered and not covered
9. **Address review feedback** on the same branch and push. Resolve all of it, unless a change would introduce a security problem the reviewer did not catch; then say so on the PR.

## Reviewer Instructions

Reviewers run inside the phase workflow with the PR number, the task file path, and their review lens in the prompt. Each reviewer MUST:

1. **Check out the PR** in its worktree with `gh pr checkout <number>`
2. **Check for conflicts** — ensure the branch rebases cleanly on latest `main`. If conflicts exist, flag them and request a rebase.
3. **Run all quality checks**:
   - `go build ./...` — builds cleanly
   - `go vet ./...` — no vet errors
   - `golangci-lint run` — no lint errors
   - `go test ./...` — all tests pass (including new tests for the feature)
4. **Verify the feature works** — read the task file's acceptance criteria and confirm the implementation satisfies each criterion
5. **Code quality review**:
   - Use `/simplify` Claude skill / agent
   - Read every changed file in the diff (`gh pr diff`)
   - Check for dead code, unnecessary complexity, or missing error handling
   - Verify naming conventions and code style match the rest of the codebase
   - Ensure no commented-out code was left behind
6. **Security review**:
   - Use `/security-review` Claude skill / agent
   - Ensure secrets and tokens are not logged or exposed
   - Check that user input is validated at API/CLI boundaries
   - Flag any command injection risks (e.g. unsanitized input passed to shell)
7. **Post a PR review** via `gh pr review` with:
   - Summary of what was reviewed
   - Edge cases that ARE covered
   - Edge cases NOT covered (note whether they should be addressed now or later)
   - Any security concerns
   - Approve, request changes, or comment accordingly
8. **Return** the verdict and the list of required changes

## Git Conventions

- The main checkout stays on `main`. Never switch branches here — builders and reviewers use worktrees.
- Do `git pull --ff` after every task is completed and merged.
- Each task branch is based off the latest `main`.
- Every code change goes through a PR — no direct commits to `main`.
- Builders commit with messages: `[task-N] <description>`
- One logical change per commit — don't bundle unrelated work
- Builders should `git pull --rebase` before pushing to avoid conflicts

## Task File System

Tasks are stored as markdown files in `tasks/` with three subdirectories representing status:

```
tasks/
├── backlog/      # Not yet started
├── in-progress/  # Currently being worked on
└── done/         # Completed and merged
```

- Each task is a numbered markdown file (e.g. `01-bootstrap-go-project.md`)
- The lead creates task files in `backlog/` during planning
- Builders move their task file to `in-progress/` in their first commit
- Builders move their task file to `done/` in the PR's last commit, so it lands in `done/` when the PR merges
- Task files contain the full specification: objective, acceptance criteria, dependencies, and notes
- Task file moves are committed on the builder's branch

## Important Rules

- **Always fetch before checking task status** — run `git fetch` and `git pull --ff` before answering questions about whether a task is done, what's been merged, or what state the codebase is in
- **Always read before writing** — understand existing code before changing it
- **Test everything** — write tests for every task. If ambiguous what kind, ask the user
- **Small, focused tasks** — each task should be completable by one builder agent
- **Explicit dependencies** — if task B needs task A's output, put B in a later phase and note the dependency in its task file
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
