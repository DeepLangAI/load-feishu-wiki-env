# load-feishu-wiki-env

[中文文档](README.zh.md)

**Use Feishu as your Config Center.**

Load environment variables from Feishu Wiki / Bitable into any shell, CI pipeline, or Docker deployment — no secret manager required.

[![Go Report Card](https://goreportcard.com/badge/github.com/DeepLangAI/load-feishu-wiki-env)](https://goreportcard.com/report/github.com/DeepLangAI/load-feishu-wiki-env)
[![Release](https://img.shields.io/github/v/release/DeepLangAI/load-feishu-wiki-env)](https://github.com/DeepLangAI/load-feishu-wiki-env/releases/tag/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## The Problem

Your team already lives in Feishu. But secrets are scattered across `.env` files, chat messages, and sticky notes — or locked in a tool nobody except ops can touch.

**`load-feishu-wiki-env` turns a Feishu Bitable into a live config store.** Edit credentials in the spreadsheet you already have, pull them into any environment with one command. No extra infrastructure. No `.env` files in git.

```bash
# Inject all env vars from a Feishu table into your current shell
eval $(load-feishu-wiki-env "https://xxx.feishu.cn/wiki/NodeToken?table=tblABC")
```

---

## How It Works

```
Feishu Bitable (your source of truth)
  └─▶  load-feishu-wiki-env  ──▶  export / .env / JSON
                                    │
                          ┌─────────┴──────────┐
                     eval in shell         CI writes .env
                     (local dev)           docker compose loads it
```

1. **Store** — keep `key`/`value` pairs in a Feishu table. Any teammate can view or update them.
2. **Pull** — run one command in CI or locally to fetch the latest values.
3. **Inject** — pipe into your shell, write a `.env` file, or consume JSON.

---

## Installation

**Download binary** (recommended — no Go toolchain needed):

```bash
# macOS Apple Silicon
curl -L https://github.com/DeepLangAI/load-feishu-wiki-env/releases/download/latest/load-feishu-wiki-env_darwin_arm64.tar.gz | tar xz
sudo mv load-feishu-wiki-env /usr/local/bin/
```

Other platforms — replace `darwin_arm64` with your target:

| Platform | Suffix |
|---|---|
| Linux x86-64 | `linux_amd64` |
| Linux ARM64 | `linux_arm64` |
| macOS Intel | `darwin_amd64` |
| Windows x86-64 | `windows_amd64` (`.zip`) |

Or browse [Releases](https://github.com/DeepLangAI/load-feishu-wiki-env/releases/tag/latest) to download manually.

**Via `go install`**:

```bash
GOPROXY=direct go install github.com/DeepLangAI/load-feishu-wiki-env@latest
```

---

## Quick Start

**Step 1 — Create a Feishu app** at [open.feishu.cn](https://open.feishu.cn/), grab `App ID` and `App Secret`, and grant it:
- `bitable:app:readonly`
- `wiki:node:read` (if using a Wiki link)

Add the app as a collaborator on your Bitable.

**Step 2 — Set credentials**:

```bash
export FEISHU_APP_ID="cli_xxx"
export FEISHU_APP_SECRET="xxx"
```

**Step 3 — Pull config**:

```bash
# Inject into current shell
eval $(load-feishu-wiki-env "https://xxx.feishu.cn/wiki/NodeToken?table=tblABC")

# Write .env file (for Docker, local dev)
load-feishu-wiki-env --format dotenv --output .env \
  "https://xxx.feishu.cn/wiki/NodeToken?table=tblABC"

# Output JSON (for scripting)
load-feishu-wiki-env --format json "https://xxx.feishu.cn/wiki/NodeToken?table=tblABC"
```

---

## Table Structure

Create a Bitable with two columns named `key` and `value`:

| key | value |
|-----|-------|
| `DATABASE_URL` | `postgres://user:pass@host/db` |
| `API_SECRET` | `sk-xxx` |
| `PRIVATE_KEY` | `-----BEGIN RSA PRIVATE KEY-----\n...` |

Supports text, number, boolean, and rich-text field types. Multi-line values (certificates, SSH keys) are handled correctly in all output formats.

---

## Supported URL Formats

| Type | Example |
|------|---------|
| Wiki-embedded Bitable | `https://xxx.feishu.cn/wiki/NodeToken?table=tblABC` |
| Standalone Bitable | `https://xxx.feishu.cn/base/AppToken?table=tblABC` |

Both `table`/`tableId` and `view`/`viewId` query params are recognized.

---

## Output Formats

**`export`** (default) — `eval`-safe, handles multi-line values with `$'...'` syntax:

```bash
export DATABASE_URL=$'postgres://user:pass@host/db'
export PRIVATE_KEY=$'-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----'
```

**`dotenv`** — compatible with `docker compose env_file` and most dotenv libraries:

```
DATABASE_URL="postgres://user:pass@host/db"
PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----"
```

**`json`** — indented JSON object for programmatic use:

```json
{
  "DATABASE_URL": "postgres://user:pass@host/db",
  "API_SECRET": "sk-abc123"
}
```

---

## Options

```
Usage: load-feishu-wiki-env [flags] <feishu-bitable-url>

  --app-id      Feishu App ID (or env FEISHU_APP_ID)
  --app-secret  Feishu App Secret (or env FEISHU_APP_SECRET)
  --table-id    Override table_id from URL
  --view-id     Override view_id from URL
  --key-field   Name of the key column (default: key)
  --value-field Name of the value column (default: value)
  --format      Output format: export | dotenv | json (default: export)
  --output      Write to file instead of stdout
```

---

## GitHub Actions Integration

Replace a pile of GitHub Secrets with a single Feishu table. Your ops team edits the spreadsheet; CI fetches the latest on every deploy.

### Workflow

```
Feishu Bitable  →  load-feishu-wiki-env  →  .env  →  SCP to server  →  docker compose up
```

### Required GitHub Secrets

| Secret | Description |
|--------|-------------|
| `FEISHU_APP_ID` | Your Feishu app's App ID |
| `FEISHU_APP_SECRET` | Your Feishu app's App Secret |
| `FEISHU_ENV_TABLE_URL` | Bitable URL (copy from browser) |

### Example Workflow

```yaml
deploy:
  needs: build-and-push
  if: github.event_name == 'push' && github.ref == 'refs/heads/master'
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4

    - uses: actions/setup-go@v5
      with:
        go-version: '1.22'
        cache: false

    - name: Generate .env from Feishu
      env:
        FEISHU_APP_ID: ${{ secrets.FEISHU_APP_ID }}
        FEISHU_APP_SECRET: ${{ secrets.FEISHU_APP_SECRET }}
      run: |
        go install github.com/DeepLangAI/load-feishu-wiki-env@latest
        load-feishu-wiki-env \
          --format dotenv \
          --output .env \
          "${{ secrets.FEISHU_ENV_TABLE_URL }}"

    - name: Copy to server
      uses: appleboy/scp-action@v1
      with:
        host: ${{ secrets.DEPLOY_HOST }}
        username: ${{ secrets.DEPLOY_USER }}
        key: ${{ secrets.DEPLOY_KEY }}
        source: "docker-compose.yaml,.env"
        target: ${{ secrets.DEPLOY_PATH }}

    - name: Deploy
      uses: appleboy/ssh-action@v1
      with:
        host: ${{ secrets.DEPLOY_HOST }}
        username: ${{ secrets.DEPLOY_USER }}
        key: ${{ secrets.DEPLOY_KEY }}
        script: |
          cd ${{ secrets.DEPLOY_PATH }}
          sudo docker compose pull
          sudo docker compose up -d
```

### docker-compose

```yaml
services:
  app:
    image: ghcr.io/your-org/your-app:latest
    restart: always
    env_file: ./.env   # generated by CI from Feishu
```

---

## Reliability

Network blips, rate limits (HTTP 429), and server errors (HTTP 5xx) are retried automatically — up to 3 attempts with 1s / 2s backoff.

---

## Development

```bash
go test ./...
```
