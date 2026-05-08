//go:build darwin

package httpapi

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
)

func chooseProjectDirectory(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "osascript", "-e", `POSIX path of (choose folder with prompt "Choose Foundry project")`).Output()
	if err != nil {
		return "", err
	}
	return filepath.Clean(strings.TrimSpace(string(out))), nil
}
