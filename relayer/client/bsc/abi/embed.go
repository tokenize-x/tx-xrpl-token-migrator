// Package abi provides ABI bindings and artifacts for BSC smart contracts.
package abi

import "embed"

// ArtifactFiles contains embedded contract artifact JSON files.
//
//go:embed TXToken.json TXBridge.json
var ArtifactFiles embed.FS
