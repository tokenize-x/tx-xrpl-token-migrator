FROM cosmwasm/rust-optimizer:0.13.0 AS contract-builder

COPY contract /code
RUN /usr/local/bin/optimize.sh /code

FROM golang:1.21.4-alpine3.17

RUN apk add --no-cache gcc libc-dev linux-headers make

WORKDIR /code

COPY . .
COPY --from=contract-builder /code/artifacts/threshold_bank_send.wasm contract/artifacts/threshold_bank_send.wasm

ARG BUILD_VERSION=""
RUN BUILD_VERSION=${BUILD_VERSION} make build
