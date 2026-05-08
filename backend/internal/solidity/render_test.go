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

func TestForgeCompilerArgsDoesNotDefaultSolcOrEVMVersion(t *testing.T) {
	for name, got := range map[string][]string{
		"nil":   ForgeCompilerArgs(nil),
		"empty": ForgeCompilerArgs(&model.CompilerConfig{}),
	} {
		for _, arg := range got {
			if arg == "--use" || arg == "--evm-version" {
				t.Fatalf("%s config args include version flag %#v", name, got)
			}
		}
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

func TestForgeCompilerArgsExplicitDoesNotApplyDefaults(t *testing.T) {
	got := ForgeCompilerArgsExplicit(&model.CompilerConfig{EVMVersion: "paris"})
	want := []string{"--evm-version", "paris"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ForgeCompilerArgsExplicit(custom) = %#v, want %#v", got, want)
	}
}
