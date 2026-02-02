//go:build integrationtests

package evm

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"

	bscabi "github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bsc/abi"
)

// ERC1967Proxy bytecode (OpenZeppelin v5, compiled with Solidity 0.8.24)
// Constructor: constructor(address implementation, bytes memory _data)
// Compiled from @openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol using forge
const erc1967ProxyBytecode = "0x608060405234801561000f575f80fd5b506040516106af3803806106af8339818101604052810190610031919061054d565b818161004161009e60201b60201c565b15801561004e57505f8151145b15610085576040517fc28a273c00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b61009582826100a260201b60201c565b505050506105cf565b5f90565b6100b18261012660201b60201c565b8173ffffffffffffffffffffffffffffffffffffffff167fbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b60405160405180910390a25f815111156101135761010d82826101f560201b60201c565b50610122565b61012161030460201b60201c565b5b5050565b5f8173ffffffffffffffffffffffffffffffffffffffff163b0361018157806040517f4c9c8ce300000000000000000000000000000000000000000000000000000000815260040161017891906105b6565b60405180910390fd5b806101b37f360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc5f1b61034060201b60201c565b5f015f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b60605f610208848461034960201b60201c565b905080801561024457505f61022161035d60201b60201c565b118061024357505f8473ffffffffffffffffffffffffffffffffffffffff163b115b5b1561025f5761025761036460201b60201c565b9150506102fe565b80156102a257836040517f9996b31500000000000000000000000000000000000000000000000000000000815260040161029991906105b6565b60405180910390fd5b5f6102b161035d60201b60201c565b11156102ca576102c561038160201b60201c565b6102fc565b6040517fd6bda27500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b505b92915050565b5f34111561033e576040517fb398979f00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b565b5f819050919050565b5f805f835160208501865af4905092915050565b5f3d905090565b606060405190503d81523d5f602083013e3d602001810160405290565b6040513d5f823e3d81fd5b5f604051905090565b5f80fd5b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6103c68261039d565b9050919050565b6103d6816103bc565b81146103e0575f80fd5b50565b5f815190506103f1816103cd565b92915050565b5f80fd5b5f80fd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b610445826103ff565b810181811067ffffffffffffffff821117156104645761046361040f565b5b80604052505050565b5f61047661038c565b9050610482828261043c565b919050565b5f67ffffffffffffffff8211156104a1576104a061040f565b5b6104aa826103ff565b9050602081019050919050565b5f5b838110156104d45780820151818401526020810190506104b9565b5f8484015250505050565b5f6104f16104ec84610487565b61046d565b90508281526020810184848401111561050d5761050c6103fb565b5b6105188482856104b7565b509392505050565b5f82601f830112610534576105336103f7565b5b81516105448482602086016104df565b91505092915050565b5f806040838503121561056357610562610395565b5b5f610570858286016103e3565b925050602083015167ffffffffffffffff81111561059157610590610399565b5b61059d85828601610520565b9150509250929050565b6105b0816103bc565b82525050565b5f6020820190506105c95f8301846105a7565b92915050565b60d4806105db5f395ff3fe6080604052600a600c565b005b60186014601a565b6026565b565b5f60216044565b905090565b365f80375f80365f845af43d5f803e805f81146040573d5ff35b3d5ffd5b5f606e7f360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc5f1b6095565b5f015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b5f81905091905056fea26469706673582212207f5275895ccd6a9019243ddfdd82dc77ec91c8df50b415bd6eaa24218a0c5cce64736f6c63430008180033"

// represents a Hardhat/Foundry compiled contract artifact.
type ContractArtifact struct {
	ABI      json.RawMessage `json:"abi"`
	Bytecode string          `json:"bytecode"`
}

// holds addresses of deployed contracts.
type DeployedContracts struct {
	TokenAddress  common.Address
	BridgeAddress common.Address
	Token         *bscabi.TxToken
	Bridge        *bscabi.TxBridge
}

// holds configuration for the bridge contract.
type BridgeConfig struct {
	MinAmount     *big.Int
	MaxAmount     *big.Int
	AddressPrefix string
}

