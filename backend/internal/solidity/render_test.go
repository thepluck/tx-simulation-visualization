package solidity

import (
	"reflect"
	"testing"

	"tx-simulation-visualization/backend/internal/model"
)

func TestForgeCompilerArgsDefaultsToCurrentCompilePath(t *testing.T) {
	got := ForgeCompilerArgs(nil)
	want := []string{"--via-ir", "--optimize=true"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ForgeCompilerArgs(nil) = %#v, want %#v", got, want)
	}
}

func TestForgeCompilerArgsUsesRequestConfig(t *testing.T) {
	viaIR := false
	optimize := false
	runs := uint32(10_000)

	got := ForgeCompilerArgs(&model.CompilerConfig{
		Use:               "0.8.30",
		Offline:           true,
		NoAutoDetect:      true,
		ViaIR:             &viaIR,
		UseLiteralContent: true,
		NoMetadata:        true,
		EVMVersion:        "cancun",
		Optimize:          &optimize,
		OptimizerRuns:     &runs,
		RevertStrings:     "debug",
	})
	want := []string{
		"--no-auto-detect",
		"--use", "0.8.30",
		"--offline",
		"--use-literal-content",
		"--no-metadata",
		"--evm-version", "cancun",
		"--optimize=false",
		"--optimizer-runs", "10000",
		"--revert-strings", "debug",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ForgeCompilerArgs(custom) = %#v, want %#v", got, want)
	}
}
