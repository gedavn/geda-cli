# Troubleshooting

## `missing CLI profile, run \`geda auth login\``

Cause: no saved auth profile.

Fix:
```bash
go run ./cmd/geda auth login --base-url=http://geda.localhost --email=<email> --password=<password>
```

## `failed to decode response: invalid character '<'`

Cause: backend returned HTML error instead of JSON.

Fix:
1. Check API directly with curl.
2. Fix `geda-web` runtime/dependency issue.
3. Re-run CLI command.

## Post not visible on website

Cause: public pages only show published posts.

Check:
- `status` must be `published`
- `published_at` must be non-null and `<= now`

## 403 forbidden on resource commands

Cause: user/token missing required permission.

Fix:
- Use account with matching permission (`manage posts`, `manage media`, etc.)
- Re-login to refresh token context if permissions changed.
