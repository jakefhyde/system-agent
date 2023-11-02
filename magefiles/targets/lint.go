package targets

import (
	"context"
	"os/exec"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Lint runs golangci-lint with the project config.
func Lint(_ context.Context) {
	mg.Deps(golangciLint)
}

func golangciLint() error {
	if lint, err := exec.LookPath("golangci-lint"); err == nil {
		return sh.RunV(lint, "run")
	}
	return sh.RunV(mg.GoCmd(), "run", "github.com/golangci/golangci-lint/cmd/golangci-lint@latest", "run")
}
