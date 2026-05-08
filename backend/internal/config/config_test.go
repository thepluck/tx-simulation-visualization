package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExpandsAndNormalizesExplorerURLs(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")
	workDir := filepath.Join(configDir, "runs")

	t.Setenv("TXSIM_CONFIG", configPath)
	t.Setenv("MAINNET_RPC_URL", "https://rpc.example")
	t.Setenv("MAINNET_EXPLORER_URL", "https://explorer.example/")
	t.Setenv("PROJECT_ROOT", filepath.Join(configDir, "projects"))

	data := []byte(`listen_addr: "127.0.0.1:0"
repo_root: "."
project_roots:
  - "${PROJECT_ROOT}"
work_dir: "` + filepath.ToSlash(workDir) + `"
timeout_seconds: 1
max_concurrent_runs: 1
forge_bin: "forge"
rpc_urls:
  mainnet: "${MAINNET_RPC_URL}"
explorer_urls:
  mainnet: "${MAINNET_EXPLORER_URL}"
`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, gotPath, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != configPath {
		t.Fatalf("config path = %q, want %q", gotPath, configPath)
	}
	if cfg.RPCURLs["mainnet"] != "https://rpc.example" {
		t.Fatalf("mainnet rpc = %q", cfg.RPCURLs["mainnet"])
	}
	if cfg.ExplorerURLs["mainnet"] != "https://explorer.example" {
		t.Fatalf("mainnet explorer = %q", cfg.ExplorerURLs["mainnet"])
	}
	if len(cfg.ProjectRoots) != 1 || cfg.ProjectRoots[0] != filepath.Join(configDir, "projects") {
		t.Fatalf("project roots = %#v", cfg.ProjectRoots)
	}
}

func TestLoadAllowsListenAddressEnvOverride(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")

	t.Setenv("TXSIM_CONFIG", configPath)
	t.Setenv("TXSIM_LISTEN_ADDR", "127.0.0.1:9090")

	data := []byte(`listen_addr: "127.0.0.1:8080"
repo_root: "."
rpc_urls:
  mainnet: "https://rpc.example"
`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != "127.0.0.1:9090" {
		t.Fatalf("listen addr = %q, want env override", cfg.ListenAddr)
	}
}
