# load-feishu-wiki-env

从飞书多维表格读取 key/value 记录，输出为环境变量格式，方便在 CI/CD 或本地开发中直接注入环境变量。

## 前置条件

1. 在[飞书开放平台](https://open.feishu.cn/)创建自建应用，记录 App ID 和 App Secret
2. 为应用开通以下权限：
   - `bitable:app:readonly`（读取多维表格）
   - `wiki:node:read` 或 `wiki:wiki:readonly`（若使用 wiki 链接）
3. 将应用添加为多维表格的协作者

## 安装

**从 GitHub Releases 下载预编译二进制**（推荐）：

```bash
# macOS Apple Silicon
curl -L https://github.com/DeepLangAI/load-feishu-wiki-env/releases/download/latest/load-feishu-wiki-env_darwin_arm64.tar.gz | tar xz
sudo mv load-feishu-wiki-env /usr/local/bin/
```

其他平台替换 `darwin_arm64` 为对应平台：`linux_amd64`、`linux_arm64`、`darwin_amd64`、`windows_amd64`（`.zip`）。

也可以前往 [Releases 页面](https://github.com/DeepLangAI/load-feishu-wiki-env/releases/tag/latest) 手动下载。

**通过 `go install`**：

```bash
GOPROXY=direct go install github.com/DeepLangAI/load-feishu-wiki-env@latest
```

**从源码编译**：

```bash
git clone https://github.com/DeepLangAI/load-feishu-wiki-env.git
cd load-feishu-wiki-env
go build -o load-feishu-wiki-env .
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

**export**（默认）：使用 `$'...'` 语法，可直接 `eval` 注入当前 shell，正确支持含换行符的值（如证书、SSH key）

```bash
export DATABASE_URL=$'postgres://user:pass@host/db'
export PRIVATE_KEY=$'-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----'
```

**dotenv**：兼容 docker compose `env_file` 和大多数 dotenv 库；`\`、`"` 自动转义，换行符以 `\n` 转义存储，支持多行值

```
DATABASE_URL="postgres://user:pass@host/db"
PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----"
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

---

## 在 GitHub Actions 中使用

### 整体流程

```
飞书多维表格（存储生产密钥）
  → CI 调用 load-feishu-wiki-env 生成 .env 文件      # 从飞书读取所有 key/value
  → SCP 将 .env 和 docker-compose.yaml 推送到服务器   # 传输配置文件
  → docker-compose 通过 env_file 将变量注入容器        # 容器启动时加载
  → Go 应用通过 cleanenv 读取 env tag 字段            # 运行时覆盖 yaml 配置中的敏感值
```

### 飞书表格结构

表格需包含 `key` 和 `value` 两列，每行是一个环境变量，示例：

| key | value |
|-----|-------|
| MONGO_ADDR | mongodb://user:pass@host:27017/dbname |
| REDIS_USERNAME | default |
| REDIS_PASSWORD | your-redis-password |
| SESSION_SECRET | your-session-secret |
| CRYPTO_SECRET | your-crypto-secret |
| JUST_ONE_API_TOKEN | sk-xxx |
| NOTIFY_FEISHU_WEBHOOK | https://open.feishu.cn/open-apis/bot/v2/hook/xxx |

### GitHub Secrets 配置

在仓库的 **Settings → Secrets and variables → Actions** 中添加以下三个 Secret：

| Secret | 说明 | 获取方式 |
|--------|------|----------|
| `FEISHU_APP_ID` | 飞书自建应用的 App ID | 飞书开放平台 → 应用详情 → 凭证与基础信息 |
| `FEISHU_APP_SECRET` | 飞书自建应用的 App Secret | 同上 |
| `FEISHU_ENV_TABLE_URL` | 多维表格链接（含 table 参数） | 打开目标表格，复制浏览器地址栏 URL |

飞书应用需已开通 `bitable:app:readonly` 权限，并作为协作者加入目标多维表格。

### GitHub Actions workflow 示例

在 deploy job 中，安装工具生成 .env，再通过 SCP/SSH 部署：

```yaml
deploy:
  needs: build-and-push
  if: github.event_name == 'push' && github.ref == 'refs/heads/master'
  runs-on: ubuntu-latest
  steps:
    - name: Checkout
      uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
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

    - name: Copy files to server
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

`go install` 会将二进制安装到 `$GOPATH/bin`，GitHub Actions 默认已将该目录加入 `$PATH`，可直接调用 `load-feishu-wiki-env`。

### docker-compose 配置

用 `env_file` 指向同目录下的 `.env`：

```yaml
services:
  app:
    image: ghcr.io/your-org/your-app:latest
    restart: always
    environment:
      MODE_ENV: prod
    env_file: ./.env   # 由 CI 从飞书表格生成，包含所有运行时密钥
```

多个服务共享同一份 `.env` 时，每个服务都可以声明 `env_file: ./.env`。

### Go 应用集成（cleanenv）

安装依赖：

```bash
go get github.com/ilyakaznacheev/cleanenv@v1.5.0
go get github.com/joho/godotenv@v1.5.1
```

在配置 struct 中，敏感字段同时声明 `yaml` tag 和 `env` tag：

```go
type System struct {
    SessionSecret string `yaml:"session_secret" env:"SESSION_SECRET"`
    CryptoSecret  string `yaml:"crypto_secret"  env:"CRYPTO_SECRET"`
    EmailWebHook  string `yaml:"email_web_hook"  env:"EMAIL_WEB_HOOK"`
}

type Mongo struct {
    Addr   string `yaml:"addr"    env:"MONGO_ADDR"`
    DbName string `yaml:"db_name"`
}

type Redis struct {
    Addrs    []string `yaml:"addrs"`
    Username string   `yaml:"username" env:"REDIS_USERNAME"`
    Password string   `yaml:"password" env:"REDIS_PASSWORD"`
    UseTls   bool     `yaml:"use_tls"`
}
```

`Init()` 中的加载顺序：先读 yaml 配置，再用环境变量覆盖敏感字段：

```go
import (
    "github.com/ilyakaznacheev/cleanenv"
    "github.com/joho/godotenv"
    "gopkg.in/yaml.v3"
)

func Init() {
    godotenv.Load(filepath.Join(GetProjectPath(), ".env")) // 加载 .env（文件不存在时静默跳过）

    yaml.Unmarshal(dataBytes, &ConfigData)  // 读取 yaml，填充非敏感配置

    cleanenv.ReadEnv(&ConfigData)           // 用环境变量覆盖带 env tag 的敏感字段
}
```

`cleanenv.ReadEnv` 只覆盖带 `env` tag 的字段，yaml 中的非敏感配置不受影响。  
yaml 配置文件中，敏感字段留空即可：

```yaml
system:
  session_secret: ""  # env: SESSION_SECRET
  crypto_secret: ""   # env: CRYPTO_SECRET

mongo:
  addr: ""            # env: MONGO_ADDR
  db_name: "mydb"

redis:
  addrs: ["redis-host:6379"]
  username: ""        # env: REDIS_USERNAME
  password: ""        # env: REDIS_PASSWORD
```

### 本地开发

在项目根目录创建 `.env` 文件，填入本地开发用的密钥：

```bash
# .env（本地开发用，不提交到 git）
MONGO_ADDR=mongodb://localhost:27017/mydb
REDIS_PASSWORD=
SESSION_SECRET=local-dev-secret
```

`godotenv.Load` 在文件不存在时会静默跳过，不影响生产环境。务必将 `.env` 加入 `.gitignore`：

```gitignore
.env
```
