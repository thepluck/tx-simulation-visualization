package prices

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRPCMetadataProviderFetch(t *testing.T) {
	token := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		var call struct {
			To   string `json:"to"`
			Data string `json:"data"`
		}
		if err := json.Unmarshal(payload.Params[0], &call); err != nil {
			t.Fatal(err)
		}
		if !strings.EqualFold(call.To, token) {
			t.Fatalf("call target = %s, want %s", call.To, token)
		}

		switch call.Data {
		case erc20DecimalsSelector:
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000012"}`))
		case erc20SymbolSelector:
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":"%s"}`, abiString("WETH"))
		default:
			t.Fatalf("unexpected data: %s", call.Data)
		}
	}))
	defer server.Close()

	provider := RPCMetadataProvider{RPCURLs: map[string]string{"mainnet": server.URL}}
	got, err := provider.Fetch(context.Background(), "mainnet", []string{token})
	if err != nil {
		t.Fatal(err)
	}
	price := got[token]
	if price.Decimals != 18 || !price.HasDecimals || price.Symbol != "WETH" {
		t.Fatalf("metadata = %#v", price)
	}
	wantLogo := "https://raw.githubusercontent.com/trustwallet/assets/master/blockchains/ethereum/assets/0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2/logo.png"
	if price.LogoURL != wantLogo {
		t.Fatalf("logo = %q, want %q", price.LogoURL, wantLogo)
	}
}

func TestParseBytes32StringResult(t *testing.T) {
	raw := hex.EncodeToString(append([]byte("MKR"), make([]byte, 29)...))
	got, ok := parseStringResult("0x" + raw)
	if !ok || got != "MKR" {
		t.Fatalf("parseStringResult = %q %v, want MKR true", got, ok)
	}
}

func abiString(value string) string {
	encoded := hex.EncodeToString([]byte(value))
	padding := strings.Repeat("0", (32-len(value))*2)
	return "0x" +
		abiUintWord(32) +
		abiUintWord(len(value)) +
		encoded + padding
}

func abiUintWord(value int) string {
	return fmt.Sprintf("%064x", value)
}
