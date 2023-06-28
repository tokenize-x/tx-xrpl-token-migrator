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

* Set public variables

```bash
export COREUM_CHAIN_ID="{Chain ID}"
export COREUM_CONTRACT_TRUSTED_ADDRESSES="{Trusted address 2,trusted address 1}"
export COREUM_CONTRACT_THRESHOLD="{Threshold}"
export COREUM_CONTRACT_OWNER="{Owner which is able to withdraw contract balance}"
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
    --coreum-sender-address $(./relayer keys show contract-deployer -a --coreum-chain-id $COREUM_CHAIN_ID --keyring-backend os --home $HOME/.xrpl-bridge)
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
export PROMETHEUS_LOGIN="{Prometheus login}"
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
    --prometheus-login $PROMETHEUS_LOGIN \\
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
