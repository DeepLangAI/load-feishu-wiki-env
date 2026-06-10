package main

import (
	"encoding/json"
	"fmt"
	"io"
)

func writeExport(out io.Writer, records []map[string]any, keyField, valueField string) {
	for _, fields := range records {
		k := fieldStr(fields[keyField])
		v := fieldStr(fields[valueField])
		if k == "" {
			continue
		}
		fmt.Fprintf(out, "export %s=%q\n", k, v)
	}
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
