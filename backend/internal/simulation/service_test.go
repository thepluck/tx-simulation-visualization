package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tx-simulation-visualization/backend/internal/config"
	"tx-simulation-visualization/backend/internal/forge"
	"tx-simulation-visualization/backend/internal/fundflow"
	"tx-simulation-visualization/backend/internal/model"
)

const (
	wethAddress = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
	baycAddress = "0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D"
	baycTokenID = "1"
)

func TestSimulateWETHBalanceApprovalAndTransferFrom(t *testing.T) {
	cfg := loadTestConfig(t)
	blockNumber := mainnetBlockNumber(t, cfg.RPCURLs["mainnet"])

	owner := "0x0000000000000000000000000000000000000001"
	spender := "0x0000000000000000000000000000000000000002"
	recipient := "0x0000000000000000000000000000000000000003"
	amount := "1000000000000000000"

	req := model.SimulateRequest{
		Chain:       "mainnet",
		BlockNumber: model.Uint256(blockNumber),
		LabelOverrides: []model.LabelOverride{
			{Account: owner, Label: "WETHOwner"},
			{Account: spender, Label: "WETHSpender"},
			{Account: recipient, Label: "WETHRecipient"},
		},
		ERC20BalanceOverrides: []model.ERC20BalanceOverride{
			{
				Token:   wethAddress,
				Account: owner,
				Balance: model.Uint256(amount),
			},
		},
		ERC20ApprovalOverrides: []model.ERC20ApprovalOverride{
			{
				Token:   wethAddress,
				Owner:   owner,
				Spender: spender,
				Amount:  model.Uint256(amount),
			},
		},
		Compiler: &model.CompilerConfig{
			ViaIR:         boolPtr(true),
			Optimize:      boolPtr(true),
			OptimizerRuns: uint32Ptr(200),
			EVMVersion:    "cancun",
			RevertStrings: "default",
		},
		Sender: spender,
		Target: wethAddress,
		Data:   transferFromCalldata(owner, recipient, mustBigInt(t, amount)),
	}

	resp, status := newTestService(cfg).Simulate(context.Background(), req)
	t.Cleanup(func() {
		if resp.RunDir != "" {
			_ = os.RemoveAll(resp.RunDir)
		}
	})

	requireSimulationSuccess(t, status, resp)
	logResponseIfEnabled(t, resp)
	requireStructuredTrace(t, resp)
	if !strings.Contains(resp.Trace, "transferFrom") {
		t.Fatalf("expected transferFrom in trace, got:\n%s", resp.Trace)
	}
	for _, want := range []string{"WETHOwner", "WETHSpender", "WETHRecipient"} {
		if !strings.Contains(resp.Trace, want) {
			t.Fatalf("expected label %q in trace, got:\n%s", want, resp.Trace)
		}
	}
	requireERC20Transfer(t, resp, amount, owner, recipient)
	requireBalanceAnalysis(t, resp, amount, "-2000", "2000")
}

func TestSimulateStateOverrideContractDealsWETHBalanceAndApproval(t *testing.T) {
	cfg := loadTestConfig(t)
	blockNumber := mainnetBlockNumber(t, cfg.RPCURLs["mainnet"])

	owner := "0x0000000000000000000000000000000000000011"
	spender := "0x0000000000000000000000000000000000000012"
	recipient := "0x0000000000000000000000000000000000000013"
	amount := "1000000000000000000"

	req := model.SimulateRequest{
		Chain:       "mainnet",
		BlockNumber: model.Uint256(blockNumber),
		StateOverride: &model.StateOverride{
			ContractName: "WETHStateOverride",
			Source:       wethStateOverrideSource(owner, spender, amount),
		},
		Sender: spender,
		Target: wethAddress,
		Data:   transferFromCalldata(owner, recipient, mustBigInt(t, amount)),
	}

	resp, status := newTestService(cfg).Simulate(context.Background(), req)
	t.Cleanup(func() {
		if resp.RunDir != "" {
			_ = os.RemoveAll(resp.RunDir)
		}
	})

	requireSimulationSuccess(t, status, resp)
	logResponseIfEnabled(t, resp)
	requireStructuredTrace(t, resp)
	for _, want := range []string{"fallback", "approve", "transferFrom"} {
		if !strings.Contains(resp.Trace, want) {
			t.Fatalf("expected %q in trace, got:\n%s", want, resp.Trace)
		}
	}
}

