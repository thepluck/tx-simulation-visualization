package simulation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tx-simulation-visualization/backend/internal/config"
	"tx-simulation-visualization/backend/internal/forge"
	"tx-simulation-visualization/backend/internal/model"
	"tx-simulation-visualization/backend/internal/runid"
	"tx-simulation-visualization/backend/internal/solidity"
	"tx-simulation-visualization/backend/internal/traceparser"
)

type Service struct {
	cfg        config.Config
	forge      forge.Runner
	runLimiter chan struct{}
}

func NewService(cfg config.Config) *Service {
	return &Service{
		cfg: cfg,
		forge: forge.Runner{
			Bin:      cfg.ForgeBin,
			RepoRoot: cfg.RepoRoot,
		},
		runLimiter: make(chan struct{}, cfg.MaxConcurrent),
	}
}

func (s *Service) Simulate(parent context.Context, req model.SimulateRequest) (model.SimulateResponse, int) {
	start := time.Now()
	runID := runid.New()
	resp := model.SimulateResponse{ID: runID}

	rpcURL, err := s.validateRequest(&req)
	if err != nil {
		resp.Error = err.Error()
		return resp, http.StatusBadRequest
	}

	timeout := time.Duration(s.cfg.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	release, err := s.acquireRunSlot(ctx)
	if err != nil {
		resp.Error = "rate limit: timed out waiting for an available simulation slot"
		return resp, http.StatusTooManyRequests
	}
	defer release()

	runDir := filepath.Join(s.cfg.WorkDir, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		resp.Error = "create run directory: " + err.Error()
		return resp, http.StatusInternalServerError
	}
	resp.RunDir = runDir

	stateBytecode := "0x"
	source, contractName := req.StateOverrideSourceAndName()
	if strings.TrimSpace(source) != "" {
		if contractName == "" {
			contractName = solidity.DetectContractName(source)
			if contractName == "" {
				resp.Error = "stateOverride.contractName is required when the source contains no `contract Name` declaration"
				return resp, http.StatusBadRequest
			}
		}

		statePath := filepath.Join(runDir, "StateOverride.sol")
		if err := os.WriteFile(statePath, []byte(source), 0o644); err != nil {
			resp.Error = "write state override source: " + err.Error()
			return resp, http.StatusInternalServerError
		}

		bytecode, compileResult, err := s.compileStateOverride(ctx, statePath, contractName, req.Compiler)
		if err != nil {
			resp.DurationMillis = time.Since(start).Milliseconds()
			resp.Stdout = solidity.RedactRPC(compileResult.Stdout, rpcURL, req.Chain)
			resp.Stderr = solidity.RedactRPC(compileResult.Stderr, rpcURL, req.Chain)
			resp.Trace = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)
			resp.StructuredTrace = traceparser.Parse(resp.Trace)
			resp.ExitCode = compileResult.ExitCode
			resp.Error = "compile state override: " + err.Error()
			return resp, forge.StatusFromCommandError(compileResult.Err)
		}
		stateBytecode = bytecode
	}

	forgeArgs := []string{
		"script",
		"contracts/SimulateTx.s.sol:SimulateTxScript",
		"--sig",
		"run((address,string)[],(address,address,uint256)[],(address,address,address,uint256)[],(address,address,address,uint256)[],bytes,address,address,bytes)",
	}
	forgeArgs = append(forgeArgs, solidity.ForgeRunArgs(req, stateBytecode)...)
	forgeArgs = append(forgeArgs,
		"--root", s.cfg.RepoRoot,
		"--rpc-url", rpcURL,
		"--fork-block-number", req.BlockNumber.String(),
		"-vvvvv",
		"--color", "never",
		"--non-interactive",
	)
	forgeArgs = append(forgeArgs, solidity.ForgeCompilerArgs(req.Compiler)...)

	result := s.forge.Run(ctx, forgeArgs...)
	resp.DurationMillis = time.Since(start).Milliseconds()
	resp.Stdout = solidity.RedactRPC(solidity.StripANSI(result.Stdout), rpcURL, req.Chain)
	resp.Stderr = solidity.RedactRPC(solidity.StripANSI(result.Stderr), rpcURL, req.Chain)
	combined := strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)
	resp.Trace = solidity.ExtractTrace(combined)
	resp.StructuredTrace = traceparser.Parse(resp.Trace)
	resp.ExitCode = result.ExitCode
	resp.Success = result.Err == nil
	if result.Err != nil {
		resp.Error = result.Err.Error()
		return resp, forge.StatusFromCommandError(result.Err)
	}

	return resp, http.StatusOK
}

