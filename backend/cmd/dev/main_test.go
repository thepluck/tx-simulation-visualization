package main

import (
	"os"
	"testing"
)

func TestSignalExitStatus(t *testing.T) {
	if got := signalExitStatus(os.Interrupt); got != 130 {
		t.Fatalf("interrupt status = %d, want 130", got)
	}
	if got := signalExitStatus(os.Kill); got != 137 {
		t.Fatalf("kill status = %d, want 137", got)
	}
}