func TestSimulateNFTApprovalAndTransferFrom(t *testing.T) {
	cfg := loadTestConfig(t)
	rpcURL := cfg.RPCURLs["mainnet"]
	blockNumber := mainnetBlockNumber(t, rpcURL)
	owner := erc721OwnerOf(t, rpcURL, blockNumber, baycAddress, mustBigInt(t, baycTokenID))

	spender := "0x0000000000000000000000000000000000000002"
	recipient := "0x0000000000000000000000000000000000000003"

	req := model.SimulateRequest{
		Chain:       "mainnet",
		BlockNumber: model.Uint256(blockNumber),
		ERC721ApprovalOverrides: []model.ERC721ApprovalOverride{
			{
				Token:   baycAddress,
				Owner:   owner,
				Spender: spender,
				TokenID: model.Uint256(baycTokenID),
			},
		},
		Sender: spender,
		Target: baycAddress,
		Data:   transferFromCalldata(owner, recipient, mustBigInt(t, baycTokenID)),
	}

	resp, status := newTestService(cfg).Simulate(context.Background(), req)
	t.Cleanup(func() {
		if resp.RunDir != "" {
			_ = os.RemoveAll(resp.RunDir)
		}
	})

	requireSimulationSuccess(t, status, resp)
	logResponseIfEnabled(t, resp)
	requireStructuredTrace(t, resp)
	if !strings.Contains(resp.Trace, "transferFrom") {
		t.Fatalf("expected transferFrom in trace, got:\n%s", resp.Trace)
	}
	if len(resp.ERC20Transfers) != 0 {
		t.Fatalf("expected no ERC20 transfers for NFT transfer, got %#v", resp.ERC20Transfers)
	}
}

func TestSimulateReturnsTraceWhenScriptFailsWithoutFundFlow(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "contracts"), 0o755); err != nil {
		t.Fatal(err)
	}
	scriptSource := []byte("// SPDX-License-Identifier: UNLICENSED\npragma solidity ^0.8.0;\ncontract SimulateTxScript {}\n")
	if err := os.WriteFile(filepath.Join(repoRoot, "contracts", "SimulateTx.s.sol"), scriptSource, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		ListenAddr:     "127.0.0.1:0",
		RepoRoot:       repoRoot,
		WorkDir:        filepath.Join(t.TempDir(), "runs"),
		TimeoutSeconds: 30,
		MaxConcurrent:  1,
		ForgeBin:       "forge",
		RPCURLs: map[string]string{
			"mainnet": "http://127.0.0.1:8545",
		},
	}
	fake := &fakeForgeRunner{
		results: []forge.Result{
			{
				Stdout: `Traces:
  [1000] SimulateTxScript::run()
    ├─ [500] WETH9::transfer(0x0000000000000000000000000000000000000003, 1)
    │   ├─ emit Transfer(from: 0x0000000000000000000000000000000000000001, to: 0x0000000000000000000000000000000000000003, value: 1)
    │   └─ ← [Revert] ERC20: transfer amount exceeds balance
    └─ ← [Revert] ERC20: transfer amount exceeds balance
`,
				Stderr:   "Error: script failed\n",
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
			},
		},
	}
	service := NewService(cfg)
	service.forge = fake

	resp, status := service.Simulate(context.Background(), model.SimulateRequest{
		Chain:       "mainnet",
		BlockNumber: "1",
		Sender:      "0x0000000000000000000000000000000000000001",
		Target:      "0x0000000000000000000000000000000000000002",
		Data:        "0x",
	})

	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200; resp=%#v", status, resp)
	}
	if resp.Success {
		t.Fatalf("success = true, want false for failing forge script")
	}
	if resp.ExitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", resp.ExitCode)
	}
	if resp.Error != "" {
		t.Fatalf("error = %q, want empty", resp.Error)
	}
	requireStructuredTrace(t, resp)
	if len(resp.ERC20Transfers) != 0 {
		t.Fatalf("ERC20 transfers should be skipped on script failure: %#v", resp.ERC20Transfers)
	}
	if resp.BalanceAnalysis != nil {
		t.Fatalf("balance analysis should be skipped on script failure: %#v", resp.BalanceAnalysis)
	}
	encoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"erc20Transfers", "balanceAnalysis", "error"} {
		if _, ok := payload[field]; ok {
			t.Fatalf("failed script response should omit %q: %s", field, encoded)
		}
	}
}

