#!/usr/bin/env bash
set -euo pipefail

BASE_URL="http://geda.localhost"
EMAIL=""
PASSWORD=""
CATEGORY_SLUG="chuyen-doi-so"

for arg in "$@"; do
  case "$arg" in
    --base-url=*) BASE_URL="${arg#*=}" ;;
    --email=*) EMAIL="${arg#*=}" ;;
    --password=*) PASSWORD="${arg#*=}" ;;
    --category-slug=*) CATEGORY_SLUG="${arg#*=}" ;;
    *)
      echo "Unknown arg: $arg" >&2
      exit 1
      ;;
  esac
done

if [[ -z "$EMAIL" || -z "$PASSWORD" ]]; then
  echo "Usage: publish_post_with_image.sh --email=<email> --password=<password> [--base-url=...] [--category-slug=...]" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

cd "$CLI_DIR"

if ! go run ./cmd/geda auth whoami >/tmp/geda-skill-whoami.json 2>/tmp/geda-skill-whoami.err; then
  go run ./cmd/geda auth login --base-url="$BASE_URL" --email="$EMAIL" --password="$PASSWORD" >/tmp/geda-skill-login.json
fi

img_file="/tmp/geda-skill-$(date +%s).png"
cat <<'B64' | base64 -d > "$img_file"
iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7+RvwAAAAASUVORK5CYII=
B64

go run ./cmd/geda post upload-image --file "$img_file" --alt-vi="Anh tu skill" --alt-en="Image from skill" >/tmp/geda-skill-upload.json
media_url=$(php -r '$d=json_decode(file_get_contents($argv[1]), true); echo $d["data"]["url"] ?? "";' /tmp/geda-skill-upload.json)

slug="skill-post-$(date +%s)"
payload="/tmp/geda-skill-post-${slug}.json"
cat > "$payload" <<JSON
{
  "slug": "$slug",
  "title": {"vi": "Bai viet tu skill $slug", "en": "Post from skill $slug"},
  "excerpt": {"vi": "Demo skill", "en": "Skill demo"},
  "body": {"vi": "<p>Post tao boi skill.</p>", "en": "<p>Post created by skill.</p>"},
  "featured_image": "$media_url",
  "category_id": 1,
  "status": "published",
  "is_featured": true,
  "tags": [1]
}
JSON

go run ./cmd/geda post upsert --file "$payload" >/tmp/geda-skill-upsert.json
public_url="$BASE_URL/tin-tuc/$CATEGORY_SLUG/$slug"
http_code=$(curl -s -o /tmp/geda-skill-show.html -w "%{http_code}" "$public_url")

echo "slug=$slug"
echo "media_url=$media_url"
echo "public_url=$public_url"
echo "http_code=$http_code"
