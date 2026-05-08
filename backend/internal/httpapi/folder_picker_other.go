//go:build !darwin

package httpapi

import (
	"context"
	"errors"
)

func chooseProjectDirectory(_ context.Context) (string, error) {
	return "", errors.New("native project browsing is only supported on macOS")
}
