# syntax=docker/dockerfile:1.4
FROM cosmwasm/rust-optimizer:0.13.0 AS contract-builder

COPY contract /code
RUN /usr/local/bin/optimize.sh /code

FROM golang:1.23.3-alpine3.20

# install git and openssh-client so git+ssh can use the forwarded agent
RUN apk add --no-cache gcc libc-dev linux-headers make git openssh-client

WORKDIR /code

COPY . .
COPY --from=contract-builder /code/artifacts/threshold_bank_send.wasm contract/artifacts/threshold_bank_send.wasm

ARG BUILD_VERSION=""
# 1. Set the private module variable
ENV GOPRIVATE=github.com/tokenize-x/*

# 2. Configure Git to use SSH for GitHub URLs
RUN git config --global url."git@github.com:".insteadOf "https://github.com/"

# 3. Ensure your SSH keys are available (using BuildKit) and download modules via SSH
RUN --mount=type=ssh go mod download

# Build using SSH mount so private modules can be fetched during the build
RUN --mount=type=ssh BUILD_VERSION=${BUILD_VERSION} make build
