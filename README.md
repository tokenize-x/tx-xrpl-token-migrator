# XRPL bridge

The XRPL bridge is one way XRPL to coreum bridge.

## Build

### Use binary sources (linux only)

Download binary from the [releases](https://github.com/CoreumFoundation/xrpl-bridge/releases) page to your machine. Pay
attention that repo is private so the binary should be downloaded from the trusted machine.

### Build from sources

(The build phase is optional for the deployment, since it's possible to use built binary).

Run to build from sources

```bash
make build-contract
make build
```

Run to build in docker (linux only)

```bash
make build-in-docker
```
## Deploy contract

* Set public variable:

```bash
export COREUM_CHAIN_ID="{Chain ID}"
export COREUM_CONTRACT_TRUSTED_ADDRESSES="{Trusted address 2,trusted address 1}"
export COREUM_CONTRACT_THRESHOLD="{Threshold}"
export COREUM_CONTRACT_OWNER="{Owner which is able to withdraw coins}"
```

* Store deployer mnemonic to the keystore

```
./relayer keys add --recover contract-deployer --coreum-chain-id $COREUM_CHAIN_ID
```

* Deploy smart contract

```
./relayer deploy --coreum-chain-id $COREUM_CHAIN_ID \
    --coreum-contract-trusted-addresses $COREUM_CONTRACT_TRUSTED_ADDRESSES \
    --coreum-contract-threshold $COREUM_CONTRACT_THRESHOLD \
    --coreum-contract-owner-address $COREUM_CONTRACT_OWNER \
    --coreum-sender-address $(./relayer keys show contract-deployer -a --coreum-chain-id $COREUM_CHAIN_ID)
```

## Start relayer

* Store relayer mnemonic

Call the command and add the relayer mnemonic there.

```
./relayer keys add --recover relayer --coreum-chain-id $COREUM_CHAIN_ID
```

```bash
export COREUM_CHAIN_ID="{Chain ID}"
export COREUM_CONTRACT_ADDRESS="{Contract address}"
export COREUM_RELAYER_ADDRESS="$(./relayer keys show relayer -a --coreum-chain-id $COREUM_CHAIN_ID)"
```

* Create start service script

```bash
echo "
$PWD/relayer start --coreum-chain-id $COREUM_CHAIN_ID --coreum-contract-address $COREUM_CONTRACT_ADDRESS --coreum-sender-address $COREUM_RELAYER_ADDRESS
    " > "run.sh"
chmod +x run.sh
```

* Start relayer manually to test that it is configured correctly.

```bash
./run.sh
```

* Add it as a service

```bash
echo "
    [Unit]
    After=network.target
    
    [Service]
    User=root
    ExecStart=/bin/sh $PWD/run.sh
    Restart=always
    RestartSec=3
    
    [Install]
    WantedBy=multi-user.target
    " > "/etc/systemd/system/relayer.service"
    
    systemctl daemon-reload
    systemctl enable relayer
    systemctl start relayer
    systemctl status relayer --no-pager
```

* Check the logs

```bash
journalctl -u relayer -f
```

## Relayer support

### Load keys after reboot

Call the command load key to let the service take it at the next restart.

```bash
pass xrpl-bridge/coreum-contract-relayer-mnemonic > /dev/null
```

### Read errors

```bash
journalctl -u relayer -n 100000 --no-pager | grep "error"
```
