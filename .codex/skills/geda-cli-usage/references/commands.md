# Command Reference

## Health

```bash
go run ./cmd/geda health check --base-url=http://geda.localhost
```

## Auth

```bash
go run ./cmd/geda auth login --base-url=http://geda.localhost --email=<email> --password=<password>
go run ./cmd/geda auth whoami
go run ./cmd/geda auth logout
```

## Post

```bash
go run ./cmd/geda post list --search=<keyword> --per-page=10
go run ./cmd/geda post get --slug=<slug>
go run ./cmd/geda post upsert --file=/path/to/post.json
go run ./cmd/geda post delete --slug=<slug>
```

## Image Upload

```bash
go run ./cmd/geda post upload-image --file=/path/to/image.png --alt-vi="..." --alt-en="..."
```

## Payload Minimum For Post Upsert

```json
{
  "slug": "example-slug",
  "title": {"vi": "Tieu de", "en": "Title"},
  "body": {"vi": "<p>Noi dung</p>", "en": "<p>Content</p>"},
  "category_id": 1,
  "status": "published"
}
```
