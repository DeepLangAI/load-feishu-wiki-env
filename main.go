package main

// 从飞书多维表格读取 key/value 记录，输出为环境变量格式。
//
// 用法:
//
//	export FEISHU_APP_ID="cli_xxx"
//	export FEISHU_APP_SECRET="xxx"
//
//	# 输出 export 语句，可直接 eval
//	eval $(load-feishu-wiki-env "https://xxx.feishu.cn/wiki/XYZ?table=tblABC")
//
//	# 输出 .env 格式
//	load-feishu-wiki-env --format dotenv "https://..." > .env

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	var (
		appID      = flag.String("app-id", os.Getenv("FEISHU_APP_ID"), "飞书 App ID（或 FEISHU_APP_ID 环境变量）")
		appSecret  = flag.String("app-secret", os.Getenv("FEISHU_APP_SECRET"), "飞书 App Secret（或 FEISHU_APP_SECRET 环境变量）")
		tableIDArg = flag.String("table-id", "", "覆盖 URL 中的 table_id")
		viewIDArg  = flag.String("view-id", "", "覆盖 URL 中的 view_id")
		keyField   = flag.String("key-field", "key", "key 字段名")
		valueField = flag.String("value-field", "value", "value 字段名")
		format     = flag.String("format", "export", "输出格式: export | dotenv | json")
		outputFile = flag.String("output", "", "输出到文件（默认 stdout）")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: %s [flags] <飞书多维表格链接>\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "示例:")
		fmt.Fprintln(os.Stderr, `  eval $(load-feishu-wiki-env "https://xxx.feishu.cn/wiki/XYZ?table=tblABC")`)
		fmt.Fprintln(os.Stderr, `  load-feishu-wiki-env --format dotenv "https://..." > .env`)
		fmt.Fprintln(os.Stderr, "")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	rawURL := flag.Arg(0)

	if *appID == "" || *appSecret == "" {
		fmt.Fprintln(os.Stderr, "错误: 需要 --app-id / --app-secret，或设置环境变量 FEISHU_APP_ID / FEISHU_APP_SECRET")
		os.Exit(1)
	}

	urlType, nodeToken, tableID, viewID, err := parseFeishuURL(rawURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
	if *tableIDArg != "" {
		tableID = *tableIDArg
	}
	if *viewIDArg != "" {
		viewID = *viewIDArg
	}
	if tableID == "" {
		fmt.Fprintln(os.Stderr, "错误: URL 中未包含 table 参数，请用 --table-id 手动指定")
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "获取 tenant_access_token ...")
	token, err := getTenantAccessToken(*appID, *appSecret)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "tenant_access_token 获取成功")

	appToken := nodeToken
	if urlType == "wiki" {
		fmt.Fprintf(os.Stderr, "解析 wiki 节点 %s ...\n", nodeToken)
		appToken, err = resolveWikiAppToken(token, nodeToken)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "app_token: %s\n", appToken)
	}

	fmt.Fprintf(os.Stderr, "读取表格 app=%s table=%s ...\n", appToken, tableID)
	records, err := fetchAllRecords(token, appToken, tableID, viewID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "共 %d 条记录\n", len(records))

	var out io.Writer = os.Stdout
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}

	switch *format {
	case "export":
		writeExport(out, records, *keyField, *valueField)
	case "dotenv":
		if err := writeDotenv(out, records, *keyField, *valueField); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
	case "json":
		writeJSON(out, records, *keyField, *valueField)
	default:
		fmt.Fprintf(os.Stderr, "错误: 未知输出格式 %q，支持: export | dotenv | json\n", *format)
		os.Exit(1)
	}
}
