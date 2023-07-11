# XRPL bridge

The XRPL bridge is one way XRPL to coreum bridge.

## Build

### Use binary (linux only)

Download binary from the [releases](https://github.com/CoreumFoundation/xrpl-bridge/releases) page to your machine and
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

* Set public chain specific variables

```bash
export COREUM_CHAIN_ID="{Chain ID}"
export COREUM_CONTRACT_TRUSTED_ADDRESSES="{Trusted address 2,trusted address 1}"
export COREUM_CONTRACT_THRESHOLD="{Threshold}"
export COREUM_CONTRACT_OWNER="{Owner which is able to withdraw contract balance}"
export COREUM_CONTRACT_MIN_AMOUNT="{Min allowed amount for a transaction}"
export COREUM_CONTRACT_MAX_AMOUNT="{Max allowed amount for automated transaction processing}"
```

* Store deployer mnemonic to the keystore

```
./relayer keys add --recover contract-deployer \
    --coreum-chain-id $COREUM_CHAIN_ID \
    --keyring-backend os \
    --home $HOME/.xrpl-bridge
```

* Deploy smart contract

```
./relayer deploy --coreum-chain-id $COREUM_CHAIN_ID \
    --coreum-contract-trusted-addresses $COREUM_CONTRACT_TRUSTED_ADDRESSES \
    --coreum-contract-threshold $COREUM_CONTRACT_THRESHOLD \
    --coreum-contract-owner-address $COREUM_CONTRACT_OWNER \
    --coreum-contract-min-amount $COREUM_CONTRACT_MIN_AMOUNT \
    --coreum-contract-max-amount $COREUM_CONTRACT_MAX_AMOUNT \
    --coreum-sender-address $(./relayer keys show contract-deployer -a --coreum-chain-id $COREUM_CHAIN_ID --keyring-backend os --home $HOME/.xrpl-bridge) \
    --keyring-backend os \
    --home $HOME/.xrpl-bridge
```

## Start relayer

### Start relayer manually

* Import relayer mnemonic to keyring.

```
./relayer keys add --recover relayer \
    --coreum-chain-id $COREUM_CHAIN_ID \
    --keyring-backend os \
    --home $HOME/.xrpl-bridge
```

* Set `run` variables

```bash
export COREUM_CONTRACT_ADDRESS="{Contract address}"
export PROMETHEUS_INSTANCE_NAME="{Unique name of your instance}"
export PROMETHEUS_USERNAME="{Prometheus username}"
export PROMETHEUS_PASSWORD="{Prometheus password}"
```

* Create `start` script.

```bash
echo "
echo \$(systemd-ask-password \"Enter keyring password:\") | $PWD/relayer start \\
    --coreum-chain-id $COREUM_CHAIN_ID \\
    --coreum-contract-address $COREUM_CONTRACT_ADDRESS \\
    --coreum-sender-address $(./relayer keys show relayer -a --coreum-chain-id $COREUM_CHAIN_ID --keyring-backend os --home $HOME/.xrpl-bridge) \\
    --prometheus-instance-name $PROMETHEUS_INSTANCE_NAME \\
    --prometheus-username $PROMETHEUS_USERNAME \\
    --prometheus-password $PROMETHEUS_PASSWORD \\
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
If you don't see errors after the start, it's conferred correctly. Stop it.

#### Add systemctrl service (prev step is required)

* Update `/etc/systemd/journald.conf` to store logs and reboot system.

```
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

Some transactions might reach the max allowed contract amount and will be kept in pending until manual execution of them.

### Get list of pending approved transactions.

```bash
./relayer get-pending-approved-transactions --coreum-chain-id $COREUM_CHAIN_ID
```

Output example:

```
2023-07-06T16:02:27.411231+03:00        info    relayer cmd/main.go:405 Approved pending transactions:  {"total": 2, "evidenceIDs": ["096c29de43b2499849d3bae66144dfc3cb43a8b00eb0751a90f8f5f4cb7a2255-1296000000utestcore-testcore1cz8x502s930v0ux8m6lpfw6s3l5tydz3gsx87w", "12e002d9cf20ef1941bdeddf426beb5b2455fb512f4a5b1275c81916e04ffc19-100000000utestcore-testcore1lqfzshr8r4v8nr4wa8mpg8mf5p27t78yl8hnta"]}
```

Using the output you can choose which transactions you would like to execute.

### Prepare transaction to be executed.

* Export variables for the execution of the pending approved transactions.

```bash
export COREUM_EXECUTOR_ADDRESS="{The address which will execute the approved transactions and pay for them}"
export COREUM_EVIDENCE_IDS="{Comma separated evidence IDs of the pending approved transactions to be executed}" # (Optional, by default all transactions will be executed.)
```

* Print to file `execute` transaction.

```bash
./relayer build-execute-pending-approved-transaction \
  --coreum-chain-id $COREUM_CHAIN_ID \
  --coreum-contract-evidence-ids $COREUM_EVIDENCE_IDS \
  --coreum-sender-address $COREUM_EXECUTOR_ADDRESS > unsigned.json
```

The `--coreum-contract-evidence-ids $COREUM_EVIDENCE_IDS` part is optional, by default all transactions will be
executed.

* Check the transaction file content before the execution.

```bash
cat unsigned.json
```

* Export node URL

```
export COREUM_NODE="{Node RPC URL}"
```

* Sign with `cored` (the same can be done with the multisig account).

```bash
cored tx sign unsigned.json --from $COREUM_EXECUTOR_ADDRESS --output-document signed.json --chain-id $COREUM_CHAIN_ID --node $COREUM_NODE
```

* Broadcast with `cored`.

```bash
cored tx broadcast signed.json -y -b block --chain-id $COREUM_CHAIN_ID --node $COREUM_NODE
```
