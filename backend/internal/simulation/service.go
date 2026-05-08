package simulation

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tx-simulation-visualization/backend/internal/config"
	"tx-simulation-visualization/backend/internal/forge"
	"tx-simulation-visualization/backend/internal/fundflow"
	"tx-simulation-visualization/backend/internal/model"
	"tx-simulation-visualization/backend/internal/prices"
	"tx-simulation-visualization/backend/internal/runid"
	"tx-simulation-visualization/backend/internal/solidity"
	"tx-simulation-visualization/backend/internal/traceparser"
)

const (
	scriptContractName = "SimulateTxScript"
	localFoundryDir    = "contracts"
	localScriptRelPath = "src/SimulateTx.s.sol"
	localScriptTarget  = localFoundryDir + "/" + localScriptRelPath + ":" + scriptContractName
	senderLabel        = "Sender"
)

type forgeRunner interface {
	Run(ctx context.Context, args ...string) forge.Result
}

type priceProvider interface {
	Fetch(ctx context.Context, chain string, tokens []string) (map[string]fundflow.TokenPrice, error)
}

type Service struct {
	cfg        config.Config
	forge      forgeRunner
	prices     priceProvider
	runLimiter chan struct{}
}

type foundryExecution struct {
	Root         string
	ScriptTarget string
	ScriptDir    string
	ScriptPath   string
	External     bool
	tempFiles    []string
}

func NewService(cfg config.Config) *Service {
	return &Service{
		cfg: cfg,
		forge: forge.Runner{
			Bin:      cfg.ForgeBin,
			RepoRoot: cfg.RepoRoot,
		},
		prices:     prices.DefaultProvider(cfg.RPCURLs),
		runLimiter: make(chan struct{}, cfg.MaxConcurrent),
	}
}

func (e *foundryExecution) cleanup() {
	for i := len(e.tempFiles) - 1; i >= 0; i-- {
		_ = os.Remove(e.tempFiles[i])
	}
}

func (s *Service) localFoundryRoot() string {
	return filepath.Join(s.cfg.RepoRoot, localFoundryDir)
}

func (s *Service) localScriptPath() string {
	return filepath.Join(s.localFoundryRoot(), filepath.FromSlash(localScriptRelPath))
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

	execution, err := s.prepareFoundryExecution(&req, runID)
	if err != nil {
		resp.Error = "prepare foundry project: " + err.Error()
		return resp, http.StatusInternalServerError
	}
	defer execution.cleanup()
	resp.ScriptPath = execution.ScriptPath

	if execution.External {
		buildResult := s.buildProjectSrc(ctx, execution, req.Compiler)
		if buildResult.Err != nil {
			status := populateForgeFailure(&resp, start, buildResult, rpcURL, req.Chain, "build project src", buildResult.Err)
			return resp, status
		}
	}

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

		statePath, err := s.writeStateOverrideSource(runDir, &execution, runID, source)
		if err != nil {
			resp.Error = "write state override source: " + err.Error()
			return resp, http.StatusInternalServerError
		}

		bytecode, compileResult, err := s.compileStateOverride(ctx, execution.Root, statePath, contractName, req.Compiler)
		if err != nil {
			status := populateForgeFailure(&resp, start, compileResult, rpcURL, req.Chain, "compile state override", err)
			return resp, status
		}
		stateBytecode = bytecode
	}

	forgeArgs := []string{
		"script",
		execution.ScriptTarget,
		"--sig",
		"run((address,string)[],(address,address,uint256)[],(address,address,address,uint256)[],(address,address,address,uint256)[],bytes,address,address,bytes)",
	}
	forgeArgs = append(forgeArgs, solidity.ForgeRunArgs(req, stateBytecode)...)
	forgeArgs = append(forgeArgs,
		"--root", execution.Root,
		"--rpc-url", rpcURL,
		"--fork-block-number", req.BlockNumber.String(),
		"-vvvvv",
		"--color", "never",
		"--non-interactive",
	)
	forgeArgs = append(forgeArgs, solidity.ForgeCompilerArgs(req.Compiler)...)
	if req.EtherscanAPIKey != "" {
		forgeArgs = append(forgeArgs, "--etherscan-api-key", req.EtherscanAPIKey)
	}

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
		return resp, http.StatusOK
	}
	resp.ERC20Transfers = fundflow.ExtractERC20Transfers(resp.Trace, resp.StructuredTrace, excludedERC721Tokens(req))
	priceMap := s.fetchTokenPrices(ctx, req.Chain, resp.ERC20Transfers)
	resp.ERC20Transfers = fundflow.EnrichERC20Transfers(resp.ERC20Transfers, priceMap)
	resp.BalanceAnalysis = fundflow.AnalyzeBalanceChanges(resp.ERC20Transfers, priceMap)

	return resp, http.StatusOK
}

