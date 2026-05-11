package fundflow

import (
	"regexp"
	"strings"
)

var addressPattern = regexp.MustCompile(`0x[0-9a-fA-F]{40}`)

func normalizeAddressValue(value string) string {
	if address := addressPattern.FindString(value); address != "" {
		return strings.ToLower(address)
	}
	return strings.TrimSpace(value)
}
