module github.com/tokenize-x/tx-xrpl-token-migrator/build

go 1.24

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

require github.com/tokenize-x/tx-crust v0.0.0-20260129072642-443b98cfb118

// Use local tx-crust with TXCrustModule fix
replace github.com/tokenize-x/tx-crust => /Users/can/Projects/Coreum-repos/tx-crust

require (
	github.com/pkg/errors v0.9.1 // indirect
	github.com/samber/lo v1.49.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/tokenize-x/tx-tools v0.0.0-20251006151522-f6df01ec2033 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/mod v0.23.0 // indirect
	golang.org/x/text v0.22.0 // indirect
)
