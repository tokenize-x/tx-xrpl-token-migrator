package main

import (
	"github.com/CoreumFoundation/crust/build"
	selfBuild "github.com/CoreumFoundation/xrpl-bridge/build"
)

func main() {
	build.Main(selfBuild.Commands)
}
