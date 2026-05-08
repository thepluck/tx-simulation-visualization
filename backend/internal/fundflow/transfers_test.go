package fundflow

import (
	"reflect"
	"testing"

	"tx-simulation-visualization/backend/internal/model"
)

func TestExtractERC20TransfersFromDecodedEvents(t *testing.T) {
	nodes := []model.TraceNode{
		{
			Kind:     "call",
			Target:   "SimulateTxScript",
			Function: "run",
			Children: []model.TraceNode{
				{
					Kind:     "call",
					Target:   "WETH",
					Function: "transferFrom",
					Children: []model.TraceNode{
						{
							Kind:  "event",
							Value: "Transfer(from: WETHOwner: [0x0000000000000000000000000000000000000001], to: WETHRecipient: [0x0000000000000000000000000000000000000003], value: 1000000000000000000 [1e18])",
						},
					},
				},
				{
					Kind:     "call",
					Target:   "BAYC",
					Function: "transferFrom",
					Children: []model.TraceNode{
						{
							Kind:  "event",
							Value: "Transfer(from: 0x0000000000000000000000000000000000000001, to: 0x0000000000000000000000000000000000000003, tokenId: 1)",
						},
					},
				},
			},
		},
	}

	got := ExtractERC20Transfers("", nodes, nil)
	want := []model.ERC20Transfer{
		{
			Token:  "WETH",
			From:   "0x0000000000000000000000000000000000000001",
			To:     "0x0000000000000000000000000000000000000003",
			Amount: "1000000000000000000",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractERC20Transfers() = %#v, want %#v", got, want)
	}
}

func TestExtractERC20TransfersFromTopicLogs(t *testing.T) {
	trace := `Traces:
  [259496] SimulateTxScript::run()
    ├─ [8948] 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2::transferFrom(WETHOwner: [0x0000000000000000000000000000000000000001], WETHRecipient: [0x0000000000000000000000000000000000000003], 1000000000000000000 [1e18])
    │   ├─  emit topic 0: 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef
    │   │        topic 1: 0x0000000000000000000000000000000000000000000000000000000000000001
    │   │        topic 2: 0x0000000000000000000000000000000000000000000000000000000000000003
    │   │           data: 0x0000000000000000000000000000000000000000000000000de0b6b3a7640000
    │   └─ ← [Return] 0x0000000000000000000000000000000000000000000000000000000000000001
    └─ ← [Stop]`

	got := ExtractERC20Transfers(trace, nil, nil)
	want := []model.ERC20Transfer{
		{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			From:   "0x0000000000000000000000000000000000000001",
			To:     "0x0000000000000000000000000000000000000003",
			Amount: "1000000000000000000",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractERC20Transfers() = %#v, want %#v", got, want)
	}
}

func TestExtractERC20TransfersExcludesTokens(t *testing.T) {
	trace := `Traces:
  [1] SimulateTxScript::run()
    ├─ [1] 0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D::transferFrom(0x1, 0x2, 1)
    │   ├─  emit topic 0: 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef
    │   │        topic 1: 0x0000000000000000000000000000000000000000000000000000000000000001
    │   │        topic 2: 0x0000000000000000000000000000000000000000000000000000000000000002
    │   │           data: 0x01
    │   └─ ← [Stop]`

	got := ExtractERC20Transfers(trace, nil, []string{"0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D"})
	if len(got) != 0 {
		t.Fatalf("ExtractERC20Transfers() = %#v, want none", got)
	}
}

func TestExtractERC20TransfersSkipsRootScriptEvents(t *testing.T) {
	nodes := []model.TraceNode{
		{
			Kind:   "call",
			Target: "SimulateTxScript",
			Children: []model.TraceNode{
				{
					Kind:  "event",
					Value: "Transfer(from: 0x0000000000000000000000000000000000000001, to: 0x0000000000000000000000000000000000000003, value: 100)",
				},
			},
		},
	}

	if got := ExtractERC20Transfers("", nodes, nil); len(got) != 0 {
		t.Fatalf("ExtractERC20Transfers() = %#v, want none", got)
	}
}
