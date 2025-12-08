package build

import (
	"github.com/tokenize-x/tx-xrpl-token-migrator/build/bridge"

	"github.com/CoreumFoundation/crust/build/crust"
	"github.com/CoreumFoundation/crust/build/types"
)

// Commands is a definition of commands available in build system.
var Commands = map[string]types.Command{
	"build/me": {Fn: crust.BuildBuilder, Description: "Builds the builder"},
	"lint":     {Fn: bridge.Lint, Description: "Lints code"},
}
