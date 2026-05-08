package fundflow

import (
	"math/big"
	"regexp"
	"strings"

	"tx-simulation-visualization/backend/internal/model"
)

const erc20TransferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

var (
	addressPattern = regexp.MustCompile(`0x[0-9a-fA-F]{40}`)
	topicPattern   = regexp.MustCompile(`topic\s+([12]):\s+(0x[0-9a-fA-F]{64})`)
	dataPattern    = regexp.MustCompile(`data:\s+(0x[0-9a-fA-F]+)`)
)

func ExtractERC20Transfers(trace string, nodes []model.TraceNode, excludedTokens []string) []model.ERC20Transfer {
	excluded := excludedTokenSet(excludedTokens)
	transfers := make([]model.ERC20Transfer, 0)
	for _, transfer := range transfersFromTopicLogs(trace) {
		if !isExcludedToken(transfer.Token, excluded) {
			transfers = appendUnique(transfers, transfer)
		}
	}
	for _, transfer := range transfersFromNodes(nodes) {
		if !isExcludedToken(transfer.Token, excluded) {
			transfers = appendUnique(transfers, transfer)
		}
	}
	return transfers
}

func transfersFromNodes(nodes []model.TraceNode) []model.ERC20Transfer {
	transfers := make([]model.ERC20Transfer, 0)
	for _, node := range nodes {
		collectERC20Transfers(node, "", &transfers)
	}
	return transfers
}

func transfersFromTopicLogs(trace string) []model.ERC20Transfer {
	lines := strings.Split(trace, "\n")
	targets := make([]string, 0)
	transfers := make([]model.ERC20Transfer, 0)

	for i := 0; i < len(lines); i++ {
		depth, content, ok := splitTreeLine(lines[i])
		if ok {
			if len(targets) <= depth {
				targets = append(targets, make([]string, depth-len(targets)+1)...)
			}
			for j := depth + 1; j < len(targets); j++ {
				targets[j] = ""
			}
			if target := callTarget(content); target != "" {
				targets[depth] = target
			}
		}

		if !strings.Contains(lines[i], erc20TransferTopic) {
			continue
		}
		token := nearestTokenTarget(targets, depth)
		if token == "" {
			continue
		}

		var from string
		var to string
		var amount string
		for j := i + 1; j < len(lines) && j <= i+6; j++ {
			if match := topicPattern.FindStringSubmatch(lines[j]); len(match) == 3 {
				switch match[1] {
				case "1":
					from = topicAddress(match[2])
				case "2":
					to = topicAddress(match[2])
				}
			}
			if match := dataPattern.FindStringSubmatch(lines[j]); len(match) == 2 {
				amount = hexUintToDecimal(match[1])
				break
			}
		}
		if from != "" && to != "" && amount != "" {
			transfers = append(transfers, model.ERC20Transfer{
				Token:  normalizeAddressValue(token),
				From:   from,
				To:     to,
				Amount: amount,
			})
		}
	}

	return transfers
}

func collectERC20Transfers(node model.TraceNode, token string, transfers *[]model.ERC20Transfer) {
	if node.Kind == "call" && node.Target != "" {
		token = node.Target
	}

	if node.Kind == "event" {
		if transfer, ok := parseERC20TransferEvent(token, node.Value); ok {
			*transfers = append(*transfers, transfer)
		}
	}

	for _, child := range node.Children {
		collectERC20Transfers(child, token, transfers)
	}
}

func parseERC20TransferEvent(token string, value string) (model.ERC20Transfer, bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "Transfer(") || !strings.HasSuffix(value, ")") {
		return model.ERC20Transfer{}, false
	}
	if strings.TrimSpace(token) == "" || token == "SimulateTxScript" {
		return model.ERC20Transfer{}, false
	}

	args := strings.TrimSuffix(strings.TrimPrefix(value, "Transfer("), ")")
	from, ok := eventField(args, "from")
	if !ok {
		return model.ERC20Transfer{}, false
	}
	to, ok := eventField(args, "to")
	if !ok {
		return model.ERC20Transfer{}, false
	}
	amount, ok := eventField(args, "value")
	if !ok {
		return model.ERC20Transfer{}, false
	}

	return model.ERC20Transfer{
		Token:  normalizeAddressValue(token),
		From:   normalizeAddressValue(from),
		To:     normalizeAddressValue(to),
		Amount: normalizeAmount(amount),
	}, true
}

