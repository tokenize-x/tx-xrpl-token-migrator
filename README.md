# XRPL bridge

The XRPL bridge is one way XRPL to TX bridge.

## Build

### Use binary (linux only)

Download binary from the [releases](https://github.com/tokenize-x/tx-xrpl-token-migrator/releases) page to your machine and
make it executable. Pay attention that repo is private so the binary should be downloaded from the trusted machine.

### Build from sources

(The build phase is optional for the deployment, since it's possible to use the release binary).

* Run to build from sources

```bash
make build-contract
make build
```

* Run to build in docker (linux only)

```bash
make build-in-docker
```

## Deploy contract

* Set variables

```bash
export TX_CHAIN_ID={TX chain ID}"
export TX_CONTRACT_TRUSTED_ADDRESSES="{Trusted address 2,trusted address 1}"
export TX_CONTRACT_THRESHOLD="{Threshold}"
export TX_CONTRACT_OWNER="{Owner which is able to withdraw contract balance}"
export TX_CONTRACT_MIN_AMOUNT="{Min allowed amount for a transaction}"
export TX_CONTRACT_MAX_AMOUNT="{Max allowed amount for automated transaction processing}"
export TX_GRPC_URL="{GRPC URL of TX node}"
```

* Store deployer mnemonic to the keystore

```
./relayer keys add --recover contract-deployer \
    --tx-chain-id $TX_CHAIN_ID \
    --keyring-backend os \
    --home $HOME/.xrpl-bridge
```

* Deploy smart contract

```
./relayer deploy-and-instantiate --tx-chain-id $TX_CHAIN_ID \
    --tx-contract-trusted-addresses $TX_CONTRACT_TRUSTED_ADDRESSES \
    --tx-contract-threshold $TX_CONTRACT_THRESHOLD \
    --tx-contract-owner-address $TX_CONTRACT_OWNER \
    --tx-contract-min-amount $TX_CONTRACT_MIN_AMOUNT \
    --tx-contract-max-amount $TX_CONTRACT_MAX_AMOUNT \
    --tx-grpc-url $TX_GRPC_URL \
    --tx-sender-address $(./relayer keys show contract-deployer -a --tx-chain-id $TX_CHAIN_ID --keyring-backend os --home $HOME/.xrpl-bridge) \
    --keyring-backend os \
    --home $HOME/.xrpl-bridge
```

## Start relayer

### Start relayer manually

* Import relayer mnemonic to keyring.

```bash
./relayer keys add --recover relayer \
    --tx-chain-id $TX_CHAIN_ID \
    --keyring-backend os \
    --home $HOME/.xrpl-bridge
```

* Set variables

```bash
export XRPL_RPC_URL="{RPC URL of XRPL node}"
export TX_CHAIN_ID={TX chain ID}"
export TX_CONTRACT_ADDRESS="{Contract address}"
export TX_GRPC_URL="{GRPC URL of TX node}"
export PROMETHEUS_INSTANCE_NAME="{Unique name of your instance}"
export PROMETHEUS_USERNAME="{Prometheus username}"
export PROMETHEUS_PASSWORD="{Prometheus password}"
export PROMETHEUS_URL="{Prometheus push URL}"
```

* Create `start` script.

**Mainnet**

```bash
echo "
echo \$(systemd-ask-password \"Enter keyring password:\") | $PWD/relayer start \\
    --xrpl-rpc-url $XRPL_RPC_URL \\
    --tx-chain-id $TX_CHAIN_ID \\
    --tx-contract-address $TX_CONTRACT_ADDRESS \\
    --tx-grpc-url $TX_GRPC_URL \\
    --tx-sender-address $(./relayer keys show relayer -a --tx-chain-id $TX_CHAIN_ID --keyring-backend os --home $HOME/.xrpl-bridge) \\
    --prometheus-instance-name $PROMETHEUS_INSTANCE_NAME \\
    --prometheus-username $PROMETHEUS_USERNAME \\
    --prometheus-password $PROMETHEUS_PASSWORD \\
    --prometheus-url $PROMETHEUS_URL \\
    --keyring-backend os \\
    --home $HOME/.xrpl-bridge
    " > "run-xrpl-bridge-relayer.sh"
chmod +x run-xrpl-bridge-relayer.sh
```

**Testnet**

```bash
echo "
echo \$(systemd-ask-password \"Enter keyring password:\") | $PWD/relayer start \\
    --xrpl-rpc-url $XRPL_RPC_URL \\
    --tx-chain-id $TX_CHAIN_ID \\
    --tx-contract-address $TX_CONTRACT_ADDRESS \\
    --tx-grpc-url $TX_GRPC_URL \\
    --tx-sender-address $(./relayer keys show relayer -a --tx-chain-id $TX_CHAIN_ID --keyring-backend os --home $HOME/.xrpl-bridge) \\
    --prometheus-instance-name $PROMETHEUS_INSTANCE_NAME \\
    --prometheus-username $PROMETHEUS_USERNAME \\
    --prometheus-password $PROMETHEUS_PASSWORD \\
    --prometheus-url $PROMETHEUS_URL \\
    --keyring-backend os \\
    --home $HOME/.xrpl-bridge
    " > "run-xrpl-bridge-relayer.sh"
chmod +x run-xrpl-bridge-relayer.sh
```

* Start script manually to test that it is configured correctly

```bash
./run-xrpl-bridge-relayer.sh
```

You will be asked to `Enter keyring password`, enter it and press `Enter`.
If you don't see errors after the start, it's configured correctly. Stop it.

#### Add systemctrl service (prev step is required)

* Update `/etc/systemd/journald.conf` to store logs and reboot system.

```toml
[Journal]
Storage=persistent
```

* Add service

```bash
echo "
    [Unit]
    After=network.target

    [Service]
    Environment=\"HOME=$HOME\"
    ExecStart=/bin/sh $PWD/run-xrpl-bridge-relayer.sh

    [Install]
    WantedBy=multi-user.target
    " > "/etc/systemd/system/xrpl-bridge-relayer.service"

systemctl daemon-reload
systemctl enable xrpl-bridge-relayer
systemctl start xrpl-bridge-relayer
```

* Run command to enter the keyring password:

```bash
systemd-tty-ask-password-agent
```

You will be asked to `Enter keyring password`, enter it and press `Enter`.

* Check status and logs

```bash
systemctl status xrpl-bridge-relayer --no-pager
journalctl -u xrpl-bridge-relayer -n 100 --no-pager
```

!!! Pass the [Run promtail](#run-promtail) step to send logs to the loki. !!!

## Relayer support

### Re-init relayer after the reboot

* Run command

```bash
systemctl restart xrpl-bridge-relayer
```

* Run command to enter the keyring password:

```bash
systemd-tty-ask-password-agent
```

You will be asked to `Enter keyring password`, enter it and press `Enter`.

### Read errors

```bash
journalctl -u xrpl-bridge-relayer --since "24 hour ago" --no-pager | grep "error" # replace error with any text you search for
```

### Disable/remove service

```bash
systemctl stop xrpl-bridge-relayer
systemctl disable xrpl-bridge-relayer
systemctl daemon-reload
```

## Execute pending approved transactions

Some transactions might reach the max allowed contract amount and will be kept in pending until manual execution of
them.

### Get list of pending approved transactions.

* Set variables

```bash
export TX_CHAIN_ID={TX chain ID}"
export TX_CONTRACT_ADDRESS="{Contract address}"
export TX_GRPC_URL="{GRPC URL of TX node}"
```

```bash
./relayer get-pending-approved-transactions \
  --tx-contract-address $TX_CONTRACT_ADDRESS \
  --tx-grpc-url $TX_GRPC_URL \
  --tx-chain-id $TX_CHAIN_ID
```

Output example:

```
2023-07-06T16:02:27.411231+03:00        info    relayer cmd/main.go:405 Approved pending transactions:  {"total": 2, "evidenceIDs": ["096c29de43b2499849d3bae66144dfc3cb43a8b00eb0751a90f8f5f4cb7a2255-1296000000utestcore-testcore1cz8x502s930v0ux8m6lpfw6s3l5tydz3gsx87w", "12e002d9cf20ef1941bdeddf426beb5b2455fb512f4a5b1275c81916e04ffc19-100000000utestcore-testcore1lqfzshr8r4v8nr4wa8mpg8mf5p27t78yl8hnta"]}
```

Using the output you can choose which transactions you would like to execute.

### Prepare transaction to be executed.

* Set variables

```bash
export TX_CHAIN_ID={TX chain ID}"
export TX_CONTRACT_ADDRESS="{Contract address}"
export TX_GRPC_URL="{GRPC URL of TX node}"
export TX_EXECUTOR_ADDRESS="{The address which will execute the approved transactions and pay for them}"
export TX_EVIDENCE_IDS="{Comma separated evidence IDs of the pending approved transactions to be executed}" # (Optional, by default all transactions will be executed.)
```

* Print to file `execute` transaction.

```bash
./relayer build-execute-pending-approved-transaction \
  --tx-contract-address $TX_CONTRACT_ADDRESS \
  --tx-grpc-url $TX_GRPC_URL \
  --tx-contract-evidence-ids $TX_EVIDENCE_IDS \
  --tx-chain-id $TX_CHAIN_ID \
  --tx-sender-address $TX_EXECUTOR_ADDRESS > unsigned.json
```

The `--tx-contract-evidence-ids $TX_EVIDENCE_IDS` part is optional, by default all transactions will be
executed.

* Check the transaction file content before the execution.

```bash
cat unsigned.json
```

* [Sign and broadcast with cored](#Sign-and-broadcast-with-cored)

### Run audit.

* Set variables

```bash
export TX_CHAIN_ID={TX chain ID}"
export TX_CONTRACT_ADDRESS="{Contract address}"
export TX_RPC_URL="{RPC URL of TX node}"
export XRPL_RPC_URL="{RPC URL of XRPL node}"
```

```bash
./relayer audit \
--tx-contract-address $TX_CONTRACT_ADDRESS \
--tx-rpc-url $TX_RPC_URL \
--tx-chain-id $TX_CHAIN_ID \
--xrpl-rpc-url $XRPL_RPC_URL
```

## Set up env

### Run promtail

#### Install binary (you need `wget` and  `unzip` to be installed)

```bash
sudo su
wget https://github.com/grafana/loki/releases/download/v2.7.3/promtail-linux-amd64.zip
unzip promtail-linux-amd64.zip
mv promtail-linux-amd64 /usr/local/bin/promtail && sudo chmod 755 /usr/local/bin/promtail
rm promtail-linux-amd64.zip
mkdir -p /etc/promtail

promtail version
```

#### Create promtail config

* Set variables

```
export LOKI_INSTANCE_NAME="{Unique name of your instance}"
export LOKI_USERNAME="{Loki username}"
export LOKI_PASSWORD="{Loki password}"
export LOKI_URL="{Loki push URL}"
```

* Create config

```bash
echo "
---
server:
  disable: true

positions:
  filename: /tmp/positions.yaml

clients:
  - url: \"$LOKI_URL\"
    basic_auth:
      username: \"$LOKI_USERNAME\"
      password: \"$LOKI_PASSWORD\"

scrape_configs:
- job_name: xrpl-bridge-relayer
  journal:
    json: false
    max_age: 12h
    path: /var/log/journal
    matches: _SYSTEMD_UNIT=xrpl-bridge-relayer.service
    labels:
      job: xrpl-bridge-relayer
      instance: \"$LOKI_INSTANCE_NAME\"
" >  /etc/promtail/config.yaml
```

* Start promtail to validate the config

```bash
promtail -config.file=/etc/promtail/config.yaml -config.expand-env=true
```

* Add service

```bash
echo "
[Unit]
Description = promtail logshipper

[Service]
ExecStart = /bin/bash -c \"/usr/local/bin/promtail -config.file=/etc/promtail/config.yaml -config.expand-env=true\"

[Install]
WantedBy=multi-user.target" > /etc/systemd/system/promtail.service

systemctl daemon-reload
systemctl enable promtail
systemctl start promtail
```

* Check status and logs

```bash
systemctl status promtail --no-pager
journalctl -u promtail -n 100 --no-pager
```

#### Additional Notes

If you wish to run two instances of bridge on the same VM, makes sure:

1. Rename xrpl bridge instance to include something instance specific
   `ex. xrpl-bridge-relayer.service => xrpl-bridge-*instance_name_one*-relayer.service`
   `ex. xrpl-bridge-relayer.service => xrpl-bridge-*instance_name_two*-relayer.service`

2. Create 2 separate configs for promtail
   `ex. /etc/promtail/config.yaml => /etc/promtail/*instance_name_one*_config.yaml`
   `ex. /etc/promtail/config.yaml => /etc/promtail/*instance_name_two*_config.yaml`

3. Create 2 separate services for promtail
   `ex. /etc/systemd/system/promtail.service => /etc/systemd/system/promtail_*instance_name_one*.service`
   `ex. /etc/systemd/system/promtail.service => /etc/systemd/system/promtail_*instance_name_two*.service`

4. Edit `promtail` instance specific service to read from instance specific promtail `config.yaml`

## Update trusted addresses keys

* Deploy new smart contract

```bash
./relayer deploy --tx-chain-id $TX_CHAIN_ID \
    --tx-grpc-url $TX_GRPC_URL \
    --tx-sender-address $(./relayer keys show contract-deployer -a --tx-chain-id $TX_CHAIN_ID --keyring-backend os --home $HOME/.xrpl-bridge) \
    --keyring-backend os \
    --home $HOME/.xrpl-bridge
```

Save generate codeID.

* Generate tx to migrate the contract

```bash
export TX_CONTRACT_OWNER={TX contract owner}
export TX_NEW_CONTRACT_CODE_ID={New contract code ID}
export TX_CONTRACT_ADDRESS={Contract address}

./relayer build-migrate-contract-transaction $TX_NEW_CONTRACT_CODE_ID \
    --tx-chain-id $TX_CHAIN_ID \
    --tx-grpc-url $TX_GRPC_URL \
    --tx-sender-address $TX_CONTRACT_OWNER \
    --tx-contract-address $TX_CONTRACT_ADDRESS > unsigned.json
```

* [Sign and broadcast with cored](#Sign-and-broadcast-with-cored)

* Generate tx to update trusted addresses

```bash
export TX_CONTRACT_OWNER={TX contract owner}
export TX_NEW_TRUSTED_ADDRESSES={New trusted addresses}
export TX_CONTRACT_ADDRESS={Contract address}

./relayer build-update-trusted-addresses \
    --tx-chain-id $TX_CHAIN_ID \
    --tx-grpc-url $TX_GRPC_URL \
    --tx-sender-address $TX_CONTRACT_OWNER \
    --tx-contract-trusted-addresses $TX_NEW_TRUSTED_ADDRESSES \
    --tx-contract-address $TX_CONTRACT_ADDRESS > unsigned.json
```

* [Sign and broadcast with cored](#Sign-and-broadcast-with-cored)

* Check now that addresses are updated

```
./relayer get-contract-config \
    --tx-chain-id $TX_CHAIN_ID \
    --tx-grpc-url $TX_GRPC_URL \
    --tx-contract-address $TX_CONTRACT_ADDRESS
```

## Sign and broadcast with cored

* Export node URL

```
export TX_CHAIN_ID={TX chain ID}"
export TX_EXECUTOR_ADDRESS="{The address which will execute the approved transactions and pay for them}"
export TX_NODE="{Node RPC URL}"
```

* Sign with `cored` (the same can be done with the multisig account).

```bash
cored tx sign unsigned.json --from $TX_EXECUTOR_ADDRESS --output-document signed.json --chain-id $TX_CHAIN_ID --node $TX_NODE
```

* Broadcast with `cored`.

```bash
cored tx broadcast signed.json -y -b block --chain-id $TX_CHAIN_ID --node $TX_NODE
```

## Migration and Upgrade Guide v2.3.0

This guide covers the process of migrating and upgrading both the relayer binary and the smart contract.

### Overview

The migration process involves:
1. **Contract Migration**: Deploy new contract code and migrate the existing contract instance
2. **Add XRPL Tokens**: Add XRPL tokens to the contract (recommended before starting relayer)
3. **Relayer Upgrade**: Update relayer configuration and upgrade the relayer binary

### Important Configuration Changes

#### Relayer Configuration Field Name Changes

The following configuration fields have been renamed:
- `--coreum-chain-id` → `--tx-chain-id`
- `--coreum-rpc-url` → `--tx-rpc-url`
- `--coreum-grpc-url` → `--tx-grpc-url`
- `--coreum-sender-address` → `--tx-sender-address`
- `--coreum-contract-address` → `--tx-contract-address`

#### XRPL Tokens Configuration

XRPL tokens are now stored in the contract. The relayer reads them automatically from the contract.

### Prerequisites

Before starting the migration:

* Ensure you have access to:
  - Contract deployer key (for deploying new contract code)
  - Contract owner key (for executing migration transaction)
  - Relayer key (for relayer binary upgrade)
* Backup all keys and save them in a safe place
* Verify current contract address and configuration
* Ensure sufficient balance in owner account for transaction fees

### Step 1: Backup Current State

* Export relayer key (recommended)

```bash
./relayer keys export relayer \
    --tx-chain-id $TX_CHAIN_ID \
    --keyring-backend os \
    --home $HOME/.xrpl-bridge
```

Save the exported key in a secure location.

* Get current contract configuration

```bash
export TX_CHAIN_ID="{TX chain ID}"
export TX_CONTRACT_ADDRESS="{Contract address}"
export TX_GRPC_URL="{GRPC URL of TX node}"

./relayer get-contract-config \
    --tx-chain-id $TX_CHAIN_ID \
    --tx-grpc-url $TX_GRPC_URL \
    --tx-contract-address $TX_CONTRACT_ADDRESS
```

Save the output for reference.

### Step 2: Deploy New Contract Code

* Set variables and ensure deployer key is in keyring

```bash
# If not already added, recover the deployer key
./relayer keys add --recover contract-deployer \
    --tx-chain-id $TX_CHAIN_ID \
    --keyring-backend os \
    --home $HOME/.xrpl-bridge
```

* Deploy new contract code

```bash
./relayer deploy \
    --tx-chain-id $TX_CHAIN_ID \
    --tx-grpc-url $TX_GRPC_URL \
    --tx-sender-address $TX_CONTRACT_DEPLOYER_ADDRESS \
    --keyring-backend os \
    --home $HOME/.xrpl-bridge
```

**Important**: Save the generated `codeID` from the output. You will need it for the migration step.

Example output:
```
2024-01-15T10:30:45.123456+00:00        info    relayer cmd/main.go:346 Contract deployed {"codeID": 123}
```

### Step 3: Migrate Contract

* Build the migration transaction (set `TX_NEW_CONTRACT_CODE_ID` from Step 2 output)

```bash
./relayer build-migrate-contract-transaction $TX_NEW_CONTRACT_CODE_ID \
    --tx-chain-id $TX_CHAIN_ID \
    --tx-grpc-url $TX_GRPC_URL \
    --tx-sender-address $TX_CONTRACT_OWNER \
    --tx-contract-address $TX_CONTRACT_ADDRESS > unsigned-migrate.json
```

* Review `unsigned-migrate.json`, then sign and broadcast

```bash
export TX_NODE="{Node RPC URL}"

cored tx sign unsigned-migrate.json \
    --from $TX_CONTRACT_OWNER \
    --output-document signed-migrate.json \
    --chain-id $TX_CHAIN_ID \
    --node $TX_NODE

cored tx broadcast signed-migrate.json \
    -y -b block \
    --chain-id $TX_CHAIN_ID \
    --node $TX_NODE
```

* Verify: `./relayer get-contract-config` should show new code ID and `version: 1` or higher

### Step 5: Add XRPL Tokens to Contract

**Important**: Add XRPL tokens before starting the relayer.

**For Testnet:**
- **CORE**: Currency `434F524500000000000000000000000000000000`, Issuer `raSEP47QAwU6jsZU493znUD2iGNHDQEyvA`, Activation `2000-01-01 00:00:00 UTC` (946684800), Multiplier `1.0`
- **XCORE**: Currency `58434F5245000000000000000000000000000000`, Issuer `rawnyFwFLkntQttzBgEFiASg5iB5ULdKpX`, Activation `2000-01-01 00:00:00 UTC` (946684800), Multiplier `1.0`
- **SOLO**: Currency `534F4C4F00000000000000000000000000000000`, Issuer `rHZwvHEs56GCmHupwjA4RY7oPA3EoAJWuN`, Activation `2000-01-01 00:00:00 UTC` (946684800), Multiplier `1.25`

```bash
./relayer build-add-xrpl-tokens \
    --tx-chain-id $TX_CHAIN_ID \
    --tx-grpc-url $TX_GRPC_URL \
    --tx-sender-address $TX_CONTRACT_OWNER \
    --tx-contract-address $TX_CONTRACT_ADDRESS \
    --xrpl-token "raSEP47QAwU6jsZU493znUD2iGNHDQEyvA/434F524500000000000000000000000000000000/946684800/1.0" \
    --xrpl-token "rawnyFwFLkntQttzBgEFiASg5iB5ULdKpX/58434F5245000000000000000000000000000000/946684800/1.0" \
    --xrpl-token "rHZwvHEs56GCmHupwjA4RY7oPA3EoAJWuN/534F4C4F00000000000000000000000000000000/946684800/1.25" > unsigned-add-tokens.json
```

**For Mainnet:**
- **CORE**: Currency `434F524500000000000000000000000000000000`, Issuer `rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D`, Activation `2000-01-01 00:00:00 UTC` (946684800), Multiplier `1.0`
- **XCORE**: Currency `58434F5245000000000000000000000000000000`, Issuer `r3dVizzUAS3U29WKaaSALqkieytA2LCoRe`, Activation `2025-03-24 04:00:00 UTC` (1742788800), Multiplier `1.0`
- **SOLO**: Currency `534F4C4F00000000000000000000000000000000`, Issuer `rsoLo2S1kiGeCcn6hCUXVrCpGMWLrRrLZz`, Activation `2025-06-25 04:00:00 UTC` (1750824000), Multiplier `1.25`

```bash
./relayer build-add-xrpl-tokens \
    --tx-chain-id $TX_CHAIN_ID \
    --tx-grpc-url $TX_GRPC_URL \
    --tx-sender-address $TX_CONTRACT_OWNER \
    --tx-contract-address $TX_CONTRACT_ADDRESS \
    --xrpl-token "rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D/434F524500000000000000000000000000000000/946684800/1.0" \
    --xrpl-token "r3dVizzUAS3U29WKaaSALqkieytA2LCoRe/58434F5245000000000000000000000000000000/1742788800/1.0" \
    --xrpl-token "rsoLo2S1kiGeCcn6hCUXVrCpGMWLrRrLZz/534F4C4F00000000000000000000000000000000/1750824000/1.25" > unsigned-add-tokens.json
```

* Sign and broadcast (see [Sign and broadcast with cored](#Sign-and-broadcast-with-cored))

* Verify: `./relayer get-contract-config` should show tokens in `xrpl_tokens` field

### Step 6: Update Relayer Configuration

After contract migration, you need to update the relayer startup script with the new configuration field names.

* Update the relayer startup script with new field names

The relayer configuration field names have changed. Update your `run-xrpl-bridge-relayer.sh` script:

**Old field names (before migration):**
- `--coreum-chain-id` → `--tx-chain-id`
- `--coreum-rpc-url` → `--tx-rpc-url` (if used)
- `--coreum-grpc-url` → `--tx-grpc-url`
- `--coreum-sender-address` → `--tx-sender-address`
- `--coreum-contract-address` → `--tx-contract-address`

**Example updated script:**

```bash
echo "
echo \$(systemd-ask-password \"Enter keyring password:\") | $PWD/relayer start \\
    --xrpl-rpc-url $XRPL_RPC_URL \\
    --tx-chain-id $TX_CHAIN_ID \\
    --tx-contract-address $TX_CONTRACT_ADDRESS \\
    --tx-grpc-url $TX_GRPC_URL \\
    --tx-sender-address $(./relayer keys show relayer -a --tx-chain-id $TX_CHAIN_ID --keyring-backend os --home $HOME/.xrpl-bridge) \\
    --prometheus-instance-name $PROMETHEUS_INSTANCE_NAME \\
    --prometheus-username $PROMETHEUS_USERNAME \\
    --prometheus-password $PROMETHEUS_PASSWORD \\
    --prometheus-url $PROMETHEUS_URL \\
    --keyring-backend os \\
    --home $HOME/.xrpl-bridge
    " > "run-xrpl-bridge-relayer.sh"
chmod +x run-xrpl-bridge-relayer.sh
```


### Step 7: Upgrade Relayer Binary

* Stop the relayer service (if it's still running)

```bash
systemctl stop xrpl-bridge-relayer
```

* Download the latest relayer binary from the [releases](https://github.com/tokenize-x/tx-xrpl-token-migrator/releases) page

```bash
# Replace VERSION with the actual version number (e.g., v2.3.0)
wget https://github.com/tokenize-x/tx-xrpl-token-migrator/releases/download/VERSION/relayer
chmod +x relayer
```

* Verify the binary version

```bash
./relayer version
```

### Step 8: Start Relayer and Verify

* Start the relayer service

```bash
systemctl start xrpl-bridge-relayer
```

* Enter the keyring password

```bash
systemd-tty-ask-password-agent
```

* Monitor the relayer logs

```bash
journalctl -u xrpl-bridge-relayer -f
```

Look for successful connection, contract config loaded, and XRPL tokens loaded from contract.

* Verify relayer is functioning correctly

```bash
# Check for pending transactions (should work normally)
./relayer get-pending-approved-transactions \
    --tx-contract-address $TX_CONTRACT_ADDRESS \
    --tx-grpc-url $TX_GRPC_URL \
    --tx-chain-id $TX_CHAIN_ID
```

* Run audit to verify data integrity

```bash
export XRPL_RPC_URL="{RPC URL of XRPL node}"
export TX_RPC_URL="{RPC URL of TX node}"

./relayer audit \
    --tx-contract-address $TX_CONTRACT_ADDRESS \
    --tx-rpc-url $TX_RPC_URL \
    --tx-chain-id $TX_CHAIN_ID \
    --xrpl-rpc-url $XRPL_RPC_URL
```

The audit should complete without discrepancies.

### Step 9: Post-Migration Tasks

* Monitor the relayer for at least 24 hours to ensure stable operation

```bash
journalctl -u xrpl-bridge-relayer -f
```

* Verify that new transactions are being processed correctly


* Clean up temporary files

```bash
rm -f unsigned-migrate.json signed-migrate.json unsigned-add-tokens.json signed-add-tokens.json
```

### Troubleshooting

#### Relayer fails to start after upgrade

* Check the relayer version matches the expected version
* Verify all environment variables are set correctly with new field names (`TX_CHAIN_ID` instead of `CoreumChainID`, etc.)
* Check logs for specific error messages: `journalctl -u xrpl-bridge-relayer -n 100 --no-pager`
* Verify the startup script uses the new field names (see Step 6)

#### Configuration field name errors

If you see errors about unknown flags or missing configuration:

* Ensure you're using the new field names:
  - `--tx-chain-id` (not `--coreum-chain-id`)
  - `--tx-grpc-url` (not `--coreum-grpc-url`)
  - `--tx-sender-address` (not `--coreum-sender-address`)
  - `--tx-contract-address` (not `--coreum-contract-address`)
* Update your startup script and environment variables

#### Contract migration fails

* Verify the contract owner address has sufficient balance for fees
* Ensure the new code ID is correct
* Check that the contract address is correct
* Verify the transaction was signed and broadcast correctly
* Check transaction status: `cored query tx <tx_hash> --chain-id $TX_CHAIN_ID --node $TX_NODE`

#### Contract state issues after migration

* If issues persist, verify the contract configuration: `./relayer get-contract-config`
* Check that the contract version was updated: the config should show `version: 1` or higher
* Run the audit command to check for data discrepancies

#### XRPL tokens not found after migration

* Add tokens using `build-add-xrpl-tokens` (see Step 5)
* Verify: `./relayer get-contract-config` should show tokens in `xrpl_tokens` field

### Notes

* Contract migration preserves all existing state
* XRPL tokens must be added after migration (see Step 5)
