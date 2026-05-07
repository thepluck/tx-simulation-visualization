package traceparser

import "testing"

func TestParseForgeTrace(t *testing.T) {
	trace := `Traces:
  [252850] SimulateTxScript::run([], [], [], 0x, 0x0000000000000000000000000000000000000002, 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2, 0x23b872dd)
    ├─ [2534] 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2::balanceOf(0x0000000000000000000000000000000000000001) [staticcall]
    │   └─ ← [Return] 1000000000000000000 [1e18]
    ├─ emit Transfer(from: 0x0000000000000000000000000000000000000001, to: 0x0000000000000000000000000000000000000003, value: 1000000000000000000 [1e18])
    └─ ← [Return]`

	nodes := Parse(trace)
	if len(nodes) != 1 {
		t.Fatalf("len(nodes) = %d, want 1", len(nodes))
	}

	root := nodes[0]
	if root.Kind != "call" || root.Gas != 252850 || root.Target != "SimulateTxScript" || root.Function != "run" {
		t.Fatalf("unexpected root: %#v", root)
	}
	if len(root.Children) != 3 {
		t.Fatalf("len(root.Children) = %d, want 3", len(root.Children))
	}

	call := root.Children[0]
	if call.Kind != "call" || call.CallType != "staticcall" || call.Function != "balanceOf" {
		t.Fatalf("unexpected call child: %#v", call)
	}
	if len(call.Children) != 1 || call.Children[0].Kind != "return" || call.Children[0].Value == "" {
		t.Fatalf("unexpected return child: %#v", call.Children)
	}

	event := root.Children[1]
	if event.Kind != "event" || event.Value == "" {
		t.Fatalf("unexpected event child: %#v", event)
	}
}

func TestParseErrorTrace(t *testing.T) {
	nodes := Parse("Error: something failed")
	if len(nodes) != 1 || nodes[0].Kind != "error" || nodes[0].Value != "Error: something failed" {
		t.Fatalf("unexpected error nodes: %#v", nodes)
	}
}
