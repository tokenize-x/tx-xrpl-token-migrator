FROM cosmwasm/rust-optimizer:0.13.0 AS contract-builder

COPY contract /code
RUN /usr/local/bin/optimize.sh /code

FROM golang:1.20.1-alpine3.17

RUN apk add --no-cache gcc libc-dev linux-headers make

ARG arch=x86_64
# we use the same arch in the CI as a workaround since we don't use the wasm in the indexer
ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.1.1/libwasmvm_muslc.${arch}.a /lib/libwasmvm_muslc.a

WORKDIR /code

COPY . .
COPY --from=contract-builder /code/artifacts/threshold_bank_send.wasm contract/artifacts/threshold_bank_send.wasm

ARG BUILD_VERSION=""
RUN LINK_STATICALLY=true BUILD_VERSION=${BUILD_VERSION} make build
