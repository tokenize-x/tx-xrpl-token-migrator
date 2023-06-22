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

## Init pass as a mnemonic storage

* Install the `pass` on you OS

* Set up the `gpg` key (at least `Real name` must be filled).

```
gpg --gen-key
```

Init pass

```
pass init xrpl-bridge
```

## Deploy contract

* Set public variable:

```bash
export COREUM_CHAIN_ID="{Chain ID}"
export COREUM_CONTRACT_TRUSTED_ADDRESSES="{Trusted address 2,trusted address 1}"
export COREUM_CONTRACT_THRESHOLD="{Threshold}"
export COREUM_CONTRACT_OWNER="{Owner which is able to withdraw coins}"
```

* Store deployer mnemonic to the `pass`

Call the command and add the deployer mnemonic there.

```
pass insert xrpl-bridge/coreum-contract-deployer-mnemonic
```

* Deploy smart contract

```
./relayer deploy --coreum-chain-id $COREUM_CHAIN_ID \
    --coreum-contract-trusted-addresses $COREUM_CONTRACT_TRUSTED_ADDRESSES \
    --coreum-contract-threshold $COREUM_CONTRACT_THRESHOLD \
    --coreum-contract-owner-address $COREUM_CONTRACT_OWNER \
    --coreum-mnemonic "$(pass show xrpl-bridge/coreum-contract-deployer-mnemonic)"
```

## Start relayer

* Store relayer mnemonic

Call the command and add the relayer mnemonic there.

```
pass insert xrpl-bridge/coreum-contract-relayer-mnemonic
```

```bash
export COREUM_CHAIN_ID="{Chain ID}"
export COREUM_CONTRACT_ADDRESSES="{Contract address}"
```

* Create start service script

```bash
echo "
$PWD/relayer start --coreum-chain-id $COREUM_CHAIN_ID --coreum-contract-address $COREUM_CONTRACT_ADDRESSES --coreum-mnemonic \"\$(pass show xrpl-bridge/coreum-contract-relayer-mnemonic)\"
    " > "run.sh"
chmod +x run.sh
```

* Check that you don't use password in the script

```bash
cat run.sh
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

Call the command to add the key to session cache to let the service take it at the next restart.

```bash
pass xrpl-bridge/coreum-contract-relayer-mnemonic > /dev/null
```

### Read errors

```bash
journalctl -u relayer -n 100000 --no-pager | grep "error"
```
