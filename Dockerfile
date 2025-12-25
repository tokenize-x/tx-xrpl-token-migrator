FROM cosmwasm/rust-optimizer:0.13.0 AS contract-builder

COPY contract /code
RUN /usr/local/bin/optimize.sh /code

FROM golang:1.23.3-alpine3.20

RUN apk add --no-cache gcc libc-dev linux-headers make git

WORKDIR /code

COPY . .
COPY --from=contract-builder /code/artifacts/threshold_bank_send.wasm contract/artifacts/threshold_bank_send.wasm

ARG BUILD_VERSION=""
# Use vendored modules so docker build does not require network/git access to private repos
ENV GOFLAGS=-mod=vendor
RUN BUILD_VERSION=${BUILD_VERSION} make build
