package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	ListenAddr     string            `json:"listen_addr"`
	RepoRoot       string            `json:"repo_root"`
	WorkDir        string            `json:"work_dir"`
	TimeoutSeconds int               `json:"timeout_seconds"`
	MaxConcurrent  int               `json:"max_concurrent_runs"`
	ForgeBin       string            `json:"forge_bin"`
	RPCURLs        map[string]string `json:"rpc_urls"`
}

func Load() (Config, string, error) {
	path := os.Getenv("TXSIM_CONFIG")
	if path == "" {
		for _, candidate := range []string{
			"config.json",
			"backend/config.json",
			"config.example.json",
			"backend/config.example.json",
		} {
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
				break
			}
		}
	}
	if path == "" {
		return Config{}, "", errors.New("set TXSIM_CONFIG or create backend/config.json")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, "", err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, "", err
	}

	configPath, err := filepath.Abs(path)
	if err != nil {
		return Config{}, "", err
	}
	configDir := filepath.Dir(configPath)

	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:8080"
	}
	if cfg.RepoRoot == "" {
		cfg.RepoRoot = ".."
	}
	if !filepath.IsAbs(cfg.RepoRoot) {
		cfg.RepoRoot = filepath.Join(configDir, cfg.RepoRoot)
	}
	cfg.RepoRoot, err = filepath.Abs(cfg.RepoRoot)
	if err != nil {
		return Config{}, "", err
	}
	if err := loadDotEnv(filepath.Join(cfg.RepoRoot, ".env"), filepath.Join(configDir, ".env")); err != nil {
		return Config{}, "", err
	}

	if cfg.WorkDir == "" {
		cfg.WorkDir = ".runs"
	}
	if !filepath.IsAbs(cfg.WorkDir) {
		cfg.WorkDir = filepath.Join(configDir, cfg.WorkDir)
	}
	cfg.WorkDir, err = filepath.Abs(cfg.WorkDir)
	if err != nil {
		return Config{}, "", err
	}

	if cfg.TimeoutSeconds == 0 {
		cfg.TimeoutSeconds = 120
	}
	if cfg.TimeoutSeconds < 0 {
		return Config{}, "", errors.New("timeout_seconds must be positive")
	}
	if cfg.MaxConcurrent < 0 {
		return Config{}, "", errors.New("max_concurrent_runs must be positive")
	}
	if cfg.MaxConcurrent == 0 {
		cfg.MaxConcurrent = 1
	}
	if cfg.ForgeBin == "" {
		cfg.ForgeBin = "forge"
	}
	if len(cfg.RPCURLs) == 0 {
		return Config{}, "", errors.New("rpc_urls must contain at least one chain")
	}
	for chain, rpcURL := range cfg.RPCURLs {
		cfg.RPCURLs[chain] = os.ExpandEnv(rpcURL)
	}

	return cfg, configPath, nil
}

func loadDotEnv(paths ...string) error {
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if _, ok := seen[absPath]; ok {
			continue
		}
		seen[absPath] = struct{}{}

		if err := loadDotEnvFile(absPath); err != nil {
			return err
		}
	}
	return nil
}

func loadDotEnvFile(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s: invalid .env line %d", path, lineNumber)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return errors.New(path + ": empty .env key")
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		os.Setenv(key, trimEnvValue(value))
	}
	return scanner.Err()
}

func trimEnvValue(value string) string {
	if len(value) < 2 {
		return value
	}
	quote := value[0]
	if (quote == '"' || quote == '\'') && value[len(value)-1] == quote {
		return value[1 : len(value)-1]
	}
	return value
}