func (s *Service) validateRequest(req *model.SimulateRequest) (string, error) {
	req.Chain = strings.TrimSpace(req.Chain)
	req.EtherscanAPIKey = strings.TrimSpace(req.EtherscanAPIKey)
	projectPath, err := s.normalizeProjectPath(req.ProjectPath)
	if err != nil {
		return "", err
	}
	req.ProjectPath = projectPath
	if err := validateSimulateRequest(req); err != nil {
		return "", err
	}

	rpcURL, ok := s.cfg.RPCURLs[req.Chain]
	if !ok {
		return "", fmt.Errorf("unknown chain %q", req.Chain)
	}
	if strings.TrimSpace(rpcURL) == "" {
		return "", fmt.Errorf("rpc url for chain %q is empty after environment expansion", req.Chain)
	}

	normalizedData, err := solidity.NormalizeBytes("data", req.Data)
	if err != nil {
		return "", err
	}
	req.Data = normalizedData
	ensureSenderLabel(req)

	if err := validateCompilerConfig(req.Compiler); err != nil {
		return "", err
	}

	return rpcURL, nil
}

func ensureSenderLabel(req *model.SimulateRequest) {
	for _, label := range req.LabelOverrides {
		if strings.EqualFold(strings.TrimSpace(label.Account), req.Sender) {
			return
		}
	}
	req.LabelOverrides = append([]model.LabelOverride{{Account: req.Sender, Label: senderLabel}}, req.LabelOverrides...)
}

func (s *Service) normalizeProjectPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	expandedValue, err := expandHomePath(value)
	if err != nil {
		return "", err
	}
	value = expandedValue
	if resolved, ok := existingDirectoryPath(s.cfg.RepoRoot, value); ok {
		return resolved, nil
	}
	if resolved, ok := s.resolveProjectRootPath(value); ok {
		return resolved, nil
	}
	absPath, err := absoluteProjectPath(s.cfg.RepoRoot, value)
	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("projectPath %q does not exist or is not mounted in the backend environment", absPath)
}

func (s *Service) resolveProjectRootPath(value string) (string, bool) {
	suffixes := pathSuffixes(value)
	for _, root := range s.cfg.ProjectRoots {
		for _, suffix := range suffixes {
			if resolved, ok := existingDirectoryPath(root, suffix); ok {
				return resolved, true
			}
		}
	}
	return "", false
}

func existingDirectoryPath(baseDir string, value string) (string, bool) {
	absPath, err := absoluteProjectPath(baseDir, value)
	if err != nil {
		return "", false
	}
	stat, err := os.Stat(absPath)
	if err != nil || !stat.IsDir() {
		return "", false
	}
	return absPath, true
}

