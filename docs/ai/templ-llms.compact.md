# templ LLM Guide (Compact)

Source: `https://templ.guide/llms.md`  
Purpose: compact, assistant-oriented reference for day-to-day coding tasks in this repo.  
Scope: preserves core templ rules, removes long tutorials and ecosystem walkthroughs.

## Assistant Changelog

- 2026-02-27: Created compact guide from upstream templ LLM docs and aligned it with repo workflow (templ generate + tests).
- 2026-02-27: Added upstream snapshot refresh workflow via `scripts/update-templ-llms.sh` and `make ai/templ-sync`.

## Core Principles

- `.templ` files are compiled to Go; run generation after edits.
- Keep normal Go code outside components and templ markup inside components.
- Prefer explicit, type-safe helper functions over ad-hoc context value assertions.
- Treat generated `*_templ.go` as build artifacts, not source.

## File Structure

- A `.templ` file starts with Go package/imports.
- Components are declared with `templ Name(args...) { ... }` and return `templ.Component`.
- You can include ordinary Go declarations outside `templ` components.

## Syntax Rules That Matter Most

- **Elements must be closed**: use `</tag>` or `/>` in source.
- **Expressions in content**: `{ expr }`.
- **Expressions in attrs**: `attr={ expr }`.
- **Control flow**: use regular Go `if`, `switch`, `for` directly in templates.
- **Raw Go in component body**: `{{ ... }}` for scoped statements.
- **Component composition**: `@OtherComponent(...)`.
- **Comments**:
  - Inside templ markup: HTML comments (`<!-- ... -->`).
  - Outside markup: standard Go comments.

## Escaping and Safety

- templ escapes dynamic content by default.
- Do not bypass escaping unless output is explicitly trusted and intentionally safe.
- For URLs, build safe URLs using templ-safe helpers/patterns.

## Context Usage in templ

- Components render with an implicit `ctx` (`context.Context`).
- Accessing missing context keys or wrong type assertions can panic.
- Prefer package helpers like `GetTheme(ctx)` over inline `ctx.Value(...).(...)` assertions.

## Fragments (HTMX-Oriented)

- Use `@templ.Fragment("name") { ... }` to mark renderable subtrees.
- Render selected fragments in handlers via `templ.WithFragments("name")` when useful.
- Only output is filtered; component logic still executes.

## What to Avoid

- Editing generated `*_templ.go` by hand.
- Mixing large, non-view business logic directly inside templates.
- Copying broad tutorial patterns that conflict with existing repo architecture.

## Repo-Specific Workflow

When touching templates in this project:

1. Edit `internal/view/*.templ` or `internal/mailer/*.templ`.
2. Regenerate templates with `make templ`.
3. Run `go test ./...`.
4. Include both source and regenerated file changes in review.

## Omitted on Purpose

This compact file intentionally excludes:

- installation instructions
- beginner end-to-end app tutorials
- unrelated framework integrations (unless needed for task)
- repeated examples that do not add new rules

Use the upstream docs for deep dives and edge-case exploration.
