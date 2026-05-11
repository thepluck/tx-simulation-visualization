package simulation

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"foundry-tx-simulator/backend/internal/model"
)

const (
	requestRecordFile  = "request.json"
	responseRecordFile = "response.json"
)

var (
	ErrInvalidRecordID = errors.New("invalid request id")
	ErrRecordNotFound  = errors.New("request record not found")
)

func (s *Service) LoadRecord(id string) (model.SimulationRecord, error) {
	id = strings.TrimSpace(id)
	recordDir, err := recordDirectory(s.cfg.WorkDir, id)
	if err != nil {
		return model.SimulationRecord{}, err
	}

	var req model.SimulateRequest
	if err := readRecordJSON(filepath.Join(recordDir, requestRecordFile), &req); err != nil {
		return model.SimulationRecord{}, err
	}

	var resp model.SimulateResponse
	if err := readRecordJSON(filepath.Join(recordDir, responseRecordFile), &resp); err != nil {
		return model.SimulationRecord{}, err
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

func writeSimulationRecord(runDir string, req model.SimulateRequest, resp model.SimulateResponse) error {
	if strings.TrimSpace(runDir) == "" {
		return nil
	}
	if err := writeRecordJSON(filepath.Join(runDir, requestRecordFile), req); err != nil {
		return err
	}
	if err := writeRecordJSON(filepath.Join(runDir, responseRecordFile), resp); err != nil {
		return err
	}
	return nil
}

func recordDirectory(workDir string, id string) (string, error) {
	if id == "" || id == "." || id == ".." || filepath.IsAbs(id) || filepath.Clean(id) != id || strings.ContainsAny(id, `/\`) {
		return "", ErrInvalidRecordID
	}
	if strings.TrimSpace(workDir) == "" {
		return "", fmt.Errorf("work dir is empty")
	}
	return filepath.Join(workDir, id), nil
}

func readRecordJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrRecordNotFound
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", filepath.Base(path), err)
	}
	return nil
}

func writeRecordJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
