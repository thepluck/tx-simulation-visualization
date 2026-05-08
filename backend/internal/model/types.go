package model

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

type SimulateRequest struct {
	Chain                   string                   `json:"chain"`
	BlockNumber             Uint256                  `json:"blockNumber"`
	ProjectPath             string                   `json:"projectPath"`
	LabelOverrides          []LabelOverride          `json:"labelOverrides"`
	ERC20BalanceOverrides   []ERC20BalanceOverride   `json:"erc20BalanceOverrides"`
	ERC20ApprovalOverrides  []ERC20ApprovalOverride  `json:"erc20ApprovalOverrides"`
	ERC721ApprovalOverrides []ERC721ApprovalOverride `json:"erc721ApprovalOverrides"`
	StateOverride           *StateOverride           `json:"stateOverride"`
	StateOverrideCode       string                   `json:"stateOverrideCode"`
	StateOverrideContract   string                   `json:"stateOverrideContractName"`
	Compiler                *CompilerConfig          `json:"compiler"`
	EtherscanAPIKey         string                   `json:"etherscanApiKey"`
	Sender                  string                   `json:"sender"`
	Target                  string                   `json:"target"`
	Data                    string                   `json:"data"`
}

type LabelOverride struct {
	Account string `json:"account"`
	Label   string `json:"label"`
}

type ERC20BalanceOverride struct {
	Token   string  `json:"token"`
	Account string  `json:"account"`
	Balance Uint256 `json:"balance"`
}

type ERC20ApprovalOverride struct {
	Token   string  `json:"token"`
	Owner   string  `json:"owner"`
	Spender string  `json:"spender"`
	Amount  Uint256 `json:"amount"`
}

type ERC721ApprovalOverride struct {
	Token   string  `json:"token"`
	Owner   string  `json:"owner"`
	Spender string  `json:"spender"`
	TokenID Uint256 `json:"tokenId"`
}

type StateOverride struct {
	Source       string `json:"source"`
	ContractName string `json:"contractName"`
}

type CompilerConfig struct {
	Use               string  `json:"use"`
	Offline           bool    `json:"offline"`
	NoAutoDetect      bool    `json:"noAutoDetect"`
	ViaIR             *bool   `json:"viaIR"`
	UseLiteralContent bool    `json:"useLiteralContent"`
	NoMetadata        bool    `json:"noMetadata"`
	EVMVersion        string  `json:"evmVersion"`
	Optimize          *bool   `json:"optimize"`
	OptimizerRuns     *uint32 `json:"optimizerRuns"`
	RevertStrings     string  `json:"revertStrings"`
}

type SimulateResponse struct {
	ID              string           `json:"id"`
	Success         bool             `json:"success"`
	ExitCode        int              `json:"exitCode"`
	DurationMillis  int64            `json:"durationMillis"`
	Trace           string           `json:"trace"`
	StructuredTrace []TraceNode      `json:"structuredTrace,omitempty"`
	ERC20Transfers  []ERC20Transfer  `json:"erc20Transfers,omitempty"`
	BalanceAnalysis *BalanceAnalysis `json:"balanceAnalysis,omitempty"`
	Stdout          string           `json:"-"`
	Stderr          string           `json:"-"`
	Error           string           `json:"error,omitempty"`
	RunDir          string           `json:"-"`
	ScriptPath      string           `json:"-"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ERC20Transfer struct {
	Token            string `json:"token"`
	From             string `json:"from"`
	To               string `json:"to"`
	Amount           string `json:"amount"`
	NormalizedAmount string `json:"normalizedAmount,omitempty"`
	Symbol           string `json:"symbol,omitempty"`
	LogoURL          string `json:"logoUrl,omitempty"`
}

type BalanceAnalysis struct {
	Changes    []TokenBalanceChange `json:"changes,omitempty"`
	UserTotals []UserUSDChange      `json:"userTotals,omitempty"`
}

type TokenBalanceChange struct {
	User      string   `json:"user"`
	Token     string   `json:"token"`
	Symbol    string   `json:"symbol,omitempty"`
	LogoURL   string   `json:"logoUrl,omitempty"`
	RawAmount string   `json:"rawAmount"`
	Amount    string   `json:"amount"`
	USDValue  *float64 `json:"usdValue,omitempty"`
}

type UserUSDChange struct {
	User     string  `json:"user"`
	USDValue float64 `json:"usdValue"`
}

type TraceNode struct {
	Raw        string      `json:"raw"`
	Kind       string      `json:"kind"`
	Depth      int         `json:"-"`
	Gas        uint64      `json:"gas,omitempty"`
	Target     string      `json:"target,omitempty"`
	Function   string      `json:"function,omitempty"`
	Arguments  string      `json:"arguments,omitempty"`
	CallType   string      `json:"callType,omitempty"`
	ResultType string      `json:"resultType,omitempty"`
	Value      string      `json:"value,omitempty"`
	Children   []TraceNode `json:"children,omitempty"`
}

type Uint256 string

func (req SimulateRequest) StateOverrideSourceAndName() (string, string) {
	if req.StateOverride != nil {
		return req.StateOverride.Source, req.StateOverride.ContractName
	}
	return req.StateOverrideCode, req.StateOverrideContract
}

func (u *Uint256) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "null" {
		return nil
	}

	var value string
	if len(raw) >= 2 && raw[0] == '"' {
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
	} else {
		value = raw
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	n := new(big.Int)
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		parsed, ok := n.SetString(value[2:], 16)
		if !ok {
			return fmt.Errorf("invalid uint256 hex value %q", value)
		}
		n = parsed
	} else {
		if strings.HasPrefix(value, "-") {
			return fmt.Errorf("uint256 cannot be negative: %q", value)
		}
		parsed, ok := n.SetString(value, 10)
		if !ok {
			return fmt.Errorf("invalid uint256 decimal value %q", value)
		}
		n = parsed
	}

	maxUint256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	if n.Sign() < 0 || n.Cmp(maxUint256) > 0 {
		return fmt.Errorf("uint256 out of range: %q", value)
	}

	*u = Uint256(n.String())
	return nil
}

func (u Uint256) String() string {
	if u == "" {
		return "0"
	}
	return string(u)
}
