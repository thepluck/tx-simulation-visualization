package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadListenAddrUsesYAMLParser(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(configPath, []byte(`listen_addr: "127.0.0.1:18080" # local backend`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readListenAddr(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != "127.0.0.1:18080" {
		t.Fatalf("listen_addr = %q, want 127.0.0.1:18080", got)
	}
}

func TestWriteDevBackendConfigUsesYAMLParser(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(configPath, []byte(`listen_addr: "127.0.0.1:8080" # original
work_dir: ".runs"
rpc_urls:
  mainnet: "${MAINNET_RPC_URL}"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	devConfigPath, err := writeDevBackendConfig(configPath, "127.0.0.1:18080")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Remove(devConfigPath)
	})

	got, err := readListenAddr(devConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != "127.0.0.1:18080" {
		t.Fatalf("dev listen_addr = %q, want 127.0.0.1:18080", got)
	}
	config := newDevConfigViper(devConfigPath)
	if err := config.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	if got := config.GetString("work_dir"); got != ".runs" {
		t.Fatalf("work_dir = %q, want .runs", got)
	}
	rpcURLs := config.GetStringMapString("rpc_urls")
	if rpcURLs["mainnet"] != "${MAINNET_RPC_URL}" {
		t.Fatalf("mainnet rpc url = %#v", rpcURLs["mainnet"])
	}
}
