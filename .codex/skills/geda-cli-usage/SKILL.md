---
name: geda-cli-usage
description: Use when working with geda-cli in this workspace for running commands, validating auth and API connectivity, creating or updating posts, uploading images, and debugging CLI-to-geda-web integration issues.
---

# Geda CLI Usage

## Overview

Use this skill to execute reliable `geda-cli` workflows against local `geda-web`, especially for auth, health checks, content upsert, image upload, and public-site verification.

## Core Workflow

1. Work from the `geda-cli` repository root for CLI commands.
2. Confirm API availability first:
```bash
go run ./cmd/geda health check --base-url=http://geda.localhost
```
3. Ensure auth profile exists:
```bash
go run ./cmd/geda auth login --base-url=http://geda.localhost --email=<email> --password=<password>
go run ./cmd/geda auth whoami
```
4. Run task-specific command (`post upsert`, `post upload-image`, resource CRUD).
5. For user-visible post tasks, verify both API and public page.

## Publish Post With Image

Run the deterministic helper script when user asks to create a live demo post:

```bash
./.codex/skills/geda-cli-usage/scripts/publish_post_with_image.sh \
  --base-url=http://geda.localhost \
  --email=admin@geda.vn \
  --password=password
```

The script:
- logs in if needed
- uploads a tiny image using `post upload-image`
- creates a published post with `featured_image`
- prints slug and public URL
- verifies the post returns HTTP 200

## Verification Rules

1. Prefer `go run ./cmd/geda ...` during development.
2. After CLI code changes, run:
```bash
go test ./...
```
3. For cross-repo changes (API contract or permissions), also run relevant `geda-web` API tests and smoke checks.
4. If public post is not visible, check `status=published` and `published_at` is set and not in the future.

## Troubleshooting

Use [references/troubleshooting.md](references/troubleshooting.md) for known failures and fixes.
Use [references/commands.md](references/commands.md) for common command templates.
