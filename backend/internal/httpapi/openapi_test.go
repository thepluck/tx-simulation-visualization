package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tx-simulation-visualization/backend/internal/config"
)

func TestOpenAPIEndpoint(t *testing.T) {
	server := NewServer(testConfig(t), "")
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var spec map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatalf("missing paths in spec: %#v", spec)
	}
	if _, ok := paths["/simulate"]; !ok {
		t.Fatalf("missing /simulate path: %#v", paths)
	}
	components, ok := spec["components"].(map[string]any)
	if !ok {
		t.Fatalf("missing components in spec: %#v", spec)
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		t.Fatalf("missing schemas in spec: %#v", components)
	}
	if _, ok := schemas["CompilerConfig"]; !ok {
		t.Fatalf("missing CompilerConfig schema: %#v", schemas)
	}
}

func TestSwaggerUIEndpoint(t *testing.T) {
	server := NewServer(testConfig(t), "")
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "SwaggerUIBundle") || !strings.Contains(body, "/openapi.json") {
		t.Fatalf("unexpected docs body: %s", body)
	}
}

func testConfig(t *testing.T) config.Config {
	t.Helper()

	return config.Config{
		ListenAddr:     "127.0.0.1:0",
		RepoRoot:       t.TempDir(),
		WorkDir:        t.TempDir(),
		TimeoutSeconds: 1,
		MaxConcurrent:  1,
		ForgeBin:       "forge",
		RPCURLs: map[string]string{
			"mainnet": "http://127.0.0.1:8545",
		},
	}
}
