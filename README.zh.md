# load-feishu-wiki-env

**把飞书当配置中心用。**

从飞书多维表格读取环境变量，一条命令注入任意 shell、CI 流水线或 Docker 部署 —— 不需要额外的密钥管理服务。

[![Go Report Card](https://goreportcard.com/badge/github.com/DeepLangAI/load-feishu-wiki-env)](https://goreportcard.com/report/github.com/DeepLangAI/load-feishu-wiki-env)
[![Release](https://img.shields.io/github/v/release/DeepLangAI/load-feishu-wiki-env)](https://github.com/DeepLangAI/load-feishu-wiki-env/releases/tag/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## 痛点

团队已经活在飞书里，但密钥却散落在 `.env` 文件、群消息和各人的备忘录里 —— 或者锁在只有运维才能动的工具里。

**`load-feishu-wiki-env` 把飞书多维表格变成一个活的配置仓库。** 在表格里改一行，所有环境立刻拿到最新值。不需要额外基础设施，不需要把 `.env` 提交到 git。

```bash
# 把飞书表格中的所有变量注入当前 shell
eval $(load-feishu-wiki-env "https://xxx.feishu.cn/wiki/NodeToken?table=tblABC")
```

---

## 工作原理

```
飞书多维表格（唯一配置源）
  └─▶  load-feishu-wiki-env  ──▶  export / .env / JSON
                                    │
                          ┌─────────┴──────────┐
                     eval 注入 shell         CI 生成 .env
                     （本地开发）            docker compose 加载
```

1. **存储** — 在飞书多维表格里维护 `key`/`value` 键值对，任何人都可以查看和修改。
2. **拉取** — 在 CI 或本地执行一条命令，获取最新配置。
3. **注入** — 管道输出到 shell、写入 `.env` 文件，或消费 JSON。

---

## 安装

**下载预编译二进制**（推荐，无需 Go 工具链）：

```bash
# macOS Apple Silicon
curl -L https://github.com/DeepLangAI/load-feishu-wiki-env/releases/download/latest/load-feishu-wiki-env_darwin_arm64.tar.gz | tar xz
sudo mv load-feishu-wiki-env /usr/local/bin/
```

其他平台替换 `darwin_arm64`：

| 平台 | 后缀 |
|---|---|
| Linux x86-64 | `linux_amd64` |
| Linux ARM64 | `linux_arm64` |
| macOS Intel | `darwin_amd64` |
| Windows x86-64 | `windows_amd64`（`.zip`）|

也可前往 [Releases 页面](https://github.com/DeepLangAI/load-feishu-wiki-env/releases/tag/latest) 手动下载。

**通过 `go install`**：

```bash
GOPROXY=direct go install github.com/DeepLangAI/load-feishu-wiki-env@latest
```

---

## 快速开始

**第一步 — 创建飞书自建应用**，在[飞书开放平台](https://open.feishu.cn/)记录 App ID 和 App Secret，并开通以下权限：
- `bitable:app:readonly`
- `wiki:node:read`（使用 Wiki 链接时）

将应用添加为多维表格的协作者。

**第二步 — 配置凭证**：

```bash
export FEISHU_APP_ID="cli_xxx"
export FEISHU_APP_SECRET="xxx"
```

**第三步 — 拉取配置**：

```bash
# 注入当前 shell
eval $(load-feishu-wiki-env "https://xxx.feishu.cn/wiki/NodeToken?table=tblABC")

# 写入 .env 文件（Docker、本地开发）
load-feishu-wiki-env --format dotenv --output .env \
  "https://xxx.feishu.cn/wiki/NodeToken?table=tblABC"

# 输出 JSON（脚本处理）
load-feishu-wiki-env --format json "https://xxx.feishu.cn/wiki/NodeToken?table=tblABC"
```

---

## 表格结构

在多维表格中创建 `key` 和 `value` 两列：

| key | value |
|-----|-------|
| `DATABASE_URL` | `postgres://user:pass@host/db` |
| `API_SECRET` | `sk-xxx` |
| `PRIVATE_KEY` | `-----BEGIN RSA PRIVATE KEY-----\n...` |

支持文本、数字、布尔、富文本字段类型。多行值（证书、SSH key）在所有输出格式中均可正确处理。

---

## 支持的链接格式

| 类型 | 示例 |
|------|------|
| Wiki 内嵌多维表格 | `https://xxx.feishu.cn/wiki/NodeToken?table=tblABC` |
| 独立多维表格 | `https://xxx.feishu.cn/base/AppToken?table=tblABC` |

`table`/`tableId` 和 `view`/`viewId` 参数均可识别。

---

## 输出格式

**`export`**（默认）— 使用 `$'...'` 语法，可直接 `eval` 注入当前 shell，正确支持含换行符的值：

```bash
export DATABASE_URL=$'postgres://user:pass@host/db'
export PRIVATE_KEY=$'-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----'
```

**`dotenv`** — 兼容 `docker compose env_file` 和主流 dotenv 库，`\`、`"` 自动转义：

```
DATABASE_URL="postgres://user:pass@host/db"
PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----"
```

**`json`** — 缩进 JSON 对象，便于程序读取或管道处理：

```json
{
  "DATABASE_URL": "postgres://user:pass@host/db",
  "API_SECRET": "sk-abc123"
}
```

---

## 选项

```
用法: load-feishu-wiki-env [flags] <飞书多维表格链接>

  --app-id      飞书 App ID（或环境变量 FEISHU_APP_ID）
  --app-secret  飞书 App Secret（或环境变量 FEISHU_APP_SECRET）
  --table-id    覆盖 URL 中的 table_id
  --view-id     覆盖 URL 中的 view_id
  --key-field   key 列名（默认: key）
  --value-field value 列名（默认: value）
  --format      输出格式: export | dotenv | json（默认: export）
  --output      写入文件，默认输出到 stdout
```

---

## GitHub Actions 集成

用一张飞书表格代替一堆 GitHub Secrets。运维在表格里改完，下一次 CI 触发就自动生效。

### 流程

```
飞书多维表格  →  load-feishu-wiki-env  →  .env  →  SCP 到服务器  →  docker compose up
```

### 需要配置的 GitHub Secrets

| Secret | 说明 |
|--------|------|
| `FEISHU_APP_ID` | 飞书自建应用的 App ID |
| `FEISHU_APP_SECRET` | 飞书自建应用的 App Secret |
| `FEISHU_ENV_TABLE_URL` | 多维表格链接（从浏览器地址栏复制） |

### Workflow 示例

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

    - name: 从飞书生成 .env
      env:
        FEISHU_APP_ID: ${{ secrets.FEISHU_APP_ID }}
        FEISHU_APP_SECRET: ${{ secrets.FEISHU_APP_SECRET }}
      run: |
        go install github.com/DeepLangAI/load-feishu-wiki-env@latest
        load-feishu-wiki-env \
          --format dotenv \
          --output .env \
          "${{ secrets.FEISHU_ENV_TABLE_URL }}"

    - name: 推送文件到服务器
      uses: appleboy/scp-action@v1
      with:
        host: ${{ secrets.DEPLOY_HOST }}
        username: ${{ secrets.DEPLOY_USER }}
        key: ${{ secrets.DEPLOY_KEY }}
        source: "docker-compose.yaml,.env"
        target: ${{ secrets.DEPLOY_PATH }}

    - name: 部署
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

### docker-compose 配置

```yaml
services:
  app:
    image: ghcr.io/your-org/your-app:latest
    restart: always
    env_file: ./.env   # 由 CI 从飞书表格生成
```

---

## 可靠性

网络抖动、限流（HTTP 429）、服务器错误（HTTP 5xx）均自动重试，最多 3 次，退避间隔 1s / 2s。

---

## 开发

```bash
go test ./...
```
