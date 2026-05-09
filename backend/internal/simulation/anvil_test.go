package simulation

import (
	"context"
	"errors"
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

func TestProcessExitedLockedKeepsExitState(t *testing.T) {
	wantErr := errors.New("anvil exited")
	anvil := &anvilInstance{
		done: make(chan error, 1),
	}
	anvil.done <- wantErr

	firstErr, firstExited := anvil.processExitedLocked()
	secondErr, secondExited := anvil.processExitedLocked()

	if !firstExited || !secondExited {
		t.Fatalf("processExitedLocked exited = %v, %v; want true twice", firstExited, secondExited)
	}
	if !errors.Is(firstErr, wantErr) || !errors.Is(secondErr, wantErr) {
		t.Fatalf("processExitedLocked errors = %v, %v; want %v", firstErr, secondErr, wantErr)
	}
}

func TestAnvilExitErrorHandlesCleanExit(t *testing.T) {
	anvil := &anvilInstance{}

	err := anvil.anvilExitError(nil)
	if err == nil || !strings.Contains(err.Error(), "clean exit") {
		t.Fatalf("anvilExitError(nil) = %v, want clean exit error", err)
	}
}