func TestSimulateExternalProjectBuildsSrcCompilesOverrideAndRunsCopiedScript(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "contracts"), 0o755); err != nil {
		t.Fatal(err)
	}
	scriptSource := []byte("// SPDX-License-Identifier: UNLICENSED\npragma solidity ^0.8.0;\ncontract SimulateTxScript {}\n")
	if err := os.WriteFile(filepath.Join(repoRoot, "contracts", "SimulateTx.s.sol"), scriptSource, 0o644); err != nil {
		t.Fatal(err)
	}

	projectRoot := t.TempDir()
	cfg := config.Config{
		ListenAddr:     "127.0.0.1:0",
		RepoRoot:       repoRoot,
		WorkDir:        filepath.Join(t.TempDir(), "runs"),
		TimeoutSeconds: 30,
		MaxConcurrent:  1,
		ForgeBin:       "forge",
		RPCURLs: map[string]string{
			"mainnet": "http://127.0.0.1:8545",
		},
	}
	fake := &fakeForgeRunner{
		results: []forge.Result{
			{Stdout: "build ok\n"},
			{Stdout: "0x6000\n"},
			{Stdout: "Traces:\n  [1] SimulateTxScript::run()\n"},
		},
	}
	service := NewService(cfg)
	service.forge = fake

	resp, status := service.Simulate(context.Background(), model.SimulateRequest{
		Chain:           "mainnet",
		BlockNumber:     "1",
		ProjectPath:     projectRoot,
		EtherscanAPIKey: "etherscan-test-key",
		StateOverride: &model.StateOverride{
			ContractName: "OverrideState",
			Source:       "pragma solidity ^0.8.0; contract OverrideState {}",
		},
		Sender: "0x0000000000000000000000000000000000000001",
		Target: "0x0000000000000000000000000000000000000002",
		Data:   "0x",
	})

	if status != http.StatusOK || !resp.Success {
		t.Fatalf("simulation failed: status=%d resp=%#v", status, resp)
	}
	if len(fake.calls) != 3 {
		t.Fatalf("forge call count = %d, want 3: %#v", len(fake.calls), fake.calls)
	}

	buildArgs := fake.calls[0]
	if !hasArgSequence(buildArgs, "build", "src") || !hasArgSequence(buildArgs, "--root", projectRoot) {
		t.Fatalf("unexpected build args: %#v", buildArgs)
	}
	if containsArg(buildArgs, "--via-ir") {
		t.Fatalf("build should use target project defaults unless request compiler fields are set: %#v", buildArgs)
	}

	inspectArgs := fake.calls[1]
	if !hasArgSequence(inspectArgs, "inspect") ||
		!strings.HasPrefix(inspectArgs[1], "script/TxSimulationStateOverride_") ||
		!strings.HasSuffix(inspectArgs[1], ".sol:OverrideState") ||
		!hasArgSequence(inspectArgs, "--root", projectRoot) {
		t.Fatalf("unexpected inspect args: %#v", inspectArgs)
	}

	scriptArgs := fake.calls[2]
	if !hasArgSequence(scriptArgs, "script") ||
		!filepath.IsAbs(strings.TrimSuffix(scriptArgs[1], ":SimulateTxScript")) ||
		!strings.HasPrefix(scriptArgs[1], filepath.ToSlash(filepath.Join(projectRoot, "script", "TxSimulation_"))) ||
		!strings.HasSuffix(scriptArgs[1], ".s.sol:SimulateTxScript") ||
		!hasArgSequence(scriptArgs, "--root", projectRoot) ||
		!hasArgSequence(scriptArgs, "--etherscan-api-key", "etherscan-test-key") ||
		!containsArg(scriptArgs, "0x6000") {
		t.Fatalf("unexpected script args: %#v", scriptArgs)
	}
	if scriptArgs[4] != `[(0x0000000000000000000000000000000000000001,"Sender")]` {
		t.Fatalf("sender label arg = %q, want default Sender label", scriptArgs[4])
	}

	entries, err := os.ReadDir(filepath.Join(projectRoot, "script"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("temporary script files were not cleaned up: %#v", entries)
	}
}

func logResponseIfEnabled(t *testing.T, resp model.SimulateResponse) {
	t.Helper()

	if os.Getenv("TXSIM_LOG_RESPONSE") == "1" {
		encoded, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("response:\n%s", encoded)
	}
}

func loadTestConfig(t *testing.T) config.Config {
	t.Helper()

	oldConfigPath, hadConfigPath := os.LookupEnv("TXSIM_CONFIG")
	configPath := filepath.Clean(filepath.Join("..", "..", "config.example.yaml"))
	if err := os.Setenv("TXSIM_CONFIG", configPath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if hadConfigPath {
			_ = os.Setenv("TXSIM_CONFIG", oldConfigPath)
		} else {
			_ = os.Unsetenv("TXSIM_CONFIG")
		}
	})

	cfg, _, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(cfg.RPCURLs["mainnet"]) == "" {
		t.Skip("mainnet RPC URL is required")
	}
	cfg.TimeoutSeconds = 180
	return cfg
}