func (s *Service) validateRequest(req *model.SimulateRequest) (string, error) {
	req.Chain = strings.TrimSpace(req.Chain)
	if req.Chain == "" {
		return "", errors.New("chain is required")
	}

	rpcURL, ok := s.cfg.RPCURLs[req.Chain]
	if !ok {
		return "", fmt.Errorf("unknown chain %q", req.Chain)
	}
	if strings.TrimSpace(rpcURL) == "" {
		return "", fmt.Errorf("rpc url for chain %q is empty after environment expansion", req.Chain)
	}

	if req.BlockNumber == "" {
		return "", errors.New("blockNumber is required")
	}
	if err := solidity.ValidateAddress("sender", req.Sender); err != nil {
		return "", err
	}
	if err := solidity.ValidateAddress("target", req.Target); err != nil {
		return "", err
	}
	normalizedData, err := solidity.NormalizeBytes("data", req.Data)
	if err != nil {
		return "", err
	}
	req.Data = normalizedData

	for i := range req.LabelOverrides {
		item := &req.LabelOverrides[i]
		if err := solidity.ValidateAddress(fmt.Sprintf("labelOverrides[%d].account", i), item.Account); err != nil {
			return "", err
		}
		if strings.TrimSpace(item.Label) == "" {
			return "", fmt.Errorf("labelOverrides[%d].label is required", i)
		}
	}

	for i := range req.ERC20BalanceOverrides {
		item := &req.ERC20BalanceOverrides[i]
		if err := solidity.ValidateAddress(fmt.Sprintf("erc20BalanceOverrides[%d].token", i), item.Token); err != nil {
			return "", err
		}
		if err := solidity.ValidateAddress(fmt.Sprintf("erc20BalanceOverrides[%d].account", i), item.Account); err != nil {
			return "", err
		}
		if item.Balance == "" {
			return "", fmt.Errorf("erc20BalanceOverrides[%d].balance is required", i)
		}
	}

	for i := range req.ERC20ApprovalOverrides {
		item := &req.ERC20ApprovalOverrides[i]
		if err := solidity.ValidateAddress(fmt.Sprintf("erc20ApprovalOverrides[%d].token", i), item.Token); err != nil {
			return "", err
		}
		if err := solidity.ValidateAddress(fmt.Sprintf("erc20ApprovalOverrides[%d].owner", i), item.Owner); err != nil {
			return "", err
		}
		if err := solidity.ValidateAddress(fmt.Sprintf("erc20ApprovalOverrides[%d].spender", i), item.Spender); err != nil {
			return "", err
		}
		if item.Amount == "" {
			return "", fmt.Errorf("erc20ApprovalOverrides[%d].amount is required", i)
		}
	}

	for i := range req.ERC721ApprovalOverrides {
		item := &req.ERC721ApprovalOverrides[i]
		if err := solidity.ValidateAddress(fmt.Sprintf("erc721ApprovalOverrides[%d].token", i), item.Token); err != nil {
			return "", err
		}
		if err := solidity.ValidateAddress(fmt.Sprintf("erc721ApprovalOverrides[%d].owner", i), item.Owner); err != nil {
			return "", err
		}
		if err := solidity.ValidateAddress(fmt.Sprintf("erc721ApprovalOverrides[%d].spender", i), item.Spender); err != nil {
			return "", err
		}
		if item.TokenID == "" {
			return "", fmt.Errorf("erc721ApprovalOverrides[%d].tokenId is required", i)
		}
	}

	if err := validateCompilerConfig(req.Compiler); err != nil {
		return "", err
	}

	return rpcURL, nil
}

func validateCompilerConfig(config *model.CompilerConfig) error {
	if config == nil {
		return nil
	}

	config.Use = strings.TrimSpace(config.Use)
	config.EVMVersion = strings.TrimSpace(config.EVMVersion)
	config.RevertStrings = strings.TrimSpace(config.RevertStrings)
	switch config.RevertStrings {
	case "", "default", "strip", "debug", "verboseDebug":
	default:
		return fmt.Errorf("compiler.revertStrings must be one of default, strip, debug, or verboseDebug")
	}
	return nil
}

func (s *Service) compileStateOverride(ctx context.Context, sourcePath string, contractName string, compiler *model.CompilerConfig) (string, forge.Result, error) {
	contractID, err := solidity.ContractIdentifier(s.cfg.RepoRoot, sourcePath, contractName)
	if err != nil {
		return "", forge.Result{}, err
	}

	args := []string{"inspect", contractID, "bytecode", "--root", s.cfg.RepoRoot, "--contracts", ".", "--color", "never"}
	args = append(args, solidity.ForgeCompilerArgs(compiler)...)
	result := s.forge.Run(ctx, args...)
	if result.Err != nil {
		return "", result, result.Err
	}

	bytecode, ok := solidity.ExtractBytecode(result.Stdout)
	if !ok {
		return "", result, fmt.Errorf("forge inspect did not return bytecode for %s", contractID)
	}
	return bytecode, result, nil
}

func (s *Service) acquireRunSlot(ctx context.Context) (func(), error) {
	select {
	case s.runLimiter <- struct{}{}:
		return func() {
			<-s.runLimiter
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
