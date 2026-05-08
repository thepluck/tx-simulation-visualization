package fundflow

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"tx-simulation-visualization/backend/internal/model"
)

const (
	recordedLogPrefix = "TXSIM_LOG|"
)

var addressPattern = regexp.MustCompile(`0x[0-9a-fA-F]{40}`)

type recordedLog struct {
	emitter string
	topics  []string
	data    string
}

func ExtractERC20Transfers(output string) []model.ERC20Transfer {
	transfers := make([]model.ERC20Transfer, 0)
	for _, line := range strings.Split(output, "\n") {
		log, ok := parseRecordedLogLine(line)
		if !ok {
			continue
		}
		transfer, ok := erc20TransferFromRecordedLog(log)
		if ok {
			transfers = appendUnique(transfers, transfer)
		}
	}
	return transfers
}

func parseRecordedLogLine(line string) (recordedLog, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, recordedLogPrefix) {
		return recordedLog{}, false
	}

	parts := strings.Split(line, "|")
	if len(parts) < 5 || parts[0] != strings.TrimSuffix(recordedLogPrefix, "|") {
		return recordedLog{}, false
	}

	topicCount, err := strconv.Atoi(parts[2])
	if err != nil || topicCount < 0 || len(parts) != topicCount+4 {
		return recordedLog{}, false
	}

	return recordedLog{
		emitter: normalizeHex(parts[1]),
		topics:  normalizeHexSlice(parts[3 : 3+topicCount]),
		data:    normalizeHex(parts[len(parts)-1]),
	}, true
}

func erc20TransferFromRecordedLog(log recordedLog) (model.ERC20Transfer, bool) {
	if len(log.topics) < 3 {
		return model.ERC20Transfer{}, false
	}

	amount := hexUintToDecimal(log.data)
	from := topicAddress(log.topics[1])
	to := topicAddress(log.topics[2])
	if log.emitter == "" || from == "" || to == "" || amount == "" {
		return model.ERC20Transfer{}, false
	}

	return model.ERC20Transfer{
		Token:  log.emitter,
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

func normalizeAddressValue(value string) string {
	if address := addressPattern.FindString(value); address != "" {
		return strings.ToLower(address)
	}
	return strings.TrimSpace(value)
}

func normalizeHexSlice(values []string) []string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = normalizeHex(value)
	}
	return out
}

func topicAddress(topic string) string {
	topic = strings.TrimPrefix(normalizeHex(topic), "0x")
	if len(topic) < 40 {
		return ""
	}
	return "0x" + topic[len(topic)-40:]
}

func hexUintToDecimal(value string) string {
	n := new(big.Int)
	if _, ok := n.SetString(strings.TrimPrefix(normalizeHex(value), "0x"), 16); !ok {
		return ""
	}
	return n.String()
}

func appendUnique(transfers []model.ERC20Transfer, transfer model.ERC20Transfer) []model.ERC20Transfer {
	for _, existing := range transfers {
		if existing == transfer {
			return transfers
		}
	}
	return append(transfers, transfer)
}
