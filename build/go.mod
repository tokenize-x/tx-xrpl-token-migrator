module github.com/tokenize-x/tx-xrpl-token-migrator/build

go 1.23.3

// Crust replacements
replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

require github.com/CoreumFoundation/crust v0.0.0-20250404130536-23de310e6eb8

replace github.com/CoreumFoundation/crust => github.com/tokenize-x/tx-crust v0.0.0-20250404130536-23de310e6eb8

require (
	github.com/CoreumFoundation/coreum-tools v0.4.1-0.20241202115740-dbc6962a4d0a // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/samber/lo v1.49.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/mod v0.23.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)
