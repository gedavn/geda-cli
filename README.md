# geda-cli

CLI for `geda-web` API (`/api/v1/*`), including:
- health check
- auth login/logout/whoami
- CRUD for content resources (post, category, tag, page, product)
- image upload for posts

## Requirements

- Go 1.26+
- running `geda-web` with API v1 enabled

## Build and run

```bash
go build ./cmd/geda
./geda health check --base-url=http://geda.localhost
```

Or run directly:

```bash
go run ./cmd/geda health check --base-url=http://geda.localhost
```

## Tests

```bash
go test ./...
```

## Authentication

Login:

```bash
go run ./cmd/geda auth login \
  --base-url=http://geda.localhost \
  --email=admin@geda.vn \
  --password=password
```

Current user:

```bash
go run ./cmd/geda auth whoami
```

Logout:

```bash
go run ./cmd/geda auth logout
```

Profile path:
- `~/.config/geda-cli/config.json`

## Main commands

```text
geda auth <login|logout|whoami>
geda health check [--base-url=...]
geda post <list|get|upsert|delete|import|upload-image>
geda category <list|get|upsert|delete>
geda tag <list|get|upsert|delete>
geda page <list|get|upsert|delete>
geda product <list|get|upsert|delete>
geda settings <list|get|set>
```

## Upload image for post

```bash
go run ./cmd/geda post upload-image \
  --file=/path/to/image.png \
  --alt-vi="Post image (VI)" \
  --alt-en="Post image (EN)"
```

The response includes `data.url`. Use this URL for `featured_image` or `og_image` in post payload.

## Create or update post with image

Example `post.json`:

```json
{
  "slug": "post-with-image",
  "title": {
    "vi": "Bai viet co hinh",
    "en": "Post with image"
  },
  "excerpt": {
    "vi": "Tom tat",
    "en": "Summary"
  },
  "body": {
    "vi": "<p>Noi dung tieng Viet</p>",
    "en": "<p>English content</p>"
  },
  "featured_image": "http://geda.localhost/storage/media/2026/02/example.png",
  "category_id": 1,
  "status": "draft",
  "tags": [1]
}
```

Upsert by slug:

```bash
go run ./cmd/geda post upsert --file=post.json
```

Get post:

```bash
go run ./cmd/geda post get --slug=post-with-image
```

## Exit codes

- `0`: success
- `1`: validation or command input error
- `2`: auth/permission error
- `3`: network/request/server error
