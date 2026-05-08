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
	if cfg.ProjectCachePath != filepath.Join(workDir, "projects.json") {
		t.Fatalf("project cache path = %q, want default under work dir", cfg.ProjectCachePath)
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

func TestLoadAllowsEnvToOverrideConfigValues(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")
	workDir := filepath.Join(configDir, "env-runs")

	t.Setenv("TXSIM_CONFIG", configPath)
	t.Setenv("TXSIM_LISTEN_ADDR", "127.0.0.1:9090")
	t.Setenv("TXSIM_WORK_DIR", workDir)
	t.Setenv("TXSIM_TIMEOUT_SECONDS", "9")
	t.Setenv("TXSIM_MAX_CONCURRENT_RUNS", "3")
	t.Setenv("TXSIM_FORGE_BIN", "forge-env")
	t.Setenv("MAINNET_RPC_URL", "https://rpc.env")
	t.Setenv("MAINNET_EXPLORER_URL", "https://explorer.env/")

	data := []byte(`listen_addr: "127.0.0.1:8080"
repo_root: "."
work_dir: "config-runs"
timeout_seconds: 1
max_concurrent_runs: 1
forge_bin: "forge-config"
rpc_urls:
  mainnet: "https://rpc.config"
explorer_urls:
  mainnet: "https://explorer.config"
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
	if cfg.WorkDir != workDir {
		t.Fatalf("work dir = %q, want %q", cfg.WorkDir, workDir)
	}
	if cfg.TimeoutSeconds != 9 {
		t.Fatalf("timeout seconds = %d, want env override", cfg.TimeoutSeconds)
	}
	if cfg.MaxConcurrent != 3 {
		t.Fatalf("max concurrent = %d, want env override", cfg.MaxConcurrent)
	}
	if cfg.ForgeBin != "forge-env" {
		t.Fatalf("forge bin = %q, want env override", cfg.ForgeBin)
	}
	if cfg.RPCURLs["mainnet"] != "https://rpc.env" {
		t.Fatalf("mainnet rpc = %q, want env override", cfg.RPCURLs["mainnet"])
	}
	if cfg.ExplorerURLs["mainnet"] != "https://explorer.env" {
		t.Fatalf("mainnet explorer = %q, want env override", cfg.ExplorerURLs["mainnet"])
	}
}

func TestLoadReadsDotEnvWithGotenv(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")

	t.Setenv("TXSIM_CONFIG", configPath)

	data := []byte(`repo_root: "."
rpc_urls:
  mainnet: "${MAINNET_RPC_URL}"
explorer_urls:
  mainnet: "${MAINNET_EXPLORER_URL}"
`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	dotenv := []byte(`MAINNET_RPC_URL="https://rpc.dotenv"
MAINNET_EXPLORER_URL=https://explorer.dotenv/
`)
	if err := os.WriteFile(filepath.Join(configDir, ".env"), dotenv, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RPCURLs["mainnet"] != "https://rpc.dotenv" {
		t.Fatalf("mainnet rpc = %q, want dotenv value", cfg.RPCURLs["mainnet"])
	}
	if cfg.ExplorerURLs["mainnet"] != "https://explorer.dotenv" {
		t.Fatalf("mainnet explorer = %q, want dotenv value", cfg.ExplorerURLs["mainnet"])
	}
}

func TestLoadExistingEnvOverridesDotEnv(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")

	t.Setenv("TXSIM_CONFIG", configPath)
	t.Setenv("MAINNET_RPC_URL", "https://rpc.env")

	data := []byte(`repo_root: "."
rpc_urls:
  mainnet: "${MAINNET_RPC_URL}"
`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, ".env"), []byte(`MAINNET_RPC_URL=https://rpc.dotenv`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RPCURLs["mainnet"] != "https://rpc.env" {
		t.Fatalf("mainnet rpc = %q, want existing env override", cfg.RPCURLs["mainnet"])
	}
}

func TestLoadUsesViperDefaults(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")

	t.Setenv("TXSIM_CONFIG", configPath)

	data := []byte(`rpc_urls:
  mainnet: "https://rpc.example"
`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != "127.0.0.1:8080" {
		t.Fatalf("listen addr = %q, want viper default", cfg.ListenAddr)
	}
	if cfg.RepoRoot != filepath.Join(configDir, "..") {
		t.Fatalf("repo root = %q, want viper default normalized", cfg.RepoRoot)
	}
	if cfg.WorkDir != filepath.Join(configDir, ".runs") {
		t.Fatalf("work dir = %q, want viper default normalized", cfg.WorkDir)
	}
	if cfg.ProjectCachePath != filepath.Join(configDir, ".runs", "projects.json") {
		t.Fatalf("project cache path = %q, want default under work dir", cfg.ProjectCachePath)
	}
	if cfg.TimeoutSeconds != 120 {
		t.Fatalf("timeout seconds = %d, want viper default", cfg.TimeoutSeconds)
	}
	if cfg.MaxConcurrent != 1 {
		t.Fatalf("max concurrent = %d, want viper default", cfg.MaxConcurrent)
	}
	if cfg.ForgeBin != "forge" {
		t.Fatalf("forge bin = %q, want viper default", cfg.ForgeBin)
	}
}

func TestLoadExpandsHomeInProjectRoots(t *testing.T) {
	configDir := t.TempDir()
	homeDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")

	t.Setenv("HOME", homeDir)
	t.Setenv("TXSIM_CONFIG", configPath)

	data := []byte(`repo_root: "."
project_roots:
  - "~/projects"
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
	want := filepath.Join(homeDir, "projects")
	if len(cfg.ProjectRoots) != 1 || cfg.ProjectRoots[0] != want {
		t.Fatalf("project roots = %#v, want %q", cfg.ProjectRoots, want)
	}
}

func TestLoadNormalizesProjectCachePath(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")

	t.Setenv("TXSIM_CONFIG", configPath)

	data := []byte(`repo_root: "."
project_cache_path: ".cache/projects.json"
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
	want := filepath.Join(configDir, ".cache", "projects.json")
	if cfg.ProjectCachePath != want {
		t.Fatalf("project cache path = %q, want %q", cfg.ProjectCachePath, want)
	}
}
