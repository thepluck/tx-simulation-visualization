package traceparser

import (
	"strings"
	"testing"
)

func TestParseOutputForgeJSONTrace(t *testing.T) {
	output := `2026-05-11T10:28:35Z WARN evm::traces::external: could not get info
{
  "returns": {},
  "success": true,
  "raw_logs": [],
  "traces": [
    [
      "Execution",
      {
        "arena": [
          {
            "parent": null,
            "children": [1],
            "idx": 0,
            "trace": {
              "address": "0x0000000000000000000000000000000000000001",
              "kind": "CALL"
            },
            "logs": [],
            "ordering": [{"Call": 0}]
          },
          {
            "parent": 0,
            "children": [],
            "idx": 1,
            "trace": {
              "address": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
              "kind": "CALL"
            },
            "logs": [
              {
                "raw_log": {
                  "topics": [
                    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
                    "0x0000000000000000000000000000000000000000000000000000000000000001",
                    "0x0000000000000000000000000000000000000000000000000000000000000003"
                  ],
                  "data": "0x0000000000000000000000000000000000000000000000000000000000000001"
                },
                "decoded": {"name": "Transfer"}
              }
            ],
            "ordering": [{"Log": 0}]
          }
        ]
      }
    ]
  ],
  "gas_used": 12345,
  "labeled_addresses": {},
  "returned": "0x",
  "address": null
}`

	parsed, err := ParseOutput(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(parsed.Trace, `"traces"`) {
		t.Fatalf("trace should preserve formatted forge JSON, got:\n%s", parsed.Trace)
	}
	if len(parsed.ERC20Transfers) != 1 {
		t.Fatalf("erc20 transfers = %#v, want one", parsed.ERC20Transfers)
	}
	transfer := parsed.ERC20Transfers[0]
	if transfer.Token != "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2" {
		t.Fatalf("transfer token = %s", transfer.Token)
	}
	if transfer.From != "0x0000000000000000000000000000000000000001" || transfer.To != "0x0000000000000000000000000000000000000003" || transfer.Amount != "1" {
		t.Fatalf("unexpected transfer: %#v", transfer)
	}
}

func TestParseOutputERC20TransferUsesLastParentZeroNodeAndNearestCallAncestor(t *testing.T) {
	output := `{
  "returns": {},
  "success": true,
  "raw_logs": [],
  "traces": [
    [
      "Execution",
      {
        "arena": [
          {
            "parent": null,
            "children": [1, 2],
            "idx": 0,
            "trace": {
              "address": "0x0000000000000000000000000000000000000001",
              "kind": "CALL"
            },
            "logs": [],
            "ordering": [{"Call": 0}, {"Call": 1}]
          },
          {
            "parent": 0,
            "children": [],
            "idx": 1,
            "trace": {
              "address": "0x1111111111111111111111111111111111111111",
              "kind": "CALL"
            },
            "logs": [
              {
                "address": "0x1111111111111111111111111111111111111111",
                "topics": [
                  "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
                  "0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                  "0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
                ],
                "data": "0x0000000000000000000000000000000000000000000000000000000000000001"
              }
            ],
            "ordering": [{"Log": 0}]
          },
          {
            "parent": 0,
            "children": [3],
            "idx": 2,
            "trace": {
              "address": "0x2222222222222222222222222222222222222222",
              "kind": "CALL"
            },
            "logs": [],
            "ordering": [{"Call": 0}]
          },
          {
            "parent": 2,
            "children": [],
            "idx": 3,
            "trace": {
              "address": "0x3333333333333333333333333333333333333333",
              "kind": "DELEGATECALL"
            },
            "logs": [
              {
                "address": "0x3333333333333333333333333333333333333333",
                "topics": [
                  "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
                  "0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                  "0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
                ],
                "data": "0x0000000000000000000000000000000000000000000000000000000000000002"
              }
            ],
            "ordering": [{"Log": 0}]
          }
        ]
      }
    ]
  ],
  "gas_used": 1000,
  "labeled_addresses": {},
  "returned": "0x",
  "address": null
}`

	parsed, err := ParseOutput(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.ERC20Transfers) != 1 {
		t.Fatalf("erc20 transfers = %#v, want one", parsed.ERC20Transfers)
	}
	transfer := parsed.ERC20Transfers[0]
	if transfer.Token != "0x2222222222222222222222222222222222222222" {
		t.Fatalf("transfer token = %s, want proxy CALL ancestor", transfer.Token)
	}
	if transfer.Amount != "2" {
		t.Fatalf("transfer amount = %s, want 2", transfer.Amount)
	}
}

func TestParseOutputRejectsTextTrace(t *testing.T) {
	_, err := ParseOutput(`Traces:
  [252850] SimulateTxScript::run()
    └─ ← [Return]`)
	if err == nil {
		t.Fatal("expected text trace to fail JSON parsing")
	}
}
