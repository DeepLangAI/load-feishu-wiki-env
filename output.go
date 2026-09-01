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
		quoted, err := quoteDotenvValue(k, v)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s=%s\n", k, quoted)
	}
	return nil
}

// quoteDotenvValue wraps a value so every mainstream dotenv reader gives it back
// verbatim.
//
// Single quotes are the only portable answer. Reading a double-quoted value,
// docker compose env_file, godotenv, python-dotenv and node dotenv all interpolate
// variables or unescape backslashes: a $NAME inside the value is taken as a
// variable reference, and compose substitutes it with an empty string — a password
// containing $ arrives at the service with a chunk cut out. Single-quoted values
// come back untouched from all four, newlines included.
//
// The cost is that single quotes have no escape mechanism, so a value containing
// a single quote falls back to double quotes (%q). If such a value also contains
// $ there is no correct output: the double-quote escapes disagree across readers —
// \$ works only for the Go ones (godotenv, compose), $$ only for compose, while
// python and node hand back a literal backslash or a doubled dollar. Better to
// fail loudly than to ship an .env that is silently wrong on some consumer.
//
// Values containing \r also take the double-quote path: dotenv is a line-oriented
// format and a bare CR splits lines differently across readers, whereas the %q
// \r escape is understood by all four.
func quoteDotenvValue(key, value string) (string, error) {
	if !strings.ContainsAny(value, "'\r") {
		return "'" + value + "'", nil
	}
	if strings.Contains(value, "$") {
		return "", fmt.Errorf("%s: 值里同时有单引号和 $，dotenv 格式无法安全表达（双引号的 $ 转义写法各家 dotenv 解析器互不兼容），请改用 --format export 或 --format json，或调整该值", key)
	}
	return fmt.Sprintf("%q", value), nil
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
