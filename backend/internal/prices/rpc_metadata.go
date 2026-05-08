package prices

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"tx-simulation-visualization/backend/internal/fundflow"
)

const (
	erc20DecimalsSelector = "0x313ce567"
	erc20SymbolSelector   = "0x95d89b41"
)

type RPCMetadataProvider struct {
	Client  *http.Client
	RPCURLs map[string]string
}

func (p RPCMetadataProvider) Fetch(ctx context.Context, chain string, tokens []string) (map[string]fundflow.TokenPrice, error) {
	rpcURL := strings.TrimSpace(p.RPCURLs[chain])
	if rpcURL == "" {
		return nil, nil
	}
	tokens = normalizeTokens(tokens)
	if len(tokens) == 0 {
		return nil, nil
	}

	out := make(map[string]fundflow.TokenPrice)
	client := defaultHTTPClient(p.Client)
	for _, token := range tokens {
		metadata := fundflow.TokenPrice{LogoURL: trustWalletLogoURL(chain, token)}
		if decimals, ok := p.fetchDecimals(ctx, client, rpcURL, token); ok {
			metadata.Decimals = decimals
			metadata.HasDecimals = true
		}
		if symbol, ok := p.fetchSymbol(ctx, client, rpcURL, token); ok {
			metadata.Symbol = symbol
		}
		if metadata.HasDecimals || metadata.Symbol != "" || metadata.LogoURL != "" {
			out[token] = metadata
		}
	}
	return out, nil
}

func (p RPCMetadataProvider) fetchDecimals(ctx context.Context, client *http.Client, rpcURL string, token string) (int, bool) {
	result, err := ethCall(ctx, client, rpcURL, token, erc20DecimalsSelector)
	if err != nil {
		return 0, false
	}
	return parseUintResult(result)
}

func (p RPCMetadataProvider) fetchSymbol(ctx context.Context, client *http.Client, rpcURL string, token string) (string, bool) {
	result, err := ethCall(ctx, client, rpcURL, token, erc20SymbolSelector)
	if err != nil {
		return "", false
	}
	return parseStringResult(result)
}

func ethCall(ctx context.Context, client *http.Client, rpcURL string, to string, data string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_call",
		"params": []any{
			map[string]string{
				"to":   to,
				"data": data,
			},
			"latest",
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("rpc metadata request failed: %s", resp.Status)
	}

	var payload struct {
		Result string `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Error != nil {
		return "", fmt.Errorf("rpc metadata call failed: %d %s", payload.Error.Code, payload.Error.Message)
	}
	return payload.Result, nil
}

func parseUintResult(value string) (int, bool) {
	raw := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	if raw == "" {
		return 0, false
	}
	parsed, ok := new(big.Int).SetString(raw, 16)
	if !ok || parsed.Sign() < 0 || !parsed.IsInt64() {
		return 0, false
	}
	out := parsed.Int64()
	if out < 0 || out > 255 {
		return 0, false
	}
	return int(out), true
}

func parseStringResult(value string) (string, bool) {
	raw := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	if len(raw) < 64 {
		return "", false
	}

	if parsed, ok := parseDynamicStringResult(raw); ok {
		return parsed, true
	}
	return parseBytes32StringResult(raw[:64])
}

func parseDynamicStringResult(raw string) (string, bool) {
	offset, ok := new(big.Int).SetString(raw[:64], 16)
	if !ok || !offset.IsInt64() {
		return "", false
	}
	offsetIndex := int(offset.Int64()) * 2
	if offsetIndex < 0 || offsetIndex+64 > len(raw) {
		return "", false
	}

	length, ok := new(big.Int).SetString(raw[offsetIndex:offsetIndex+64], 16)
	if !ok || !length.IsInt64() {
		return "", false
	}
	dataStart := offsetIndex + 64
	dataEnd := dataStart + int(length.Int64())*2
	if dataEnd < dataStart || dataEnd > len(raw) {
		return "", false
	}
	return decodeHexString(raw[dataStart:dataEnd])
}

func parseBytes32StringResult(raw string) (string, bool) {
	decoded, err := hex.DecodeString(raw)
	if err != nil {
		return "", false
	}
	out := strings.TrimSpace(strings.TrimRight(string(decoded), "\x00"))
	if out == "" {
		return "", false
	}
	return out, true
}

func decodeHexString(raw string) (string, bool) {
	if raw == "" || len(raw)%2 != 0 {
		return "", false
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil {
		return "", false
	}
	out := strings.TrimSpace(string(decoded))
	if out == "" {
		return "", false
	}
	return out, true
}
