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
# Set the private module variable
ENV GOPRIVATE=github.com/tokenize-x/*

# Set up SSH config with host aliases for each private repo's deploy key
# This ensures the correct key is used for each repository
RUN mkdir -p /root/.ssh && \
    ssh-keyscan -H github.com >> /root/.ssh/known_hosts && \
    echo -e "Host tx-chain\n    HostName github.com\n    User git\n    IdentityFile /root/.ssh/tx-chain-deploy-key\n    IdentitiesOnly yes\n" >> /root/.ssh/config && \
    echo -e "Host tx-tools\n    HostName github.com\n    User git\n    IdentityFile /root/.ssh/tx-tools-deploy-key\n    IdentitiesOnly yes\n" >> /root/.ssh/config && \
    echo -e "Host tx-crust\n    HostName github.com\n    User git\n    IdentityFile /root/.ssh/tx-crust-deploy-key\n    IdentitiesOnly yes\n" >> /root/.ssh/config && \
    chmod 600 /root/.ssh/config

# Configure Git to use SSH host aliases for each private repo
RUN git config --global url."git@tx-chain:tokenize-x/tx-chain".insteadOf "https://github.com/tokenize-x/tx-chain" && \
    git config --global url."git@tx-tools:tokenize-x/tx-tools".insteadOf "https://github.com/tokenize-x/tx-tools" && \
    git config --global url."git@tx-crust:tokenize-x/tx-crust".insteadOf "https://github.com/tokenize-x/tx-crust"

# Download modules using SSH secrets (each key mounted to its specific path)
RUN --mount=type=secret,id=tx-chain-key,dst=/root/.ssh/tx-chain-deploy-key,mode=0600 \
    --mount=type=secret,id=tx-tools-key,dst=/root/.ssh/tx-tools-deploy-key,mode=0600 \
    --mount=type=secret,id=tx-crust-key,dst=/root/.ssh/tx-crust-deploy-key,mode=0600 \
    go mod download

# Build using SSH secrets
RUN --mount=type=secret,id=tx-chain-key,dst=/root/.ssh/tx-chain-deploy-key,mode=0600 \
    --mount=type=secret,id=tx-tools-key,dst=/root/.ssh/tx-tools-deploy-key,mode=0600 \
    --mount=type=secret,id=tx-crust-key,dst=/root/.ssh/tx-crust-deploy-key,mode=0600 \
    BUILD_VERSION=${BUILD_VERSION} make build
