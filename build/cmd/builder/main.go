package main

import (
	"github.com/tokenize-x/tx-crust/build"
	selfBuild "github.com/tokenize-x/tx-xrpl-token-migrator/build"
)

func main() {
	build.Main(selfBuild.Commands)
}
