package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"tx-simulation-visualization/backend/internal/config"
	"tx-simulation-visualization/backend/internal/httpapi"
)

func main() {
	cfg, configPath, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	server := httpapi.NewServer(cfg, configPath)
	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("tx simulation backend listening", "url", "http://"+cfg.ListenAddr)
	slog.Info("config loaded", "path", configPath)
	if err := httpServer.ListenAndServe(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
