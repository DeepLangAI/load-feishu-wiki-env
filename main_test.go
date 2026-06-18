package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------- parseFeishuURL ----------

func TestParseFeishuURL(t *testing.T) {
	cases := []struct {
		name      string
		url       string
		wantType  string
		wantToken string
		wantTable string
		wantView  string
		wantErr   bool
	}{
		{
			name:      "wiki with table and view",
			url:       "https://example.feishu.cn/wiki/NodeXYZ?table=tblABC&view=vewDEF",
			wantType:  "wiki",
			wantToken: "NodeXYZ",
			wantTable: "tblABC",
			wantView:  "vewDEF",
		},
		{
			name:      "base with tableId and viewId aliases",
			url:       "https://example.feishu.cn/base/AppToken123?tableId=tblFOO&viewId=vewBAR",
			wantType:  "base",
			wantToken: "AppToken123",
			wantTable: "tblFOO",
			wantView:  "vewBAR",
		},
		{
			name:      "fragment stripped from path",
			url:       "https://example.feishu.cn/wiki/NodeABC#heading?table=tblX",
			wantType:  "wiki",
			wantToken: "NodeABC",
			wantTable: "tblX",
		},
		{
			name:      "fragment stripped from query",
			url:       "https://example.feishu.cn/wiki/NodeABC?table=tblX#heading",
			wantType:  "wiki",
			wantToken: "NodeABC",
			wantTable: "tblX",
		},
		{
			name:    "unsupported url type",
			url:     "https://example.feishu.cn/docs/DocToken?table=tblX",
			wantErr: true,
		},
		{
			name:    "path too short",
			url:     "https://example.feishu.cn/?table=tblX",
			wantErr: true,
		},
		{
			name:      "no query string",
			url:       "https://example.feishu.cn/wiki/NodeOnly",
			wantType:  "wiki",
			wantToken: "NodeOnly",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			urlType, token, table, view, err := parseFeishuURL(tc.url)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if urlType != tc.wantType {
				t.Errorf("urlType = %q, want %q", urlType, tc.wantType)
			}
			if token != tc.wantToken {
				t.Errorf("token = %q, want %q", token, tc.wantToken)
			}
			if table != tc.wantTable {
				t.Errorf("table = %q, want %q", table, tc.wantTable)
			}
			if view != tc.wantView {
				t.Errorf("view = %q, want %q", view, tc.wantView)
			}
		})
	}
}

// ---------- fieldStr ----------

