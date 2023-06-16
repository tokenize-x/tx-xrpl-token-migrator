package contractembed

import (
	_ "embed"
)

// Bytecode is compiled smart contract bytecode.
//
//go:embed artifacts/threshold_bank_send.wasm
var Bytecode []byte
