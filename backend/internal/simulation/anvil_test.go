package simulation

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
)

func TestAnvilForkRejectsOccupiedPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = listener.Close()
	}()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}

	anvil := newAnvilInstance("anvil", "127.0.0.1", portNumber)
	_, err = anvil.Fork(context.Background(), "http://127.0.0.1:8545", "1")
	if err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("Fork error = %v, want occupied port error", err)
	}
}