func TestFieldStr(t *testing.T) {
	cases := []struct {
		name  string
		input any
		want  string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"integer float", float64(42), "42"},
		{"decimal float", float64(3.14), "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{
			"rich text segments",
			[]any{
				map[string]any{"text": "foo"},
				map[string]any{"text": "bar"},
			},
			"foobar",
		},
		{
			"rich text segments missing text key ignored",
			[]any{
				map[string]any{"text": "hello"},
				map[string]any{"type": "mention"},
			},
			"hello",
		},
		{"json fallback", map[string]any{"a": 1}, `{"a":1}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := fieldStr(tc.input)
			if got != tc.want {
				t.Errorf("fieldStr(%#v) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------- writeDotenv ----------



// ---------- withRetry ----------

func TestWithRetry(t *testing.T) {
	t.Run("succeeds on first attempt", func(t *testing.T) {
		calls := 0
		err := withRetry("op", func() error {
			calls++
			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if calls != 1 {
			t.Errorf("calls = %d, want 1", calls)
		}
	})

	t.Run("succeeds on second attempt", func(t *testing.T) {
		calls := 0
		err := withRetry("op", func() error {
			calls++
			if calls < 2 {
				return errors.New("transient")
			}
			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if calls != 2 {
			t.Errorf("calls = %d, want 2", calls)
		}
	})

	t.Run("fails after all retries", func(t *testing.T) {
		calls := 0
		err := withRetry("op", func() error {
			calls++
			return errors.New("always fails")
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if calls != maxRetries {
			t.Errorf("calls = %d, want %d", calls, maxRetries)
		}
	})
}

// ---------- getTenantAccessToken (httptest) ----------

func TestGetTenantAccessToken(t *testing.T) {
	orig := feishuAPI
	defer func() { feishuAPI = orig }()

	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]any{
				"code":               0,
				"msg":                "ok",
				"tenant_access_token": "tok123",
			})
		}))
		defer srv.Close()
		feishuAPI = srv.URL

		tok, err := getTenantAccessToken("id", "secret")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tok != "tok123" {
			t.Errorf("token = %q, want %q", tok, "tok123")
		}
	})

	t.Run("api error code", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]any{
				"code": 99991663,
				"msg":  "invalid app",
			})
		}))
		defer srv.Close()
		feishuAPI = srv.URL

		_, err := getTenantAccessToken("bad", "bad")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("http 500 triggers retry then fails", func(t *testing.T) {
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		feishuAPI = srv.URL

		_, err := getTenantAccessToken("id", "secret")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if calls != maxRetries {
			t.Errorf("calls = %d, want %d (one per retry)", calls, maxRetries)
		}
	})
}

// ---------- resolveWikiAppToken (httptest) ----------

func TestResolveWikiAppToken(t *testing.T) {
	orig := feishuAPI
	defer func() { feishuAPI = orig }()

	t.Run("success bitable node", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"node": map[string]any{
						"obj_type":  "bitable",
						"obj_token": "appTOKEN",
					},
				},
			})
		}))
		defer srv.Close()
		feishuAPI = srv.URL

		tok, err := resolveWikiAppToken("bearer", "wikiNode")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tok != "appTOKEN" {
			t.Errorf("token = %q, want appTOKEN", tok)
		}
	})

	t.Run("non-bitable node returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"node": map[string]any{
						"obj_type":  "doc",
						"obj_token": "docTOKEN",
					},
				},
			})
		}))
		defer srv.Close()
		feishuAPI = srv.URL

		_, err := resolveWikiAppToken("bearer", "wikiNode")
		if err == nil {
			t.Fatal("expected error for non-bitable node")
		}
	})
}

// ---------- output formatters ----------

func records(kvPairs ...string) []map[string]any {
	var out []map[string]any
	for i := 0; i+1 < len(kvPairs); i += 2 {
		out = append(out, map[string]any{"key": kvPairs[i], "value": kvPairs[i+1]})
	}
	return out
}

func TestWriteExport(t *testing.T) {
	cases := []struct {
		name    string
		records []map[string]any
		want    string
	}{
		{
			name:    "single record",
			records: records("FOO", "bar"),
			want:    "export FOO=$'bar'\n",
		},
		{
			name:    "value with spaces and special chars",
			records: records("DB_URL", "postgres://u:p@host/db"),
			want:    "export DB_URL=$'postgres://u:p@host/db'\n",
		},
		{
			name:    "empty key skipped",
			records: records("", "ignored", "K", "v"),
			want:    "export K=$'v'\n",
		},
		{
			name:    "multiple records",
			records: records("A", "1", "B", "2"),
			want:    "export A=$'1'\nexport B=$'2'\n",
		},
		{
			name:    "empty records",
			records: nil,
			want:    "",
		},
		{
			name:    "newline in value",
			records: records("K", "line1\nline2"),
			want:    "export K=$'line1\\nline2'\n",
		},
		{
			name:    "single quote escaped",
			records: records("K", "it's"),
			want:    `export K=$'it\'s'` + "\n",
		},
		{
			name:    "backslash escaped",
			records: records("K", `a\b`),
			want:    `export K=$'a\\b'` + "\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf strings.Builder
			writeExport(&buf, tc.records, "key", "value")
			if got := buf.String(); got != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

func TestWriteDotenv(t *testing.T) {
	cases := []struct {
		name    string
		records []map[string]any
		want    string
		wantErr bool
	}{
		{
			name:    "plain value",
			records: records("FOO", "bar"),
			want:    "FOO=\"bar\"\n",
		},
		{
			name:    "backslash and quote escaped",
			records: records("K", `a\"b`),
			want:    "K=\"a\\\\\\\"b\"\n",
		},
		{
			name:    "empty key skipped",
			records: records("", "skip", "K", "v"),
			want:    "K=\"v\"\n",
		},
		{
			name:    "multiple records",
			records: records("A", "1", "B", "2"),
			want:    "A=\"1\"\nB=\"2\"\n",
		},
		{
			name:    "newline in value supported",
			records: records("K", "line1\nline2"),
			want:    "K=\"line1\\nline2\"\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf strings.Builder
			err := writeDotenv(&buf, tc.records, "key", "value")
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := buf.String(); got != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	t.Run("basic output", func(t *testing.T) {
		var buf strings.Builder
		writeJSON(&buf, records("A", "1", "B", "2"), "key", "value")
		var m map[string]string
		if err := json.Unmarshal([]byte(buf.String()), &m); err != nil {
			t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
		}
		if m["A"] != "1" || m["B"] != "2" {
			t.Errorf("got %v", m)
		}
	})

	t.Run("empty key skipped", func(t *testing.T) {
		var buf strings.Builder
		writeJSON(&buf, records("", "ignored", "K", "v"), "key", "value")
		var m map[string]string
		if err := json.Unmarshal([]byte(buf.String()), &m); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if _, ok := m[""]; ok {
			t.Error("empty key should be excluded from JSON output")
		}
		if m["K"] != "v" {
			t.Errorf("K = %q, want v", m["K"])
		}
	})

	t.Run("empty records produces empty object", func(t *testing.T) {
		var buf strings.Builder
		writeJSON(&buf, nil, "key", "value")
		var m map[string]string
		if err := json.Unmarshal([]byte(buf.String()), &m); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if len(m) != 0 {
			t.Errorf("expected empty map, got %v", m)
		}
	})
}

func TestListRecords(t *testing.T) {
	orig := feishuAPI
	defer func() { feishuAPI = orig }()

	t.Run("single page", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"items": []any{
						map[string]any{"fields": map[string]any{"key": "K1", "value": "V1"}},
						map[string]any{"fields": map[string]any{"key": "K2", "value": "V2"}},
					},
					"has_more":   false,
					"page_token": "",
				},
			})
		}))
		defer srv.Close()
		feishuAPI = srv.URL

		items, hasMore, next, err := listRecords("tok", "app", "tbl", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 2 {
			t.Errorf("got %d items, want 2", len(items))
		}
		if hasMore {
			t.Error("hasMore should be false")
		}
		if next != "" {
			t.Errorf("nextToken = %q, want empty", next)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		page := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			page++
			hasMore := page == 1
			nextTok := ""
			if hasMore {
				nextTok = "page2token"
			}
			json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"items": []any{
						map[string]any{"fields": map[string]any{"key": fmt.Sprintf("K%d", page)}},
					},
					"has_more":   hasMore,
					"page_token": nextTok,
				},
			})
		}))
		defer srv.Close()
		feishuAPI = srv.URL

		all, err := fetchAllRecords("tok", "app", "tbl", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(all) != 2 {
			t.Errorf("got %d records, want 2", len(all))
		}
	})
}
