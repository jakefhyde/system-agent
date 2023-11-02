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

func (Build) Prepare(_ context.Context) error {
	err := sh.RunV("mkdir", "-p", "bin")
	if err != nil {
		return err
	}
	return nil
}

// SystemAgent build the system agent binary.
func (Build) SystemAgent(ctx context.Context) error {
	mg.CtxDeps(ctx, Build.Prepare)

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

	args = append(args, "-o", "./bin/rancher-system-agent-linux") // todo: ARCH

	err := sh.RunWith(map[string]string{"CGO_ENABLED": "0"}, args[0], args[1:]...)
	if err != nil {
		return err
	}

	sh.Run("pwd")
	sh.Run("ls -la ./bin")

	return nil
}
