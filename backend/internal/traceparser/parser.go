package traceparser

import (
	"strconv"
	"strings"
	"unicode"

	"tx-simulation-visualization/backend/internal/model"
)

type workNode struct {
	item     model.TraceNode
	children []*workNode
}

func Parse(trace string) []model.TraceNode {
	var roots []*workNode
	var stack []*workNode

	for _, line := range strings.Split(trace, "\n") {
		depth, content, ok := splitTraceLine(line)
		if !ok {
			continue
		}

		node := &workNode{item: parseContent(content, depth)}
		if depth <= 0 || len(stack) < depth || stack[depth-1] == nil {
			roots = append(roots, node)
		} else {
			parent := stack[depth-1]
			parent.children = append(parent.children, node)
		}

		if len(stack) <= depth {
			stack = append(stack, make([]*workNode, depth-len(stack)+1)...)
		}
		stack[depth] = node
		for i := depth + 1; i < len(stack); i++ {
			stack[i] = nil
		}
	}

	out := make([]model.TraceNode, 0, len(roots))
	for _, root := range roots {
		out = append(out, root.toModel())
	}
	return out
}

func (n *workNode) toModel() model.TraceNode {
	item := n.item
	if len(n.children) > 0 {
		item.Children = make([]model.TraceNode, 0, len(n.children))
		for _, child := range n.children {
			item.Children = append(item.Children, child.toModel())
		}
	}
	return item
}

func splitTraceLine(line string) (int, string, bool) {
	if strings.TrimSpace(line) == "" {
		return 0, "", false
	}
	if strings.TrimSpace(line) == "Traces:" || strings.TrimSpace(line) == "Trace:" {
		return 0, "", false
	}

	if idx := strings.Index(line, "├─"); idx >= 0 {
		return depthFromPrefix(line[:idx]), strings.TrimSpace(line[idx+len("├─"):]), true
	}
	if idx := strings.Index(line, "└─"); idx >= 0 {
		return depthFromPrefix(line[:idx]), strings.TrimSpace(line[idx+len("└─"):]), true
	}

	content := strings.TrimSpace(line)
	if strings.HasPrefix(content, "[") || strings.HasPrefix(content, "←") || strings.HasPrefix(content, "emit ") {
		return 0, content, true
	}
	if strings.HasPrefix(content, "Error:") {
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

func parseContent(content string, depth int) model.TraceNode {
	node := model.TraceNode{
		Raw:   content,
		Kind:  "unknown",
		Depth: depth,
	}

	if strings.HasPrefix(content, "←") {
		parseResult(content, &node)
		return node
	}
	if strings.HasPrefix(content, "emit ") {
		node.Kind = "event"
		node.Value = strings.TrimSpace(strings.TrimPrefix(content, "emit "))
		return node
	}
	if strings.HasPrefix(content, "Error:") {
		node.Kind = "error"
		node.Value = content
		return node
	}

	node.Kind = "call"
	rest := content
	if gas, afterGas, ok := parseGas(rest); ok {
		node.Gas = gas
		rest = afterGas
	}
	if callType, afterType, ok := parseCallType(rest); ok {
		node.CallType = callType
		rest = afterType
	}
	parseCall(rest, &node)
	return node
}

func parseResult(content string, node *model.TraceNode) {
	node.Kind = "result"

	rest := strings.TrimSpace(strings.TrimPrefix(content, "←"))
	if !strings.HasPrefix(rest, "[") {
		node.Value = rest
		return
	}

	end := strings.Index(rest, "]")
	if end < 0 {
		node.Value = rest
		return
	}

	node.ResultType = rest[1:end]
	node.Kind = strings.ToLower(node.ResultType)
	node.Value = strings.TrimSpace(rest[end+1:])
}

func parseGas(content string) (uint64, string, bool) {
	if !strings.HasPrefix(content, "[") {
		return 0, content, false
	}
	end := strings.Index(content, "]")
	if end < 0 {
		return 0, content, false
	}

	value, err := strconv.ParseUint(content[1:end], 10, 64)
	if err != nil {
		return 0, content, false
	}
	return value, strings.TrimSpace(content[end+1:]), true
}

func parseCallType(content string) (string, string, bool) {
	trimmed := strings.TrimSpace(content)
	if !strings.HasSuffix(trimmed, "]") {
		return "", content, false
	}

	start := strings.LastIndex(trimmed, "[")
	if start < 0 {
		return "", content, false
	}

	value := trimmed[start+1 : len(trimmed)-1]
	if value == "" || allDigits(value) {
		return "", content, false
	}

	return value, strings.TrimSpace(trimmed[:start]), true
}

func parseCall(content string, node *model.TraceNode) {
	target, rest, ok := strings.Cut(content, "::")
	if !ok {
		node.Value = content
		return
	}

	node.Target = strings.TrimSpace(target)
	paren := strings.Index(rest, "(")
	if paren < 0 {
		node.Function = strings.TrimSpace(rest)
		return
	}

	node.Function = strings.TrimSpace(rest[:paren])
	if end := strings.LastIndex(rest, ")"); end > paren {
		node.Arguments = rest[paren+1 : end]
	} else {
		node.Arguments = rest[paren+1:]
	}
}

func allDigits(value string) bool {
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return value != ""
}