// returns default bridge configuration for testing.
func DefaultBridgeConfig() BridgeConfig {
	return BridgeConfig{
		MinAmount:     big.NewInt(1000000),     // 1 token (6 decimals)
		MaxAmount:     big.NewInt(50000000000), // 50,000 tokens (6 decimals)
		AddressPrefix: "devcore",
	}
}

// loads a contract artifact from embedded files.
func loadArtifact(name string) (*ContractArtifact, error) {
	data, err := bscabi.ArtifactFiles.ReadFile(name + ".json")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read artifact %s", name)
	}

	var artifact ContractArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, errors.Wrapf(err, "failed to parse artifact %s", name)
	}

	return &artifact, nil
}

// creates transaction options for a given private key.
func getTransactOpts(ctx context.Context, client *ethclient.Client, privateKey *ecdsa.PrivateKey, chainID *big.Int) (*bind.TransactOpts, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create transactor")
	}

	nonce, err := client.PendingNonceAt(ctx, auth.From)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get nonce")
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get gas price")
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.GasPrice = gasPrice
	auth.GasLimit = 5000000

	return auth, nil
}

// deploys a contract and returns its address.
func deployContract(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts, bytecode string, constructorArgs ...[]byte) (common.Address, *types.Transaction, error) {
	bytecode = strings.TrimPrefix(bytecode, "0x")

	// combine bytecode with constructor args
	data := common.Hex2Bytes(bytecode)
	for _, arg := range constructorArgs {
		data = append(data, arg...)
	}

	tx := types.NewContractCreation(
		auth.Nonce.Uint64(),
		big.NewInt(0),
		auth.GasLimit,
		auth.GasPrice,
		data,
	)

	signedTx, err := auth.Signer(auth.From, tx)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "failed to sign transaction")
	}

	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return common.Address{}, nil, errors.Wrap(err, "failed to send transaction")
	}

	receipt, err := bind.WaitMined(ctx, client, signedTx)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "failed to wait for mining")
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return common.Address{}, nil, errors.Errorf("contract deployment failed: tx=%s, gas_used=%d", receipt.TxHash.Hex(), receipt.GasUsed)
	}

	return receipt.ContractAddress, signedTx, nil
}

// encodeInitializeData encodes initialization call data for a proxy.
func encodeInitializeData(contractABI abi.ABI, args ...interface{}) ([]byte, error) {
	return contractABI.Pack("initialize", args...)
}

// encodes the ERC1967Proxy constructor arguments.
func encodeProxyConstructor(implementation common.Address, initData []byte) ([]byte, error) {
	// ERC1967Proxy constructor: (address implementation, bytes memory _data)
	addressType, _ := abi.NewType("address", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)

	args := abi.Arguments{
		{Type: addressType},
		{Type: bytesType},
	}

	return args.Pack(implementation, initData)
}

// deploys the TXToken contract through a proxy.
func DeployTXToken(ctx context.Context, client *ethclient.Client, privateKey *ecdsa.PrivateKey, chainID *big.Int, name, symbol string) (common.Address, *bscabi.TxToken, error) {
	auth, err := getTransactOpts(ctx, client, privateKey, chainID)
	if err != nil {
		return common.Address{}, nil, err
	}

	// load artifact
	artifact, err := loadArtifact("TXToken")
	if err != nil {
		return common.Address{}, nil, err
	}

	// deploy implementation
	implAddress, _, err := deployContract(ctx, client, auth, artifact.Bytecode)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "failed to deploy token implementation")
	}

	// get owner address
	owner := crypto.PubkeyToAddress(privateKey.PublicKey)

	// encode initialize call
	tokenABI, err := bscabi.TxTokenMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "failed to get token ABI")
	}

	initData, err := encodeInitializeData(*tokenABI, name, symbol, owner)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "failed to encode initialize data")
	}

	// encode proxy constructor
	proxyArgs, err := encodeProxyConstructor(implAddress, initData)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "failed to encode proxy constructor")
	}

	// deploy proxy - need new nonce
	auth, err = getTransactOpts(ctx, client, privateKey, chainID)
	if err != nil {
		return common.Address{}, nil, err
	}

	proxyAddress, _, err := deployContract(ctx, client, auth, erc1967ProxyBytecode, proxyArgs)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "failed to deploy token proxy")
	}

	// create binding to proxy
	token, err := bscabi.NewTxToken(proxyAddress, client)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "failed to create token binding")
	}

	return proxyAddress, token, nil
}

