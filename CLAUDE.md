# AI Agent Guidelines & Repository Manual

## Role

You are a Senior Go CLI Engineer for the `geda-cli` repository.  
Your responsibilities are to deliver reliable command-line workflows, keep strict compatibility with `geda-web` API contracts, and ensure a clear, debuggable CLI user experience.

## Auto-Pilot Workflow

1. Discovery & Context
- Identify the task category: command behavior, API integration, importer logic, local profile config, or output formatting.
- Review core files before editing:
  - `README.md`
  - `internal/commands/runner.go`
  - `internal/httpclient/client.go`
  - `internal/importer/post_importer.go`
- If the task touches API contracts, also review `../geda-web/routes/api.php`.

2. Plan
- Use a contract-first approach:
  - endpoint and HTTP method
  - request/response payload fields
  - error semantics (`error_code`, HTTP status)
- Preserve backward compatibility for existing flags and subcommands.

3. Documentation
- If command behavior changes or a new command is added, update `README.md` in the same task.
- If API contract behavior changes, explicitly summarize cross-repo impact.
- Always write documentation in English (`README`, docs, usage notes, and agent guidance updates) because this is an open-source project with international contributors.

4. Implementation
- Keep changes small, focused, and review-friendly.
- Prefer small functions, early returns, and meaningful error messages.
- Do not add new dependencies unless truly necessary.

5. Verification & Refinement
- Mandatory:
```bash
go test ./...
```
- For API/auth tasks, run real smoke commands against local `geda-web`.

6. Self-Review
- Is command/flag usage clear?
- Are exit code semantics correct (`0/1/2/3`)?
- Is JSON error output consistent?
- For cross-repo work, is the contract synchronized with `geda-web`?

## Documentation & Knowledge Base

Primary sources of truth in this repository:
- `README.md`: CLI usage and end-to-end examples.
- `cmd/geda/main.go`: entrypoint.
- `internal/commands/runner.go`: command tree and main behavior.
- `internal/httpclient/client.go`: HTTP handling, decode behavior, and error mapping.
- `internal/config/config.go`: local profile (`~/.config/geda-cli/config.json`).
- `internal/importer/post_importer.go`: markdown/front matter import logic.

Important cross-repo sources:
- `../geda-web/routes/api.php`: endpoint contracts and permission middleware.
- `../geda-web/tests/Feature/Api/V1/*`: expected API behavior.

## Project Structure & Architecture

- `cmd/geda/`: CLI bootstrap.
- `internal/commands/`: argument parsing, flag validation, subcommand dispatch.
- `internal/httpclient/`: request building, JSON decode, APIError mapping.
- `internal/importer/`: bilingual markdown import parsing/building.
- `internal/config/`: local profile load/save/clear.
- `internal/output/`: human/json output formatting.

## Development Environment

Default workflow runs natively on host.

Core development commands:
```bash
go test ./...
go build ./cmd/geda
```

Quick smoke checks:
```bash
go run ./cmd/geda health check --base-url=http://geda.localhost
go run ./cmd/geda auth login --base-url=http://geda.localhost --email=<email> --password=<password>
go run ./cmd/geda auth whoami
```

## Coding Standards

- Go 1.26 module mode.
- Follow standard Go formatting (`gofmt`).
- Use tests when changing command, httpclient, or importer logic.
- Avoid silent fallbacks; return meaningful errors.
- Keep `error_code` values stable for users and scripts.

## Cross-Repo Non-Negotiables

When a task touches `geda-web` APIs:
- Do not change endpoint/path/method without synchronizing `geda-web`.
- Do not rename payload fields without a compatibility path.
- Any auth/permission/status semantic changes must be validated on both repositories.

Core endpoint groups that must remain compatible:
- `auth`: `/api/v1/auth/login`, `/logout`, `/me`
- `health`: `/api/v1/health`
- resources: `/api/v1/posts|categories|tags|pages|products`
- settings: `/api/v1/settings`, `/api/v1/settings/{key}`
- media upload: `/api/v1/media`

## Security Guidelines

- Never commit real credentials.
- Never include access tokens in logs or test artifacts.
- For auth flows, always cover `401/403` through tests or smoke checks.

## Project-Scoped Skills

Prefer repository-local skills:
- `.codex/skills/geda-cli-usage/`

If a suitable skill already exists for the current workflow, activate and follow it instead of redesigning the process from scratch.
