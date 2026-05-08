package forge

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunnerReturnsContextErrorWhenCommandIsKilledByTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	result := Runner{
		Bin:      "sh",
		RepoRoot: t.TempDir(),
	}.Run(ctx, "-c", "sleep 1")

	if !errors.Is(result.Err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context deadline exceeded", result.Err)
	}
	if result.ExitCode != -1 {
		t.Fatalf("exit code = %d, want -1", result.ExitCode)
	}
	if status := StatusFromCommandError(result.Err); status != 504 {
		t.Fatalf("status = %d, want 504", status)
	}
}