// deploys the TXBridge contract through a proxy.
func DeployTXBridge(ctx context.Context, client *ethclient.Client, privateKey *ecdsa.PrivateKey, chainID *big.Int, tokenAddress common.Address, cfg BridgeConfig) (common.Address, *bscabi.TxBridge, error) {
	auth, err := getTransactOpts(ctx, client, privateKey, chainID)
	if err != nil {
		return common.Address{}, nil, err
	}

	artifact, err := loadArtifact("TXBridge")
	if err != nil {
		return common.Address{}, nil, err
	}

	implAddress, _, err := deployContract(ctx, client, auth, artifact.Bytecode)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "failed to deploy bridge implementation")
	}

	admin := crypto.PubkeyToAddress(privateKey.PublicKey)

	bridgeABI, err := bscabi.TxBridgeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "failed to get bridge ABI")
	}

	initData, err := encodeInitializeData(*bridgeABI,
		tokenAddress,
		admin,
		cfg.MinAmount,
		cfg.MaxAmount,
		cfg.AddressPrefix,
	)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "failed to encode initialize data")
	}

	proxyArgs, err := encodeProxyConstructor(implAddress, initData)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "failed to encode proxy constructor")
	}

	auth, err = getTransactOpts(ctx, client, privateKey, chainID)
	if err != nil {
		return common.Address{}, nil, err
	}

	proxyAddress, _, err := deployContract(ctx, client, auth, erc1967ProxyBytecode, proxyArgs)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "failed to deploy bridge proxy")
	}

	bridge, err := bscabi.NewTxBridge(proxyAddress, client)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "failed to create bridge binding")
	}

	return proxyAddress, bridge, nil
}

// deploys both contracts and configures them.
func SetupBridgeEnvironment(ctx context.Context, client *ethclient.Client, privateKey *ecdsa.PrivateKey, chainID *big.Int, cfg BridgeConfig) (*DeployedContracts, error) {
	tokenAddress, token, err := DeployTXToken(ctx, client, privateKey, chainID, "tx Token", "TX")
	if err != nil {
		return nil, errors.Wrap(err, "failed to deploy token")
	}

	// Deploy bridge
	bridgeAddress, bridge, err := DeployTXBridge(ctx, client, privateKey, chainID, tokenAddress, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deploy bridge")
	}

	// Grant BRIDGE_ROLE to bridge contract on token
	auth, err := getTransactOpts(ctx, client, privateKey, chainID)
	if err != nil {
		return nil, err
	}

	// Get BRIDGE_ROLE hash
	bridgeRole, err := token.BRIDGEROLE(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get BRIDGE_ROLE")
	}

	// Grant role
	tx, err := token.GrantRole(auth, bridgeRole, bridgeAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to grant BRIDGE_ROLE")
	}

	// Finalize
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wait for grant role tx")
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, errors.New("grant role transaction failed")
	}

	return &DeployedContracts{
		TokenAddress:  tokenAddress,
		BridgeAddress: bridgeAddress,
		Token:         token,
		Bridge:        bridge,
	}, nil
}

// mints tokens to a specified address.
func MintTokens(ctx context.Context, client *ethclient.Client, privateKey *ecdsa.PrivateKey, chainID *big.Int, token *bscabi.TxToken, to common.Address, amount *big.Int) error {
	auth, err := getTransactOpts(ctx, client, privateKey, chainID)
	if err != nil {
		return err
	}

	tx, err := token.Mint(auth, to, amount)
	if err != nil {
		return errors.Wrap(err, "failed to mint tokens")
	}

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return errors.Wrap(err, "failed to wait for mint tx")
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return errors.New("mint transaction failed")
	}

	return nil
}
