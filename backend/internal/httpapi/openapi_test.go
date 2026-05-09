package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"foundry-tx-simulator/backend/internal/config"
)

func TestOpenAPIEndpoint(t *testing.T) {
	server := NewServer(testConfig(t), "")
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
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
	if _, ok := paths["/projects"]; !ok {
		t.Fatalf("missing /projects path: %#v", paths)
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
	simulateRequest, ok := schemas["SimulateRequest"].(map[string]any)
	if !ok {
		t.Fatalf("missing SimulateRequest schema: %#v", schemas)
	}
	properties, ok := simulateRequest["properties"].(map[string]any)
	if !ok {
		t.Fatalf("missing SimulateRequest properties: %#v", simulateRequest)
	}
	if _, ok := properties["etherscanApiKey"]; ok {
		t.Fatalf("etherscanApiKey should be backend config, not a request property: %#v", properties)
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

func TestCORSOptionsEndpoint(t *testing.T) {
	server := NewServer(testConfig(t), "")
	req := httptest.NewRequest(http.MethodOptions, "/simulate", nil)
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

func TestChainsEndpointIncludesExplorerURLs(t *testing.T) {
	server := NewServer(testConfig(t), "")
	req := httptest.NewRequest(http.MethodGet, "/chains", nil)
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var payload struct {
		Chains       []string          `json:"chains"`
		ExplorerURLs map[string]string `json:"explorerUrls"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Chains) != 1 || payload.Chains[0] != "mainnet" {
		t.Fatalf("unexpected chains: %#v", payload.Chains)
	}
	if payload.ExplorerURLs["mainnet"] != "https://etherscan.io" {
		t.Fatalf("unexpected explorer URLs: %#v", payload.ExplorerURLs)
	}
}

func TestBrowseProjectEndpoint(t *testing.T) {
	server := NewServer(testConfig(t), "")
	server.chooseProjectDir = func(context.Context) (string, error) {
		return "/tmp/foundry-project", nil
	}
	req := httptest.NewRequest(http.MethodGet, "/browse/project", nil)
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var payload struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Path != "/tmp/foundry-project" {
		t.Fatalf("path = %q", payload.Path)
	}

	projects := readProjects(t, server)
	if len(projects) != 1 || projects[0] != "/tmp/foundry-project" {
		t.Fatalf("cached projects = %#v", projects)
	}
}

func TestProjectsEndpoint(t *testing.T) {
	server := NewServer(testConfig(t), "")
	server.rememberProjectPath("~/alpha")
	server.rememberProjectPath("~/beta")
	server.rememberProjectPath("~/alpha")

	projects := readProjects(t, server)
	want := []string{"~/alpha", "~/beta"}
	if len(projects) != len(want) {
		t.Fatalf("projects = %#v, want %#v", projects, want)
	}
	for i := range want {
		if projects[i] != want[i] {
			t.Fatalf("projects = %#v, want %#v", projects, want)
		}
	}
}

func TestDebugHTTPLogsRequestAndResponse(t *testing.T) {
	t.Setenv("TXSIM_DEBUG_HTTP", "1")
	var logs bytes.Buffer
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(oldLogger)
	})

	server := NewServer(testConfig(t), "")
	req := httptest.NewRequest(http.MethodPost, "/simulate", strings.NewReader(`{"bad":true,"etherscanApiKey":"secret-key"}`))
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	output := logs.String()
	for _, want := range []string{
		`msg="http request"`,
		`method=POST`,
		`path=/simulate`,
		`etherscanApiKey`,
		`<redacted>`,
		`msg="http response"`,
		`status=400`,
		`invalid JSON body`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected log %q in:\n%s", want, output)
		}
	}
	if strings.Contains(output, "secret-key") {
		t.Fatalf("debug logs should redact etherscan API key:\n%s", output)
	}
}

func testConfig(t *testing.T) config.Config {
	t.Helper()

	return config.Config{
		ListenAddr:       "127.0.0.1:0",
		RepoRoot:         t.TempDir(),
		WorkDir:          t.TempDir(),
		ProjectCachePath: filepath.Join(t.TempDir(), "projects.json"),
		TimeoutSeconds:   1,
		MaxConcurrent:    1,
		ForgeBin:         "forge",
		RPCURLs: map[string]string{
			"mainnet": "http://127.0.0.1:8545",
		},
		ExplorerURLs: map[string]string{
			"mainnet": "https://etherscan.io",
		},
	}
}

func readProjects(t *testing.T, server *Server) []string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var payload struct {
		Projects []string `json:"projects"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Projects
}
