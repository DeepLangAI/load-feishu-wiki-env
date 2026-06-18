package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func writeExport(out io.Writer, records []map[string]any, keyField, valueField string) {
	for _, fields := range records {
		k := fieldStr(fields[keyField])
		v := fieldStr(fields[valueField])
		if k == "" {
			continue
		}
		fmt.Fprintf(out, "export %s=$'%s'\n", k, shellEscape(v))
	}
}

// shellEscape encodes s for use inside $'...' — the only format that
// correctly round-trips multi-line values through eval.
func shellEscape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\'':
			b.WriteString(`\'`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if c < 0x20 || c == 0x7f {
				fmt.Fprintf(&b, `\x%02x`, c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	return b.String()
}

func writeDotenv(out io.Writer, records []map[string]any, keyField, valueField string) error {
	for _, fields := range records {
		k := fieldStr(fields[keyField])
		v := fieldStr(fields[valueField])
		if k == "" {
			continue
		}
		fmt.Fprintf(out, "%s=%q\n", k, v)
	}
	return nil
}

func writeJSON(out io.Writer, records []map[string]any, keyField, valueField string) {
	envMap := make(map[string]string, len(records))
	for _, fields := range records {
		k := fieldStr(fields[keyField])
		v := fieldStr(fields[valueField])
		if k == "" {
			continue
		}
		envMap[k] = v
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	_ = enc.Encode(envMap)
}