func eventField(args string, name string) (string, bool) {
	prefix := name + ":"
	start := strings.Index(args, prefix)
	if start < 0 {
		return "", false
	}
	rest := strings.TrimSpace(args[start+len(prefix):])
	if end := strings.Index(rest, ","); end >= 0 {
		rest = rest[:end]
	}
	rest = strings.TrimSpace(rest)
	return rest, rest != ""
}

func normalizeAmount(value string) string {
	value = strings.TrimSpace(value)
	if beforeHint, _, ok := strings.Cut(value, " ["); ok {
		value = beforeHint
	}
	return strings.TrimSpace(value)
}

func splitTreeLine(line string) (int, string, bool) {
	if idx := strings.Index(line, "├─"); idx >= 0 {
		return depthFromPrefix(line[:idx]), strings.TrimSpace(line[idx+len("├─"):]), true
	}
	if idx := strings.Index(line, "└─"); idx >= 0 {
		return depthFromPrefix(line[:idx]), strings.TrimSpace(line[idx+len("└─"):]), true
	}

	content := strings.TrimSpace(line)
	if strings.HasPrefix(content, "[") {
		return 0, content, true
	}
	return 0, "", false
}

func depthFromPrefix(prefix string) int {
	depth := len([]rune(prefix)) / 4
	if depth < 1 {
		return 1
	}
	return depth
}

func callTarget(content string) string {
	if strings.HasPrefix(content, "emit ") || strings.HasPrefix(content, "←") {
		return ""
	}
	if _, afterGas, ok := parseGas(content); ok {
		content = afterGas
	}
	target, _, ok := strings.Cut(content, "::")
	if !ok {
		return ""
	}
	return strings.TrimSpace(target)
}

func parseGas(content string) (uint64, string, bool) {
	if !strings.HasPrefix(content, "[") {
		return 0, content, false
	}
	end := strings.Index(content, "]")
	if end < 0 {
		return 0, content, false
	}
	return 0, strings.TrimSpace(content[end+1:]), true
}

func nearestTokenTarget(targets []string, eventDepth int) string {
	for i := eventDepth - 1; i >= 0 && i < len(targets); i-- {
		target := targets[i]
		if target != "" && target != "VM" && target != "SimulateTxScript" {
			return target
		}
	}
	return ""
}

func topicAddress(topic string) string {
	topic = strings.TrimPrefix(topic, "0x")
	if len(topic) < 40 {
		return ""
	}
	return "0x" + strings.ToLower(topic[len(topic)-40:])
}

func hexUintToDecimal(value string) string {
	n := new(big.Int)
	if _, ok := n.SetString(strings.TrimPrefix(value, "0x"), 16); !ok {
		return ""
	}
	return n.String()
}

func normalizeAddressValue(value string) string {
	if address := addressPattern.FindString(value); address != "" {
		return strings.ToLower(address)
	}
	return strings.TrimSpace(value)
}

func excludedTokenSet(tokens []string) map[string]struct{} {
	excluded := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		if normalized := normalizeAddressValue(token); normalized != "" {
			excluded[normalized] = struct{}{}
		}
	}
	return excluded
}

func isExcludedToken(token string, excluded map[string]struct{}) bool {
	_, ok := excluded[normalizeAddressValue(token)]
	return ok
}

func appendUnique(transfers []model.ERC20Transfer, transfer model.ERC20Transfer) []model.ERC20Transfer {
	for _, existing := range transfers {
		if existing == transfer {
			return transfers
		}
	}
	return append(transfers, transfer)
}
