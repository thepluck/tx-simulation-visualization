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

func TestParseOutputERC20TransferUsesLastParentZeroNodeAndNonDelegateContext(t *testing.T) {
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

func TestParseOutputERC20TransferUsesCreateFrameAddress(t *testing.T) {
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
            "children": [1],
            "idx": 0,
            "trace": {
              "address": "0x0000000000000000000000000000000000000001",
              "kind": "CALL"
            },
            "logs": []
          },
          {
            "parent": 0,
            "children": [],
            "idx": 1,
            "trace": {
              "address": "0x4444444444444444444444444444444444444444",
              "kind": "CREATE"
            },
            "logs": [
              {
                "address": "0x4444444444444444444444444444444444444444",
                "topics": [
                  "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
                  "0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                  "0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
                ],
                "data": "0x0000000000000000000000000000000000000000000000000000000000000003"
              }
            ]
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
	if transfer.Token != "0x4444444444444444444444444444444444444444" {
		t.Fatalf("transfer token = %s, want create frame address", transfer.Token)
	}
	if transfer.Amount != "3" {
		t.Fatalf("transfer amount = %s, want 3", transfer.Amount)
	}
}

func TestParseOutputSkipsWholeRevertedTransaction(t *testing.T) {
	output := `{
  "returns": {},
  "success": false,
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
              "kind": "CALL",
              "success": false,
              "status": "Revert"
            },
            "logs": []
          },
          {
            "parent": 0,
            "children": [],
            "idx": 1,
            "trace": {
              "address": "0x5555555555555555555555555555555555555555",
              "kind": "CALL",
              "success": false,
              "status": "Revert"
            },
            "logs": [
              {
                "address": "0x5555555555555555555555555555555555555555",
                "topics": [
                  "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
                  "0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                  "0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
                ],
                "data": "0x0000000000000000000000000000000000000000000000000000000000000004"
              }
            ]
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
	if len(parsed.ERC20Transfers) != 0 {
		t.Fatalf("erc20 transfers = %#v, want none for reverted transaction", parsed.ERC20Transfers)
	}
}

func TestParseOutputSkipsRevertedBranch(t *testing.T) {
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
            "children": [1],
            "idx": 0,
            "trace": {
              "address": "0x0000000000000000000000000000000000000001",
              "kind": "CALL",
              "success": true,
              "status": "Return"
            },
            "logs": []
          },
          {
            "parent": 0,
            "children": [2, 3],
            "idx": 1,
            "trace": {
              "address": "0x6666666666666666666666666666666666666666",
              "kind": "CALL",
              "success": true,
              "status": "Return"
            },
            "logs": []
          },
          {
            "parent": 1,
            "children": [4],
            "idx": 2,
            "trace": {
              "address": "0x7777777777777777777777777777777777777777",
              "kind": "CALL",
              "success": false,
              "status": "Revert"
            },
            "logs": [
              {
                "address": "0x7777777777777777777777777777777777777777",
                "topics": [
                  "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
                  "0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                  "0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
                ],
                "data": "0x0000000000000000000000000000000000000000000000000000000000000005"
              }
            ]
          },
          {
            "parent": 1,
            "children": [],
            "idx": 3,
            "trace": {
              "address": "0x8888888888888888888888888888888888888888",
              "kind": "CALL",
              "success": true,
              "status": "Return"
            },
            "logs": [
              {
                "address": "0x8888888888888888888888888888888888888888",
                "topics": [
                  "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
                  "0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                  "0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
                ],
                "data": "0x0000000000000000000000000000000000000000000000000000000000000006"
              }
            ]
          },
          {
            "parent": 2,
            "children": [],
            "idx": 4,
            "trace": {
              "address": "0x9999999999999999999999999999999999999999",
              "kind": "CALL",
              "success": true,
              "status": "Return"
            },
            "logs": [
              {
                "address": "0x9999999999999999999999999999999999999999",
                "topics": [
                  "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
                  "0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                  "0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
                ],
                "data": "0x0000000000000000000000000000000000000000000000000000000000000007"
              }
            ]
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
		t.Fatalf("erc20 transfers = %#v, want one from non-reverted sibling only", parsed.ERC20Transfers)
	}
	transfer := parsed.ERC20Transfers[0]
	if transfer.Token != "0x8888888888888888888888888888888888888888" || transfer.Amount != "6" {
		t.Fatalf("unexpected transfer: %#v", transfer)
	}
}

func TestParseOutputPreservesDuplicateTransfers(t *testing.T) {
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
            "children": [1],
            "idx": 0,
            "trace": {
              "address": "0x0000000000000000000000000000000000000001",
              "kind": "CALL"
            },
            "logs": []
          },
          {
            "parent": 0,
            "children": [],
            "idx": 1,
            "trace": {
              "address": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
              "kind": "CALL"
            },
            "logs": [
              {
                "address": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                "topics": [
                  "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
                  "0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
                  "0x000000000000000000000000cccccccccccccccccccccccccccccccccccccccc"
                ],
                "data": "0x0000000000000000000000000000000000000000000000000000000000000008"
              },
              {
                "address": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                "topics": [
                  "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
                  "0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
                  "0x000000000000000000000000cccccccccccccccccccccccccccccccccccccccc"
                ],
                "data": "0x0000000000000000000000000000000000000000000000000000000000000008"
              }
            ]
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
	if len(parsed.ERC20Transfers) != 2 {
		t.Fatalf("erc20 transfers = %#v, want duplicate logs preserved", parsed.ERC20Transfers)
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