func requireSimulationSuccess(t *testing.T, status int, resp model.SimulateResponse) {
	t.Helper()

	if status != http.StatusOK || !resp.Success {
		t.Fatalf("simulation failed: status=%d success=%v exitCode=%d error=%q\nstdout:\n%s\nstderr:\n%s\ntrace:\n%s",
			status,
			resp.Success,
			resp.ExitCode,
			resp.Error,
			resp.Stdout,
			resp.Stderr,
			resp.Trace,
		)
	}
}

func requireStructuredTrace(t *testing.T, resp model.SimulateResponse) {
	t.Helper()

	if len(resp.StructuredTrace) == 0 {
		t.Fatalf("expected structured trace nodes for raw trace:\n%s", resp.Trace)
	}
	if resp.StructuredTrace[0].Kind != "call" {
		t.Fatalf("expected root call node, got %#v", resp.StructuredTrace[0])
	}
}

func requireERC20Transfer(t *testing.T, resp model.SimulateResponse, amount string, fromNeedle string, toNeedle string) {
	t.Helper()

	for _, transfer := range resp.ERC20Transfers {
		if transfer.Amount == amount && strings.Contains(transfer.From, fromNeedle) && strings.Contains(transfer.To, toNeedle) {
			if strings.TrimSpace(transfer.Token) == "" {
				t.Fatalf("expected transfer token: %#v", transfer)
			}
			return
		}
	}
	t.Fatalf("expected ERC20 transfer amount=%s from~%s to~%s in %#v", amount, fromNeedle, toNeedle, resp.ERC20Transfers)
}

func requireBalanceAnalysis(t *testing.T, resp model.SimulateResponse, rawAmount string, ownerUSD string, recipientUSD string) {
	t.Helper()

	if resp.BalanceAnalysis == nil {
		t.Fatal("expected balance analysis")
	}
	if len(resp.BalanceAnalysis.Changes) != 2 {
		t.Fatalf("balance changes = %#v, want 2", resp.BalanceAnalysis.Changes)
	}
	for _, change := range resp.BalanceAnalysis.Changes {
		if change.RawAmount != rawAmount && change.RawAmount != "-"+rawAmount {
			t.Fatalf("unexpected raw amount in change: %#v", change)
		}
		if change.USDValue == nil {
			t.Fatalf("expected USD value in change: %#v", change)
		}
	}

	wantTotals := map[string]string{
		"0x0000000000000000000000000000000000000001": ownerUSD,
		"0x0000000000000000000000000000000000000003": recipientUSD,
	}
	for _, total := range resp.BalanceAnalysis.UserTotals {
		if want, ok := wantTotals[total.User]; ok {
			got := fmt.Sprintf("%.0f", total.USDValue)
			if got != want {
				t.Fatalf("user %s usd total = %s, want %s", total.User, got, want)
			}
			delete(wantTotals, total.User)
		}
	}
	if len(wantTotals) != 0 {
		t.Fatalf("missing user USD totals: %#v in %#v", wantTotals, resp.BalanceAnalysis.UserTotals)
	}
}

func wethStateOverrideSource(owner string, spender string, amount string) string {
	return fmt.Sprintf(`// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.30;

import "forge-std/Test.sol";

interface IERC20 {
  function approve(address spender, uint256 amount) external returns (bool);
}

contract WETHStateOverride is Test {
  address internal constant WETH = %s;
  address internal constant OWNER = %s;
  address internal constant SPENDER = %s;
  uint256 internal constant AMOUNT = %s;

  fallback() external {
    deal(WETH, OWNER, AMOUNT);
    vm.prank(OWNER);
    IERC20(WETH).approve(SPENDER, AMOUNT);
  }
}
`, wethAddress, owner, spender, amount)
}

func mainnetBlockNumber(t *testing.T, rpcURL string) string {
	t.Helper()

	var result string
	callRPC(t, rpcURL, "eth_blockNumber", []any{}, &result)
	n := new(big.Int)
	if _, ok := n.SetString(strings.TrimPrefix(result, "0x"), 16); !ok {
		t.Fatalf("invalid eth_blockNumber result: %q", result)
	}
	return n.String()
}

func erc721OwnerOf(t *testing.T, rpcURL string, blockNumber string, token string, tokenID *big.Int) string {
	t.Helper()

	blockHex := "0x" + mustBigInt(t, blockNumber).Text(16)
	data := "0x6352211e" + encodeUint256(tokenID)
	params := []any{
		map[string]string{
			"to":   token,
			"data": data,
		},
		blockHex,
	}

	var result string
	callRPC(t, rpcURL, "eth_call", params, &result)
	result = strings.TrimPrefix(result, "0x")
	if len(result) != 64 {
		t.Fatalf("unexpected ownerOf result length: %q", result)
	}
	return "0x" + result[24:]
}

