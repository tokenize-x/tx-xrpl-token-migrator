module github.com/CoreumFoundation/xrpl-bridge/build

go 1.22.0

toolchain go1.22.10

// Crust replacements
replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

require github.com/CoreumFoundation/crust v0.0.0-20241225103102-0cb70152a971

require (
	github.com/CoreumFoundation/coreum-tools v0.4.1-0.20240321120602-0a9c50facc68 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/samber/lo v1.39.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/exp v0.0.0-20241204233417-43b7b7cde48d // indirect
	golang.org/x/mod v0.22.0 // indirect
)
