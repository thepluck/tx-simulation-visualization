package traceparser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"unicode"

	"foundry-tx-simulator/backend/internal/model"
)

type ParsedOutput struct {
	Trace          string
	ERC20Transfers []model.ERC20Transfer
}

type forgeOutput struct {
	Traces []forgeTraceEntry `json:"traces"`
}

type forgeTraceEntry struct {
	Kind string
	Body forgeTraceBody
}

type forgeTraceBody struct {
	Arena []forgeArenaNode `json:"arena"`
}

type forgeArenaNode struct {
	Parent   *int              `json:"parent"`
	Children []int             `json:"children"`
	Idx      int               `json:"idx"`
	Trace    forgeCallTrace    `json:"trace"`
	Logs     []json.RawMessage `json:"logs"`
}

type forgeCallTrace struct {
	Address string `json:"address"`
	Kind    string `json:"kind"`
}

type forgeLog struct {
	Address string       `json:"address"`
	Emitter string       `json:"emitter"`
	Topics  []string     `json:"topics"`
	Data    string       `json:"data"`
	RawLog  *forgeRawLog `json:"raw_log"`
}

type forgeRawLog struct {
	Address string   `json:"address"`
	Emitter string   `json:"emitter"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

const erc20TransferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

func ParseOutput(output string) (ParsedOutput, error) {
	raw := forgeJSONPayload(output)
	if len(raw) == 0 {
		return ParsedOutput{}, fmt.Errorf("forge json trace output is empty")
	}

	var payload forgeOutput
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ParsedOutput{}, fmt.Errorf("decode forge json trace: %w", err)
	}
	if len(payload.Traces) == 0 {
		return ParsedOutput{}, fmt.Errorf("forge json trace has no traces")
	}

	hasArena := false
	transfers := make([]model.ERC20Transfer, 0)
	for _, trace := range payload.Traces {
		hasArena = hasArena || len(trace.Body.Arena) > 0
		transfers = appendUniqueTransfer(transfers, trace.erc20Transfers()...)
	}
	if !hasArena {
		return ParsedOutput{}, fmt.Errorf("forge json trace has no arena nodes")
	}

	return ParsedOutput{
		Trace:          indentJSON(raw),
		ERC20Transfers: transfers,
	}, nil
}

func forgeJSONPayload(output string) []byte {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil
	}
	if strings.HasPrefix(output, "{") {
		return []byte(output)
	}
	start := strings.Index(output, "\n{")
	if start < 0 {
		return nil
	}
	return []byte(strings.TrimSpace(output[start+1:]))
}

func indentJSON(raw []byte) string {
	var output bytes.Buffer
	if err := json.Indent(&output, raw, "", "  "); err != nil {
		return strings.TrimSpace(string(raw))
	}
	return output.String()
}

func (entry *forgeTraceEntry) UnmarshalJSON(data []byte) error {
	var tuple []json.RawMessage
	if err := json.Unmarshal(data, &tuple); err != nil {
		return err
	}
	if len(tuple) != 2 {
		return fmt.Errorf("trace entry must be a [kind, body] tuple")
	}
	if err := json.Unmarshal(tuple[0], &entry.Kind); err != nil {
		return err
	}
	return json.Unmarshal(tuple[1], &entry.Body)
}

func (entry forgeTraceEntry) erc20Transfers() []model.ERC20Transfer {
	arenaByID := make(map[int]forgeArenaNode, len(entry.Body.Arena))
	for _, item := range entry.Body.Arena {
		arenaByID[item.Idx] = item
	}

	txRoots := transactionRootIDs(entry.Body.Arena)
	transfers := make([]model.ERC20Transfer, 0)
	for _, item := range entry.Body.Arena {
		if len(item.Logs) == 0 {
			continue
		}
		callContext, ok := nearestCallContext(item, arenaByID)
		if !ok {
			continue
		}
		if len(txRoots) > 0 && !isUnderAnyRoot(callContext.Idx, txRoots, arenaByID) {
			continue
		}

		for _, rawLog := range item.Logs {
			log, ok := parseForgeLog(rawLog)
			if !ok {
				continue
			}
			transfer, ok := erc20TransferFromLog(log, callContext.Trace.Address)
			if ok {
				transfers = appendUniqueTransfer(transfers, transfer)
			}
		}
	}
	return transfers
}

func transactionRootIDs(arena []forgeArenaNode) map[int]struct{} {
	roots := make(map[int]struct{})
	for index := len(arena) - 1; index >= 0; index-- {
		item := arena[index]
		if item.Parent == nil || *item.Parent != 0 {
			continue
		}
		roots[item.Idx] = struct{}{}
		break
	}
	return roots
}

func nearestCallContext(item forgeArenaNode, arenaByID map[int]forgeArenaNode) (forgeArenaNode, bool) {
	current := item
	for {
		if isTraceKind(current.Trace, "CALL") {
			return current, true
		}
		if current.Parent == nil {
			return forgeArenaNode{}, false
		}
		parent, ok := arenaByID[*current.Parent]
		if !ok {
			return forgeArenaNode{}, false
		}
		current = parent
	}
}

func isUnderAnyRoot(idx int, roots map[int]struct{}, arenaByID map[int]forgeArenaNode) bool {
	for {
		if _, ok := roots[idx]; ok {
			return true
		}
		item, ok := arenaByID[idx]
		if !ok || item.Parent == nil {
			return false
		}
		idx = *item.Parent
	}
}

func isTraceKind(trace forgeCallTrace, kind string) bool {
	return strings.EqualFold(strings.TrimSpace(trace.Kind), kind)
}

func parseForgeLog(raw json.RawMessage) (forgeLog, bool) {
	var log forgeLog
	if err := json.Unmarshal(raw, &log); err != nil {
		return forgeLog{}, false
	}
	if log.RawLog != nil {
		if log.Address == "" {
			log.Address = log.RawLog.Address
		}
		if log.Emitter == "" {
			log.Emitter = log.RawLog.Emitter
		}
		if len(log.Topics) == 0 {
			log.Topics = log.RawLog.Topics
		}
		if log.Data == "" {
			log.Data = log.RawLog.Data
		}
	}
	if log.Address == "" {
		log.Address = log.Emitter
	}
	return log, len(log.Topics) > 0
}

func erc20TransferFromLog(log forgeLog, tokenAddress string) (model.ERC20Transfer, bool) {
	if len(log.Topics) != 3 || normalizeHex(log.Topics[0]) != erc20TransferTopic {
		return model.ERC20Transfer{}, false
	}
	if normalizedHexBytes(log.Data) != 32 {
		return model.ERC20Transfer{}, false
	}

	token := normalizeAddress(tokenAddress)
	if token == "" {
		token = normalizeAddress(log.Address)
	}
	from := topicAddress(log.Topics[1])
	to := topicAddress(log.Topics[2])
	amount := hexUintToDecimal(log.Data)
	if token == "" || from == "" || to == "" || amount == "" {
		return model.ERC20Transfer{}, false
	}

	return model.ERC20Transfer{
		Token:  token,
		From:   from,
		To:     to,
		Amount: amount,
	}, true
}

func normalizeHex(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "0x") && !strings.HasPrefix(value, "0X") {
		return value
	}
	return "0x" + strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(value, "0x"), "0X"))
}

func normalizeAddress(value string) string {
	value = normalizeHex(value)
	if !isAddress(value) {
		return ""
	}
	return value
}

func topicAddress(topic string) string {
	topic = strings.TrimPrefix(normalizeHex(topic), "0x")
	if len(topic) < 40 {
		return ""
	}
	return "0x" + topic[len(topic)-40:]
}

func hexUintToDecimal(value string) string {
	n, ok := new(big.Int).SetString(strings.TrimPrefix(normalizeHex(value), "0x"), 16)
	if !ok {
		return ""
	}
	return n.String()
}

func normalizedHexBytes(value string) int {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		value = value[2:]
	}
	if value == "" {
		return 0
	}
	return (len(value) + 1) / 2
}

func appendUniqueTransfer(transfers []model.ERC20Transfer, items ...model.ERC20Transfer) []model.ERC20Transfer {
	for _, item := range items {
		exists := false
		for _, existing := range transfers {
			if existing == item {
				exists = true
				break
			}
		}
		if !exists {
			transfers = append(transfers, item)
		}
	}
	return transfers
}

func isAddress(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 42 || !strings.HasPrefix(value, "0x") {
		return false
	}
	for _, r := range value[2:] {
		if !unicode.IsDigit(r) && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}
