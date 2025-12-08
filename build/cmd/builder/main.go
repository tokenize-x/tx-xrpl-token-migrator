package main

import (
	selfBuild "github.com/tokenize-x/tx-xrpl-token-migrator/build"

	"github.com/CoreumFoundation/crust/build"
)

func main() {
	build.Main(selfBuild.Commands)
}
