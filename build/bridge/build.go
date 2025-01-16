package bridge

import (
	"context"

	"github.com/CoreumFoundation/crust/build/golang"
	"github.com/CoreumFoundation/crust/build/types"
)

// Lint lints coreum repo.
func Lint(ctx context.Context, deps types.DepsFunc) error {
	return golang.Lint(ctx, deps)
}
