package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"

	"tx-simulation-visualization/backend/internal/config"
	"tx-simulation-visualization/backend/internal/model"
	"tx-simulation-visualization/backend/internal/simulation"
)

type Server struct {
	cfg        config.Config
	configPath string
	simulator  *simulation.Service
}

func NewServer(cfg config.Config, configPath string) *Server {
	return &Server{
		cfg:        cfg,
		configPath: configPath,
		simulator:  simulation.NewService(cfg),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /docs", s.handleSwaggerUI)
	mux.HandleFunc("GET /docs/", s.handleSwaggerUI)
	mux.HandleFunc("GET /openapi.json", s.handleOpenAPI)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /chains", s.handleChains)
	mux.HandleFunc("POST /simulate", s.handleSimulate)
	return localCORS(mux)
}

func localCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                true,
		"chains":            len(s.cfg.RPCURLs),
		"maxConcurrentRuns": s.cfg.MaxConcurrent,
	})
}

func (s *Server) handleChains(w http.ResponseWriter, _ *http.Request) {
	chains := make([]string, 0, len(s.cfg.RPCURLs))
	for chain := range s.cfg.RPCURLs {
		chains = append(chains, chain)
	}
	sort.Strings(chains)
	writeJSON(w, http.StatusOK, map[string]any{"chains": chains})
}

func (s *Server) handleSimulate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()

	var req model.SimulateRequest
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "invalid JSON body: " + err.Error()})
		return
	}

	resp, status := s.simulator.Simulate(r.Context(), req)
	writeJSON(w, status, resp)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}
