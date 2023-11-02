package targets

import (
	"context"
	"os"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Package mg.Namespace

func (Package) All(ctx context.Context) {
	mg.CtxDeps(ctx, Package.Image)
	mg.CtxDeps(ctx, Package.Suc)
}

func (Package) Prepare(ctx context.Context) error {
	mg.SerialCtxDeps(ctx, Build.All)

	err := sh.RunV("mkdir", "-p", "dist/artifacts")
	if err != nil {
		return err
	}

	err = sh.RunV("cp", "bin/rancher-system-agent-linux", "dist/artifacts/rancher-system-agent-amd64") // todo: ARCH
	if err != nil {
		return err
	}

	return nil
}

func (Package) Image(ctx context.Context) error {
	mg.SerialCtxDeps(ctx, Build.All)
	mg.SerialCtxDeps(ctx, Package.Prepare)

	err := sh.RunV("docker",
		"build",
		"-f", "package/Dockerfile",
		"-t", "rancher/system-agent:dev", // todo: get tag
		"--build-arg", "OS=linux",
		".")
	if err != nil {
		return err
	}

	return nil
}

func (Package) Suc(ctx context.Context) error {
	mg.SerialCtxDeps(ctx, Package.Image)

	err := sh.RunV("cp", "install.sh", "dist/artifacts/install.sh")
	if err != nil {
		return err
	}

	err = sh.RunV("cp", "system-agent-uninstall.sh", "dist/artifacts/system-agent-uninstall.sh")
	if err != nil {
		return err
	}

	err = sh.RunWithV(map[string]string{
		"ARCH": os.Getenv("ARCH"),
	},
		"docker",
		"build",
		"-f", "package/Dockerfile.suc",
		"-t", "rancher/system-agent:dev-suc", // todo: get tag
		"--build-arg", "OS=linux",
		"--build-arg", "ARCH="+os.Getenv("ARCH"),
		".")
	if err != nil {
		return err
	}

	return nil
}