func callRPC(t *testing.T, rpcURL string, method string, params []any, result any) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		t.Fatal(err)
	}
	if rpcResp.Error != nil {
		t.Fatalf("rpc %s failed: %d %s", method, rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if len(rpcResp.Result) == 0 {
		t.Fatalf("rpc %s returned empty result", method)
	}
	if err := json.Unmarshal(rpcResp.Result, result); err != nil {
		t.Fatal(err)
	}
}

func transferFromCalldata(from string, to string, tokenIDOrAmount *big.Int) string {
	return "0x23b872dd" + encodeAddress(from) + encodeAddress(to) + encodeUint256(tokenIDOrAmount)
}

func encodeAddress(address string) string {
	return leftPadHex(strings.TrimPrefix(strings.ToLower(address), "0x"), 64)
}

func encodeUint256(value *big.Int) string {
	return leftPadHex(value.Text(16), 64)
}

func leftPadHex(value string, length int) string {
	if len(value) >= length {
		return value
	}
	return strings.Repeat("0", length-len(value)) + value
}

func boolPtr(value bool) *bool {
	return &value
}

func uint32Ptr(value uint32) *uint32 {
	return &value
}

type fakeForgeRunner struct {
	calls   [][]string
	results []forge.Result
}

func (f *fakeForgeRunner) Run(_ context.Context, args ...string) forge.Result {
	copiedArgs := append([]string(nil), args...)
	f.calls = append(f.calls, copiedArgs)
	if len(f.results) == 0 {
		return forge.Result{}
	}
	result := f.results[0]
	f.results = f.results[1:]
	return result
}

type fakePriceProvider struct {
	prices map[string]fundflow.TokenPrice
}

func (f fakePriceProvider) Fetch(_ context.Context, _ string, _ []string) (map[string]fundflow.TokenPrice, error) {
	return f.prices, nil
}

func newTestService(cfg config.Config) *Service {
	service := NewService(cfg)
	service.prices = fakePriceProvider{
		prices: map[string]fundflow.TokenPrice{
			strings.ToLower(wethAddress): {
				PriceUSD: 2000,
				Decimals: 18,
			},
		},
	}
	return service
}

func hasArgSequence(args []string, want ...string) bool {
	if len(want) == 0 || len(want) > len(args) {
		return false
	}
	for i := 0; i <= len(args)-len(want); i++ {
		matched := true
		for j := range want {
			if args[i+j] != want[j] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func mustBigInt(t *testing.T, value string) *big.Int {
	t.Helper()

	n, ok := new(big.Int).SetString(value, 10)
	if !ok {
		t.Fatalf("invalid integer %q", value)
	}
	return n
}

func TestTransferFromCalldata(t *testing.T) {
	got := transferFromCalldata(
		"0x0000000000000000000000000000000000000001",
		"0x0000000000000000000000000000000000000003",
		mustBigInt(t, "1000000000000000000"),
	)
	want := "0x23b872dd000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000de0b6b3a7640000"
	if got != want {
		t.Fatalf("transferFromCalldata() = %s, want %s", got, want)
	}
}

func TestSimulateResponseJSONOmitsInternalFields(t *testing.T) {
	resp := model.SimulateResponse{
		ID:              "run",
		Success:         true,
		ExitCode:        0,
		DurationMillis:  1,
		Trace:           "Traces:",
		StructuredTrace: []model.TraceNode{{Kind: "call", Raw: "raw"}},
		Stdout:          "stdout",
		Stderr:          "stderr",
		RunDir:          "/tmp/run",
		ScriptPath:      "/tmp/script.sol",
	}

	encoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	payload := string(encoded)
	for _, field := range []string{"stdout", "stderr", "runDir", "scriptPath"} {
		if strings.Contains(payload, field) {
			t.Fatalf("response JSON should not include %q: %s", field, payload)
		}
	}
	if strings.Contains(payload, "depth") {
		t.Fatalf("response JSON should not include trace node depth: %s", payload)
	}
	for _, field := range []string{"structuredTrace", "trace", "success"} {
		if !strings.Contains(payload, field) {
			t.Fatalf("response JSON should include %q: %s", field, payload)
		}
	}
}

func Example_transferFromCalldata() {
	fmt.Println(transferFromCalldata(
		"0x0000000000000000000000000000000000000001",
		"0x0000000000000000000000000000000000000003",
		big.NewInt(1),
	))
	// Output:
	// 0x23b872dd000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000001
}
