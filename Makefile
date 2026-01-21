IMPORT_PREFIX=github.com/CoreumFoundation
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
CONTRACT_DIR:=$(ROOT_DIR)/contract
SCAN_FILES := $(shell find . -type f -name '*.go' -not -name '*mock.go' -not -name '*_gen.go' -not -path "*/vendor/*")
TX_BUILDER:=$(ROOT_DIR)/../coreum/bin/coreum-builder
BUILDER = ./bin/tx-xrpl-token-migrator-builder

###############################################################################
###                                 Build                                  ###
###############################################################################


###############################################################################
###                                Build flags                              ###
###############################################################################

BUILD_VERSION := $(BUILD_VERSION)
ifeq ($(BUILD_VERSION),)
    BUILD_VERSION = $(shell echo $(shell git describe --tags) | sed 's/^v//')
endif

.PHONY: build
build:
	CGO_ENABLED=0 go build --trimpath -mod=readonly -ldflags '-X main.BuildVersion="${BUILD_VERSION} -extldflags=-static' -o artifacts/relayer ./relayer/cmd

.PHONY: build-in-docker
build-in-docker:
	# Enable BuildKit and pass SSH keys as secrets so private modules can be fetched during docker build
	# SSH keys should be at ~/.ssh/tx-{chain,tools,crust}-deploy-key
	DOCKER_BUILDKIT=1 docker build \
		--secret id=tx-chain-key,src=$(HOME)/.ssh/tx-chain-deploy-key \
		--secret id=tx-tools-key,src=$(HOME)/.ssh/tx-tools-deploy-key \
		--secret id=tx-crust-key,src=$(HOME)/.ssh/tx-crust-deploy-key \
		--build-arg BUILD_VERSION=$(BUILD_VERSION) \
		. -t tx-xrpl-token-migrator-builder
	mkdir -p artifacts
	docker run --rm --entrypoint cat tx-xrpl-token-migrator-builder /code/artifacts/relayer > artifacts/relayer

###############################################################################
###                               Development                               ###
###############################################################################

.PHONY: all
all: fmt lint test build-contract test-integration

.PHONY: test
test:
	@go test -v -mod=readonly -parallel=4 ./...

.PHONY: test-integration
test-integration: test-integration-xrpl

.PHONY: test-integration-xrpl
test-integration-xrpl:
	@go test -v --tags=integrationtests -mod=readonly -parallel=4 -run 'Test(XRPL|Contract|Migration|Config)' ./integration-tests -timeout 300s

.PHONY: test-integration-bsc
test-integration-bsc:
	@go test -v --tags=integrationtests -mod=readonly -run 'TestBSC' ./integration-tests -timeout 300s

.PHONY: lint
lint:
	$(BUILDER) lint

.PHONY: fmt
fmt:
	which gofumpt || @go install mvdan.cc/gofumpt@v0.4.0
	which gogroup || @go install github.com/vasi-stripe/gogroup/cmd/gogroup@v0.0.0-20200806161525-b5d7f67a97b5
	@gofumpt -lang=1.9 -extra -w $(SCAN_FILES)
	@gogroup -order std,other,prefix=$(IMPORT_PREFIX) -rewrite $(SCAN_FILES)

.PHONY: build-contract
build-contract:
	docker run --user $(id -u):$(id -g) --rm -v $(CONTRACT_DIR):/code \
      --mount type=volume,source="contract_cache",target=/code/target \
      --mount type=volume,source=registry_cache,target=/usr/local/cargo/registry \
      cosmwasm/rust-optimizer:0.13.0


.PHONY: test-contract
test-contract:
	rustup install 1.81.0 && rustup default 1.81.0 && rustup component add clippy && \
      cd contract   && cargo clippy --all && cargo test --verbose

.PHONY: restart-dev-env
restart-dev-env:
	${TX_BUILDER} znet remove && ${TX_BUILDER} znet start --profiles=1cored,xrpl

stop-dev-env:
	${TX_BUILDER} znet remove 
	
.PHONY: rebuild-dev-env
rebuild-dev-env:
	${TX_BUILDER} build images
