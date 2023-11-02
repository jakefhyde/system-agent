package targets

import (
	"context"

	"github.com/magefile/mage/mg"
)

// Default runs all generate and build targets.
func Default(ctx context.Context) {
	mg.SerialCtxDeps(ctx, Build.All)
}