func absoluteProjectPath(baseDir string, value string) (string, error) {
	value = strings.TrimSpace(value)
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

func pathSuffixes(value string) []string {
	cleaned := filepath.Clean(strings.TrimSpace(value))
	if cleaned == "." || cleaned == string(filepath.Separator) {
		return nil
	}
	parts := strings.Split(strings.Trim(cleaned, string(filepath.Separator)), string(filepath.Separator))
	suffixes := make([]string, 0, len(parts))
	for i := 0; i < len(parts); i++ {
		suffix := filepath.Join(parts[i:]...)
		if suffix != "." && suffix != "" {
			suffixes = append(suffixes, suffix)
		}
	}
	return suffixes
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

func (s *Service) prepareFoundryExecution(req *model.SimulateRequest, runID string) (foundryExecution, error) {
	if req.ProjectPath == "" {
		return foundryExecution{
			Root:         s.localFoundryRoot(),
			ScriptTarget: localScriptTarget,
			ScriptPath:   s.localScriptPath(),
		}, nil
	}

	execution := foundryExecution{
		Root:      req.ProjectPath,
		ScriptDir: filepath.Join(req.ProjectPath, "script"),
		External:  true,
	}
	if err := os.MkdirAll(execution.ScriptDir, 0o755); err != nil {
		return foundryExecution{}, err
	}

	sourcePath := s.localScriptPath()
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return foundryExecution{}, err
	}

	scriptName := "TxSimulation_" + safeRunID(runID) + ".s.sol"
	scriptPath := filepath.Join(execution.ScriptDir, scriptName)
	if err := os.WriteFile(scriptPath, source, 0o644); err != nil {
		return foundryExecution{}, err
	}

	execution.ScriptPath = scriptPath
	execution.ScriptTarget = filepath.ToSlash(scriptPath) + ":" + scriptContractName
	execution.tempFiles = append(execution.tempFiles, scriptPath)
	return execution, nil
}

func (s *Service) buildProjectSrc(ctx context.Context, execution foundryExecution, compiler *model.CompilerConfig) forge.Result {
	args := []string{"build", "src", "--root", execution.Root, "--color", "never"}
	args = append(args, solidity.ForgeCompilerArgsExplicit(compiler)...)
	return s.forge.Run(ctx, args...)
}

func (s *Service) writeStateOverrideSource(runDir string, execution *foundryExecution, runID string, source string) (string, error) {
	if execution.External {
		statePath := filepath.Join(execution.ScriptDir, "TxSimulationStateOverride_"+safeRunID(runID)+".sol")
		if err := os.WriteFile(statePath, []byte(source), 0o644); err != nil {
			return "", err
		}
		execution.tempFiles = append(execution.tempFiles, statePath)
		return statePath, nil
	}

	statePath := filepath.Join(runDir, "StateOverride.sol")
	if execution.Root != "" {
		stateDir := filepath.Join(execution.Root, ".txsim", safeRunID(runID))
		if err := os.MkdirAll(stateDir, 0o755); err != nil {
			return "", err
		}
		statePath = filepath.Join(stateDir, "StateOverride.sol")
		execution.tempFiles = append(execution.tempFiles, stateDir, statePath)
	}
	if err := os.WriteFile(statePath, []byte(source), 0o644); err != nil {
		return "", err
	}
	return statePath, nil
}

func safeRunID(runID string) string {
	return strings.NewReplacer(".", "_", "-", "_").Replace(runID)
}

func (s *Service) compileStateOverride(ctx context.Context, projectRoot string, sourcePath string, contractName string, compiler *model.CompilerConfig) (string, forge.Result, error) {
	contractID, err := solidity.ContractIdentifier(projectRoot, sourcePath, contractName)
	if err != nil {
		return "", forge.Result{}, err
	}

	args := []string{"inspect", contractID, "bytecode", "--root", projectRoot, "--contracts", ".", "--color", "never"}
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

func (s *Service) fetchTokenPrices(ctx context.Context, chain string, transfers []model.ERC20Transfer) map[string]fundflow.TokenPrice {
	if len(transfers) == 0 {
		return nil
	}
	priceMap := make(map[string]fundflow.TokenPrice)
	if s.prices != nil {
		if fetched, err := s.prices.Fetch(ctx, chain, transferTokens(transfers)); err == nil {
			priceMap = fetched
		}
	}
	return priceMap
}

func populateForgeFailure(resp *model.SimulateResponse, start time.Time, result forge.Result, rpcURL string, chain string, prefix string, err error) int {
	resp.DurationMillis = time.Since(start).Milliseconds()
	resp.Stdout = solidity.RedactRPC(solidity.StripANSI(result.Stdout), rpcURL, chain)
	resp.Stderr = solidity.RedactRPC(solidity.StripANSI(result.Stderr), rpcURL, chain)
	resp.Trace = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)
	resp.StructuredTrace = traceparser.Parse(resp.Trace)
	resp.ERC20Transfers = fundflow.ExtractERC20Transfers(resp.Trace, resp.StructuredTrace, nil)
	resp.ExitCode = result.ExitCode
	resp.Error = prefix + ": " + err.Error()
	if result.Err != nil {
		return forge.StatusFromCommandError(result.Err)
	}
	return http.StatusInternalServerError
}

func transferTokens(transfers []model.ERC20Transfer) []string {
	seen := make(map[string]struct{}, len(transfers))
	tokens := make([]string, 0, len(transfers))
	for _, transfer := range transfers {
		token := strings.ToLower(strings.TrimSpace(transfer.Token))
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}
	return tokens
}

func excludedERC721Tokens(req model.SimulateRequest) []string {
	tokens := make([]string, 0, len(req.ERC721ApprovalOverrides))
	for _, override := range req.ERC721ApprovalOverrides {
		tokens = append(tokens, override.Token)
	}
	return tokens
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
