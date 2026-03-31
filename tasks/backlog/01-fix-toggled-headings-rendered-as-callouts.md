# Fix toggled headings rendered as callouts instead of headings

## Problem

Notion toggle headings (h2, h3 with `is_toggleable: true`) are currently rendered as Obsidian callouts (`> [!faq]- ...`) instead of regular markdown headings. This is wrong — they should be plain headings (`##`, `###`), since Obsidian headings are natively togglable.

## Example

**Notion source** (toggle h3 with children):
```
### December 2, 2025
- Retro
    - In Obsidian, figure out how to share
- This month
- Q1
```

**Current (wrong) output:**
```markdown
> [!faq]- [2025-12-02](Periodic/Days/2025-12-02.md)
> - Retro
>     - In Obsidian, figure out how to share
> - This month
> - Q1
```

**Expected output:**
```markdown
### [2025-12-02](Periodic/Days/2025-12-02.md)

- Retro
    - In Obsidian, figure out how to share
- This month
- Q1
```

## Root Cause

The transform code likely checks `is_toggleable` on heading blocks and wraps them in callout syntax. Instead, it should emit a normal heading and render children as regular block content underneath.

## Acceptance Criteria

- Toggle headings (h1, h2, h3 with `is_toggleable: true`) render as plain markdown headings (`#`, `##`, `###`)
- Children of toggle headings render as normal block content below the heading (not inside a callout blockquote)
- Non-toggle headings continue to work as before
- Date formatting and daily note linking in heading text is preserved
- Existing tests updated, new tests added for toggle heading cases

## Files to Investigate

- `internal/transform/markdown.go` — block-to-markdown conversion, likely where callout wrapping happens
- `internal/transform/markdown_test.go` — add/update test cases
