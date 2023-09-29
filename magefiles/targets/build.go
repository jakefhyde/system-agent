package targets

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Build mg.Namespace

// All builds the system-agent binary.
func (Build) All(ctx context.Context) {
	mg.CtxDeps(ctx, Build.SystemAgent)
}

// SystemAgent build the system agent binary.
func (Build) SystemAgent(_ context.Context) error {
	tag, _ := os.LookupEnv("VERSION")
	tag = strings.TrimSpace(tag)

	version := "unversioned"
	if tag != "" {
		version = tag
	}

	args := []string{
		"go", "build", "-v",
	}
	args = append(args,
		"-ldflags", fmt.Sprintf("-w -s -X github.com/rancher/system-agent/pkg/versions.Version=%s", version),
		"-trimpath",
	)

	args = append(args, "-o", "bin/rancher-system-agent")

	err := sh.RunWith(map[string]string{"CGO_ENABLED": "0"}, args[0], args[1:]...)
	return err
}
