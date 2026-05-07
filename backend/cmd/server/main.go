package main

import (
	"log"
	"net/http"
	"time"

	"tx-simulation-visualization/backend/internal/config"
	"tx-simulation-visualization/backend/internal/httpapi"
)

func main() {
	cfg, configPath, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	server := httpapi.NewServer(cfg, configPath)
	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("tx simulation backend listening on http://%s", cfg.ListenAddr)
	log.Printf("config: %s", configPath)
	log.Fatal(httpServer.ListenAndServe())
}
