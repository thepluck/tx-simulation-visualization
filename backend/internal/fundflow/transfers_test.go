package fundflow

import (
	"reflect"
	"testing"

	"tx-simulation-visualization/backend/internal/model"
)

func TestExtractERC20TransfersFromRecordedLogs(t *testing.T) {
	output := `Traces:
  [1] SimulateTxScript::run()

== Logs ==
  TXSIM_LOG|0x4200000000000000000000000000000000000006|3|0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef|0x000000000000000000000000dead00000000000000000000000000000000dead|0x0000000000000000000000008f10b468b06c6fd214b65f87778827f7d113f996|0x00000000000000000000000000000000000000000000000006f05b59d3b20000
  TXSIM_LOG|0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913|3|0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef|0x0000000000000000000000008f10b468b06c6fd214b65f87778827f7d113f996|0x0000000000000000000000001fd108cf42a59c635bd4703b8dbc8a741ff834be|0x00000000000000000000000000000000000000000000000000000000453a970c`

	got := ExtractERC20Transfers(output)
	want := []model.ERC20Transfer{
		{
			Token:  "0x4200000000000000000000000000000000000006",
			From:   "0xdead00000000000000000000000000000000dead",
			To:     "0x8f10b468b06c6fd214b65f87778827f7d113f996",
			Amount: "500000000000000000",
		},
		{
			Token:  "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
			From:   "0x8f10b468b06c6fd214b65f87778827f7d113f996",
			To:     "0x1fd108cf42a59c635bd4703b8dbc8a741ff834be",
			Amount: "1161467660",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractERC20Transfers() = %#v, want %#v", got, want)
	}
}

func TestExtractERC20TransfersRequiresRecordedLogMarker(t *testing.T) {
	output := `Traces:
  [1] SimulateTxScript::run()
    ├─ emit Transfer(from: 0x0000000000000000000000000000000000000001, to: 0x0000000000000000000000000000000000000002, amount: 100)`

	if got := ExtractERC20Transfers(output); len(got) != 0 {
		t.Fatalf("ExtractERC20Transfers() = %#v, want none", got)
	}
}

func TestExtractERC20TransfersDeduplicatesRepeatedMarkers(t *testing.T) {
	line := "TXSIM_LOG|0x4200000000000000000000000000000000000006|3|0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef|0x000000000000000000000000dead00000000000000000000000000000000dead|0x0000000000000000000000008f10b468b06c6fd214b65f87778827f7d113f996|0x00000000000000000000000000000000000000000000000006f05b59d3b20000"
	output := line + "\n" + line

	got := ExtractERC20Transfers(output)
	if len(got) != 1 {
		t.Fatalf("ExtractERC20Transfers() = %#v, want one transfer", got)
	}
}
