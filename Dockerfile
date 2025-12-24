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
# Use SSH mount for the build step so private modules can be fetched via the forwarded ssh-agent
RUN --mount=type=ssh BUILD_VERSION=${BUILD_VERSION} make build
