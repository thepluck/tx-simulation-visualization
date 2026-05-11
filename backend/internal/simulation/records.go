package simulation

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"foundry-tx-simulator/backend/internal/model"
)

const recordDatabaseFile = "records.sqlite"

var (
	ErrInvalidRecordID = errors.New("invalid request id")
	ErrRecordNotFound  = errors.New("request record not found")
)

func (s *Service) LoadRecord(id string) (model.SimulationRecord, error) {
	id = strings.TrimSpace(id)
	if err := validateRecordID(id); err != nil {
		return model.SimulationRecord{}, err
	}

	db, err := openRecordDatabase(s.cfg.WorkDir, false)
	if err != nil {
		return model.SimulationRecord{}, err
	}
	defer closeRecordDatabase(db)
	if err := ensureRecordSchema(db); err != nil {
		return model.SimulationRecord{}, err
	}

	var requestJSON []byte
	var responseJSON []byte
	err = db.QueryRow(
		"SELECT request_json, response_json FROM simulation_records WHERE id = ?",
		id,
	).Scan(&requestJSON, &responseJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return model.SimulationRecord{}, ErrRecordNotFound
	}
	if err != nil {
		return model.SimulationRecord{}, err
	}

	var req model.SimulateRequest
	if err := json.Unmarshal(requestJSON, &req); err != nil {
		return model.SimulationRecord{}, fmt.Errorf("decode request record: %w", err)
	}

	var resp model.SimulateResponse
	if err := json.Unmarshal(responseJSON, &resp); err != nil {
		return model.SimulationRecord{}, fmt.Errorf("decode response record: %w", err)
	}
	if resp.ID == "" {
		resp.ID = id
	}

	return model.SimulationRecord{
		ID:       id,
		Request:  req,
		Response: resp,
	}, nil
}

func (s *Service) SaveRecord(req model.SimulateRequest, resp model.SimulateResponse) error {
	id := strings.TrimSpace(resp.ID)
	if err := validateRecordID(id); err != nil {
		return err
	}

	requestJSON, err := json.Marshal(req)
	if err != nil {
		return err
	}
	responseJSON, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	db, err := openRecordDatabase(s.cfg.WorkDir, true)
	if err != nil {
		return err
	}
	defer closeRecordDatabase(db)

	if err := ensureRecordSchema(db); err != nil {
		return err
	}

	_, err = db.Exec(
		`INSERT INTO simulation_records (id, request_json, response_json, created_at, updated_at)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		 ON CONFLICT(id) DO UPDATE SET
		   request_json = excluded.request_json,
		   response_json = excluded.response_json,
		   updated_at = CURRENT_TIMESTAMP`,
		id,
		requestJSON,
		responseJSON,
	)
	return err
}

func openRecordDatabase(workDir string, create bool) (*sql.DB, error) {
	dbPath, err := recordDatabasePath(workDir)
	if err != nil {
		return nil, err
	}
	if create {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			return nil, err
		}
	} else if _, err := os.Stat(dbPath); errors.Is(err, os.ErrNotExist) {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func closeRecordDatabase(db *sql.DB) {
	if err := db.Close(); err != nil {
		slog.Warn("close record database", "error", err)
	}
}

func ensureRecordSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS simulation_records (
		id TEXT PRIMARY KEY,
		request_json BLOB NOT NULL,
		response_json BLOB NOT NULL,
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

func recordDatabasePath(workDir string) (string, error) {
	if strings.TrimSpace(workDir) == "" {
		return "", fmt.Errorf("work dir is empty")
	}
	return filepath.Join(workDir, recordDatabaseFile), nil
}

func validateRecordID(id string) error {
	if id == "" || id == "." || id == ".." || filepath.IsAbs(id) || filepath.Clean(id) != id || strings.ContainsAny(id, `/\`) {
		return ErrInvalidRecordID
	}
	return nil
}
