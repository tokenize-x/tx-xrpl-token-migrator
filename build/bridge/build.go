package bridge

import (
	"context"

	"github.com/tokenize-x/tx-crust/build/golang"
	"github.com/tokenize-x/tx-crust/build/types"
)

// Lint lints tx-xrpl-token-migrator repo.
func Lint(ctx context.Context, deps types.DepsFunc) error {
	return golang.Lint(ctx, deps)
}
