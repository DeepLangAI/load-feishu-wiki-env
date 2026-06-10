# load-feishu-wiki-env

从飞书多维表格读取 key/value 记录，输出为环境变量格式，方便在 CI/CD 或本地开发中直接注入环境变量。

## 前置条件

1. 在[飞书开放平台](https://open.feishu.cn/)创建自建应用，记录 App ID 和 App Secret
2. 为应用开通以下权限：
   - `bitable:app:readonly`（读取多维表格）
   - `wiki:node:read` 或 `wiki:wiki:readonly`（若使用 wiki 链接）
3. 将应用添加为多维表格的协作者

## 安装

```bash
git clone <repo>
cd load-feishu-wiki-env
go build -o load-feishu-wiki-env .
```

或通过 `go install`（需已推送到 GitHub 并打好 tag）：

```bash
go install github.com/DeepLangAI/load-feishu-wiki-env@latest
```

## 快速开始

```bash
export FEISHU_APP_ID="cli_xxx"
export FEISHU_APP_SECRET="xxx"

# 注入当前 shell（export 格式）
eval $(load-feishu-wiki-env "https://xxx.feishu.cn/wiki/NodeToken?table=tblABC")

# 写入 .env 文件
load-feishu-wiki-env --format dotenv "https://xxx.feishu.cn/wiki/NodeToken?table=tblABC" > .env

# 输出 JSON
load-feishu-wiki-env --format json "https://xxx.feishu.cn/wiki/NodeToken?table=tblABC"
```

## 支持的链接格式

| 类型 | 示例 |
|------|------|
| Wiki 内嵌多维表格 | `https://xxx.feishu.cn/wiki/NodeToken?table=tblABC` |
| 独立多维表格 | `https://xxx.feishu.cn/base/AppToken?table=tblABC` |

URL 中的 `table` / `tableId` 和 `view` / `viewId` 参数均可识别。

## 表格结构

默认读取名为 `key` 和 `value` 的两列。支持文本、数字、布尔、富文本字段类型。

| key | value |
|-----|-------|
| DATABASE_URL | postgres://... |
| API_SECRET | sk-xxx |

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

## 输出格式说明

**export**（默认）：可直接 `eval` 注入当前 shell

```bash
export DATABASE_URL="postgres://user:pass@host/db"
export API_KEY="sk-abc123"
```

**dotenv**：兼容 docker compose `env_file` 和大多数 dotenv 库；值中的 `\` 和 `"` 自动转义，不支持多行值

```
DATABASE_URL="postgres://user:pass@host/db"
API_KEY="sk-abc123"
```

**json**：输出缩进 JSON 对象，方便程序读取或管道处理

```json
{
  "DATABASE_URL": "postgres://user:pass@host/db",
  "API_KEY": "sk-abc123"
}
```

## 重试机制

网络抖动、限流（HTTP 429）、服务器错误（HTTP 5xx）均会自动重试，最多 3 次，退避间隔为 1s / 2s。

## 开发

```bash
go test ./...
```
