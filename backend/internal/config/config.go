package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

type Config struct {
	ListenAddr       string            `mapstructure:"listen_addr" yaml:"listen_addr"`
	RepoRoot         string            `mapstructure:"repo_root" yaml:"repo_root"`
	ProjectRoots     []string          `mapstructure:"project_roots" yaml:"project_roots"`
	WorkDir          string            `mapstructure:"work_dir" yaml:"work_dir"`
	ProjectCachePath string            `mapstructure:"project_cache_path" yaml:"project_cache_path"`
	TimeoutSeconds   int               `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
	MaxConcurrent    int               `mapstructure:"max_concurrent_runs" yaml:"max_concurrent_runs"`
	ForgeBin         string            `mapstructure:"forge_bin" yaml:"forge_bin"`
	RPCURLs          map[string]string `mapstructure:"rpc_urls" yaml:"rpc_urls"`
	ExplorerURLs     map[string]string `mapstructure:"explorer_urls" yaml:"explorer_urls"`
}

var configCandidates = []string{
	"config.yaml",
	"backend/config.yaml",
	"config.yml",
	"backend/config.yml",
	"config.example.yaml",
	"backend/config.example.yaml",
	"config.example.yml",
	"backend/config.example.yml",
}

func Load() (Config, string, error) {
	path, err := resolveConfigPath()
	if err != nil {
		return Config{}, "", err
	}

	v := newConfigViper(path)
	if err := v.ReadInConfig(); err != nil {
		return Config{}, "", err
	}

	configPath, err := filepath.Abs(v.ConfigFileUsed())
	if err != nil {
		return Config{}, "", err
	}
	configDir := filepath.Dir(configPath)
	if err := loadDotEnvForConfig(v, configDir); err != nil {
		return Config{}, "", err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, "", err
	}
	cfg.ListenAddr = strings.TrimSpace(cfg.ListenAddr)
	cfg.RepoRoot, err = normalizeConfigPath(configDir, cfg.RepoRoot)
	if err != nil {
		return Config{}, "", err
	}
	cfg.ProjectRoots, err = normalizeConfigPaths(configDir, cfg.ProjectRoots)
	if err != nil {
		return Config{}, "", err
	}

	if !filepath.IsAbs(cfg.WorkDir) {
		cfg.WorkDir = filepath.Join(configDir, cfg.WorkDir)
	}
	cfg.WorkDir, err = filepath.Abs(cfg.WorkDir)
	if err != nil {
		return Config{}, "", err
	}
	if cfg.ProjectCachePath == "" {
		cfg.ProjectCachePath = filepath.Join(cfg.WorkDir, "projects.json")
	} else {
		cfg.ProjectCachePath, err = normalizeConfigPath(configDir, cfg.ProjectCachePath)
		if err != nil {
			return Config{}, "", err
		}
	}

	if cfg.TimeoutSeconds < 0 {
		return Config{}, "", errors.New("timeout_seconds must be positive")
	}
	if cfg.MaxConcurrent < 0 {
		return Config{}, "", errors.New("max_concurrent_runs must be positive")
	}
	if len(cfg.RPCURLs) == 0 {
		return Config{}, "", errors.New("rpc_urls must contain at least one chain")
	}
	cfg.RPCURLs = resolveRPCURLs(cfg.RPCURLs)
	if cfg.ExplorerURLs == nil {
		cfg.ExplorerURLs = map[string]string{}
	}
	cfg.ExplorerURLs = resolveExplorerURLs(cfg.ExplorerURLs)

	return cfg, configPath, nil
}

func resolveConfigPath() (string, error) {
	env := viper.New()
	if err := env.BindEnv("config", "TXSIM_CONFIG"); err != nil {
		return "", err
	}
	if path := strings.TrimSpace(env.GetString("config")); path != "" {
		return path, nil
	}

	for _, candidate := range configCandidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", errors.New("set TXSIM_CONFIG or create backend/config.yaml")
}

func newConfigViper(path string) *viper.Viper {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetDefault("listen_addr", "127.0.0.1:8080")
	v.SetDefault("repo_root", "..")
	v.SetDefault("work_dir", ".runs")
	v.SetDefault("timeout_seconds", 120)
	v.SetDefault("max_concurrent_runs", 1)
	v.SetDefault("forge_bin", "forge")
	v.SetEnvPrefix("TXSIM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	for _, key := range []string{
		"listen_addr",
		"repo_root",
		"project_roots",
		"work_dir",
		"project_cache_path",
		"timeout_seconds",
		"max_concurrent_runs",
		"forge_bin",
	} {
		_ = v.BindEnv(key)
	}
	return v
}

func loadDotEnvForConfig(v *viper.Viper, configDir string) error {
	repoRoot, err := normalizeConfigPath(configDir, v.GetString("repo_root"))
	if err != nil {
		return err
	}
	return loadDotEnv(filepath.Join(repoRoot, ".env"), filepath.Join(configDir, ".env"))
}

func normalizeConfigPaths(baseDir string, values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized, err := normalizeConfigPath(baseDir, os.ExpandEnv(value))
		if err != nil {
			return nil, err
		}
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out, nil
}

func normalizeConfigPath(baseDir string, value string) (string, error) {
	value = strings.TrimSpace(os.ExpandEnv(value))
	if value == "" {
		return "", nil
	}
	expanded, err := expandHomePath(value)
	if err != nil {
		return "", err
	}
	value = expanded
	if !filepath.IsAbs(value) {
		value = filepath.Join(baseDir, value)
	}
	return filepath.Abs(value)
}

func expandHomePath(value string) (string, error) {
	if value != "~" && !strings.HasPrefix(value, "~/") {
		return value, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if value == "~" {
		return homeDir, nil
	}
	return filepath.Join(homeDir, strings.TrimPrefix(value, "~/")), nil
}

func loadDotEnv(paths ...string) error {
	existing := make([]string, 0, len(paths))
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

		if _, err := os.Stat(absPath); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		existing = append(existing, absPath)
	}
	if len(existing) == 0 {
		return nil
	}
	return gotenv.Load(existing...)
}

func resolveRPCURLs(values map[string]string) map[string]string {
	resolved := make(map[string]string, len(values))
	for chain, rpcURL := range values {
		if envValue := strings.TrimSpace(os.Getenv(chainEnvName(chain, "RPC_URL"))); envValue != "" {
			rpcURL = envValue
		}
		resolved[chain] = os.ExpandEnv(rpcURL)
	}
	return resolved
}

func resolveExplorerURLs(values map[string]string) map[string]string {
	resolved := make(map[string]string, len(values))
	for chain, explorerURL := range values {
		if envValue := strings.TrimSpace(os.Getenv(chainEnvName(chain, "EXPLORER_URL"))); envValue != "" {
			explorerURL = envValue
		}
		resolved[chain] = strings.TrimRight(os.ExpandEnv(explorerURL), "/")
	}
	return resolved
}

func chainEnvName(chain string, suffix string) string {
	normalized := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r - ('a' - 'A')
		}
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, strings.TrimSpace(chain))
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return suffix
	}
	return normalized + "_" + suffix
}
