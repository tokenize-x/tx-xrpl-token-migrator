// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package abi

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// TxBridgeMetaData contains all meta data concerning the TxBridge contract.
var TxBridgeMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AccessControlBadConfirmation\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"neededRole\",\"type\":\"bytes32\"}],\"name\":\"AccessControlUnauthorizedAccount\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maximum\",\"type\":\"uint256\"}],\"name\":\"AmountAboveMaximum\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"minimum\",\"type\":\"uint256\"}],\"name\":\"AmountBelowMinimum\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyString\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EnforcedPause\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ExpectedPause\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidAddressDataLength\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidAddressPrefix\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidAmount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidChainSuffix\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidInitialization\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotInitializing\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ReentrancyGuardReentrantCall\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroAddress\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"txchainAddress\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"BridgeInitiated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"version\",\"type\":\"uint64\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"previousAdminRole\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"newAdminRole\",\"type\":\"bytes32\"}],\"name\":\"RoleAdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleGranted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"oldToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newToken\",\"type\":\"address\"}],\"name\":\"TokenUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEFAULT_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OPERATOR_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"addressDataLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"addressPrefix\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"txchainAddress\",\"type\":\"string\"}],\"name\":\"bridge\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"chainSuffix\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleAdmin\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"grantRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRole\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_admin\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_minAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_maxAmount\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"_chainSuffix\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_addressPrefix\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"_addressDataLength\",\"type\":\"uint256\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"txchainAddress\",\"type\":\"string\"}],\"name\":\"isValidAddress\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"callerConfirmation\",\"type\":\"address\"}],\"name\":\"renounceRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_addressDataLength\",\"type\":\"uint256\"}],\"name\":\"setAddressDataLength\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_addressPrefix\",\"type\":\"string\"}],\"name\":\"setAddressPrefix\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_chainSuffix\",\"type\":\"string\"}],\"name\":\"setChainSuffix\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_maxAmount\",\"type\":\"uint256\"}],\"name\":\"setMaxAmount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_minAmount\",\"type\":\"uint256\"}],\"name\":\"setMinAmount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"}],\"name\":\"setToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token\",\"outputs\":[{\"internalType\":\"contractITxToken\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// TxBridgeABI is the input ABI used to generate the binding from.
// Deprecated: Use TxBridgeMetaData.ABI instead.
var TxBridgeABI = TxBridgeMetaData.ABI

// TxBridge is an auto generated Go binding around an Ethereum contract.
type TxBridge struct {
	TxBridgeCaller     // Read-only binding to the contract
	TxBridgeTransactor // Write-only binding to the contract
	TxBridgeFilterer   // Log filterer for contract events
}

// TxBridgeCaller is an auto generated read-only Go binding around an Ethereum contract.
type TxBridgeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TxBridgeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TxBridgeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TxBridgeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TxBridgeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TxBridgeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TxBridgeSession struct {
	Contract     *TxBridge         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TxBridgeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TxBridgeCallerSession struct {
	Contract *TxBridgeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// TxBridgeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TxBridgeTransactorSession struct {
	Contract     *TxBridgeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// TxBridgeRaw is an auto generated low-level Go binding around an Ethereum contract.
type TxBridgeRaw struct {
	Contract *TxBridge // Generic contract binding to access the raw methods on
}

// TxBridgeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TxBridgeCallerRaw struct {
	Contract *TxBridgeCaller // Generic read-only contract binding to access the raw methods on
}

// TxBridgeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TxBridgeTransactorRaw struct {
	Contract *TxBridgeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTxBridge creates a new instance of TxBridge, bound to a specific deployed contract.
func NewTxBridge(address common.Address, backend bind.ContractBackend) (*TxBridge, error) {
	contract, err := bindTxBridge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TxBridge{TxBridgeCaller: TxBridgeCaller{contract: contract}, TxBridgeTransactor: TxBridgeTransactor{contract: contract}, TxBridgeFilterer: TxBridgeFilterer{contract: contract}}, nil
}

// NewTxBridgeCaller creates a new read-only instance of TxBridge, bound to a specific deployed contract.
func NewTxBridgeCaller(address common.Address, caller bind.ContractCaller) (*TxBridgeCaller, error) {
	contract, err := bindTxBridge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TxBridgeCaller{contract: contract}, nil
}

// NewTxBridgeTransactor creates a new write-only instance of TxBridge, bound to a specific deployed contract.
func NewTxBridgeTransactor(address common.Address, transactor bind.ContractTransactor) (*TxBridgeTransactor, error) {
	contract, err := bindTxBridge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TxBridgeTransactor{contract: contract}, nil
}

// NewTxBridgeFilterer creates a new log filterer instance of TxBridge, bound to a specific deployed contract.
func NewTxBridgeFilterer(address common.Address, filterer bind.ContractFilterer) (*TxBridgeFilterer, error) {
	contract, err := bindTxBridge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TxBridgeFilterer{contract: contract}, nil
}

// bindTxBridge binds a generic wrapper to an already deployed contract.
func bindTxBridge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TxBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TxBridge *TxBridgeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TxBridge.Contract.TxBridgeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TxBridge *TxBridgeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TxBridge.Contract.TxBridgeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TxBridge *TxBridgeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TxBridge.Contract.TxBridgeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TxBridge *TxBridgeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TxBridge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TxBridge *TxBridgeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TxBridge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TxBridge *TxBridgeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TxBridge.Contract.contract.Transact(opts, method, params...)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_TxBridge *TxBridgeCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _TxBridge.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_TxBridge *TxBridgeSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _TxBridge.Contract.DEFAULTADMINROLE(&_TxBridge.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_TxBridge *TxBridgeCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _TxBridge.Contract.DEFAULTADMINROLE(&_TxBridge.CallOpts)
}

// OPERATORROLE is a free data retrieval call binding the contract method 0xf5b541a6.
//
// Solidity: function OPERATOR_ROLE() view returns(bytes32)
func (_TxBridge *TxBridgeCaller) OPERATORROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _TxBridge.contract.Call(opts, &out, "OPERATOR_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// OPERATORROLE is a free data retrieval call binding the contract method 0xf5b541a6.
//
// Solidity: function OPERATOR_ROLE() view returns(bytes32)
func (_TxBridge *TxBridgeSession) OPERATORROLE() ([32]byte, error) {
	return _TxBridge.Contract.OPERATORROLE(&_TxBridge.CallOpts)
}

// OPERATORROLE is a free data retrieval call binding the contract method 0xf5b541a6.
//
// Solidity: function OPERATOR_ROLE() view returns(bytes32)
func (_TxBridge *TxBridgeCallerSession) OPERATORROLE() ([32]byte, error) {
	return _TxBridge.Contract.OPERATORROLE(&_TxBridge.CallOpts)
}

// AddressDataLength is a free data retrieval call binding the contract method 0x31c5a374.
//
// Solidity: function addressDataLength() view returns(uint256)
func (_TxBridge *TxBridgeCaller) AddressDataLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TxBridge.contract.Call(opts, &out, "addressDataLength")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// AddressDataLength is a free data retrieval call binding the contract method 0x31c5a374.
//
// Solidity: function addressDataLength() view returns(uint256)
func (_TxBridge *TxBridgeSession) AddressDataLength() (*big.Int, error) {
	return _TxBridge.Contract.AddressDataLength(&_TxBridge.CallOpts)
}

// AddressDataLength is a free data retrieval call binding the contract method 0x31c5a374.
//
// Solidity: function addressDataLength() view returns(uint256)
func (_TxBridge *TxBridgeCallerSession) AddressDataLength() (*big.Int, error) {
	return _TxBridge.Contract.AddressDataLength(&_TxBridge.CallOpts)
}

// AddressPrefix is a free data retrieval call binding the contract method 0xf0e0de8f.
//
// Solidity: function addressPrefix() view returns(string)
func (_TxBridge *TxBridgeCaller) AddressPrefix(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _TxBridge.contract.Call(opts, &out, "addressPrefix")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// AddressPrefix is a free data retrieval call binding the contract method 0xf0e0de8f.
//
// Solidity: function addressPrefix() view returns(string)
func (_TxBridge *TxBridgeSession) AddressPrefix() (string, error) {
	return _TxBridge.Contract.AddressPrefix(&_TxBridge.CallOpts)
}

// AddressPrefix is a free data retrieval call binding the contract method 0xf0e0de8f.
//
// Solidity: function addressPrefix() view returns(string)
func (_TxBridge *TxBridgeCallerSession) AddressPrefix() (string, error) {
	return _TxBridge.Contract.AddressPrefix(&_TxBridge.CallOpts)
}

// ChainSuffix is a free data retrieval call binding the contract method 0xbb76bc0a.
//
// Solidity: function chainSuffix() view returns(string)
func (_TxBridge *TxBridgeCaller) ChainSuffix(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _TxBridge.contract.Call(opts, &out, "chainSuffix")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// ChainSuffix is a free data retrieval call binding the contract method 0xbb76bc0a.
//
// Solidity: function chainSuffix() view returns(string)
func (_TxBridge *TxBridgeSession) ChainSuffix() (string, error) {
	return _TxBridge.Contract.ChainSuffix(&_TxBridge.CallOpts)
}

// ChainSuffix is a free data retrieval call binding the contract method 0xbb76bc0a.
//
// Solidity: function chainSuffix() view returns(string)
func (_TxBridge *TxBridgeCallerSession) ChainSuffix() (string, error) {
	return _TxBridge.Contract.ChainSuffix(&_TxBridge.CallOpts)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_TxBridge *TxBridgeCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _TxBridge.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_TxBridge *TxBridgeSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _TxBridge.Contract.GetRoleAdmin(&_TxBridge.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_TxBridge *TxBridgeCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _TxBridge.Contract.GetRoleAdmin(&_TxBridge.CallOpts, role)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_TxBridge *TxBridgeCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _TxBridge.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_TxBridge *TxBridgeSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _TxBridge.Contract.HasRole(&_TxBridge.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_TxBridge *TxBridgeCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _TxBridge.Contract.HasRole(&_TxBridge.CallOpts, role, account)
}

// IsValidAddress is a free data retrieval call binding the contract method 0x335b08ba.
//
// Solidity: function isValidAddress(string txchainAddress) view returns(bool)
func (_TxBridge *TxBridgeCaller) IsValidAddress(opts *bind.CallOpts, txchainAddress string) (bool, error) {
	var out []interface{}
	err := _TxBridge.contract.Call(opts, &out, "isValidAddress", txchainAddress)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsValidAddress is a free data retrieval call binding the contract method 0x335b08ba.
//
// Solidity: function isValidAddress(string txchainAddress) view returns(bool)
func (_TxBridge *TxBridgeSession) IsValidAddress(txchainAddress string) (bool, error) {
	return _TxBridge.Contract.IsValidAddress(&_TxBridge.CallOpts, txchainAddress)
}

// IsValidAddress is a free data retrieval call binding the contract method 0x335b08ba.
//
// Solidity: function isValidAddress(string txchainAddress) view returns(bool)
func (_TxBridge *TxBridgeCallerSession) IsValidAddress(txchainAddress string) (bool, error) {
	return _TxBridge.Contract.IsValidAddress(&_TxBridge.CallOpts, txchainAddress)
}

// MaxAmount is a free data retrieval call binding the contract method 0x5f48f393.
//
// Solidity: function maxAmount() view returns(uint256)
func (_TxBridge *TxBridgeCaller) MaxAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TxBridge.contract.Call(opts, &out, "maxAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxAmount is a free data retrieval call binding the contract method 0x5f48f393.
//
// Solidity: function maxAmount() view returns(uint256)
func (_TxBridge *TxBridgeSession) MaxAmount() (*big.Int, error) {
	return _TxBridge.Contract.MaxAmount(&_TxBridge.CallOpts)
}

// MaxAmount is a free data retrieval call binding the contract method 0x5f48f393.
//
// Solidity: function maxAmount() view returns(uint256)
func (_TxBridge *TxBridgeCallerSession) MaxAmount() (*big.Int, error) {
	return _TxBridge.Contract.MaxAmount(&_TxBridge.CallOpts)
}

// MinAmount is a free data retrieval call binding the contract method 0x9b2cb5d8.
//
// Solidity: function minAmount() view returns(uint256)
func (_TxBridge *TxBridgeCaller) MinAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TxBridge.contract.Call(opts, &out, "minAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinAmount is a free data retrieval call binding the contract method 0x9b2cb5d8.
//
// Solidity: function minAmount() view returns(uint256)
func (_TxBridge *TxBridgeSession) MinAmount() (*big.Int, error) {
	return _TxBridge.Contract.MinAmount(&_TxBridge.CallOpts)
}

// MinAmount is a free data retrieval call binding the contract method 0x9b2cb5d8.
//
// Solidity: function minAmount() view returns(uint256)
func (_TxBridge *TxBridgeCallerSession) MinAmount() (*big.Int, error) {
	return _TxBridge.Contract.MinAmount(&_TxBridge.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_TxBridge *TxBridgeCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _TxBridge.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_TxBridge *TxBridgeSession) Paused() (bool, error) {
	return _TxBridge.Contract.Paused(&_TxBridge.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_TxBridge *TxBridgeCallerSession) Paused() (bool, error) {
	return _TxBridge.Contract.Paused(&_TxBridge.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_TxBridge *TxBridgeCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _TxBridge.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_TxBridge *TxBridgeSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _TxBridge.Contract.SupportsInterface(&_TxBridge.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_TxBridge *TxBridgeCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _TxBridge.Contract.SupportsInterface(&_TxBridge.CallOpts, interfaceId)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_TxBridge *TxBridgeCaller) Token(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _TxBridge.contract.Call(opts, &out, "token")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_TxBridge *TxBridgeSession) Token() (common.Address, error) {
	return _TxBridge.Contract.Token(&_TxBridge.CallOpts)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_TxBridge *TxBridgeCallerSession) Token() (common.Address, error) {
	return _TxBridge.Contract.Token(&_TxBridge.CallOpts)
}

// Bridge is a paid mutator transaction binding the contract method 0xaaf4ce4a.
//
// Solidity: function bridge(uint256 amount, string txchainAddress) returns()
func (_TxBridge *TxBridgeTransactor) Bridge(opts *bind.TransactOpts, amount *big.Int, txchainAddress string) (*types.Transaction, error) {
	return _TxBridge.contract.Transact(opts, "bridge", amount, txchainAddress)
}

// Bridge is a paid mutator transaction binding the contract method 0xaaf4ce4a.
//
// Solidity: function bridge(uint256 amount, string txchainAddress) returns()
func (_TxBridge *TxBridgeSession) Bridge(amount *big.Int, txchainAddress string) (*types.Transaction, error) {
	return _TxBridge.Contract.Bridge(&_TxBridge.TransactOpts, amount, txchainAddress)
}

// Bridge is a paid mutator transaction binding the contract method 0xaaf4ce4a.
//
// Solidity: function bridge(uint256 amount, string txchainAddress) returns()
func (_TxBridge *TxBridgeTransactorSession) Bridge(amount *big.Int, txchainAddress string) (*types.Transaction, error) {
	return _TxBridge.Contract.Bridge(&_TxBridge.TransactOpts, amount, txchainAddress)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_TxBridge *TxBridgeTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _TxBridge.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_TxBridge *TxBridgeSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _TxBridge.Contract.GrantRole(&_TxBridge.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_TxBridge *TxBridgeTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _TxBridge.Contract.GrantRole(&_TxBridge.TransactOpts, role, account)
}

// Initialize is a paid mutator transaction binding the contract method 0x983e49b3.
//
// Solidity: function initialize(address _token, address _admin, uint256 _minAmount, uint256 _maxAmount, string _chainSuffix, string _addressPrefix, uint256 _addressDataLength) returns()
func (_TxBridge *TxBridgeTransactor) Initialize(opts *bind.TransactOpts, _token common.Address, _admin common.Address, _minAmount *big.Int, _maxAmount *big.Int, _chainSuffix string, _addressPrefix string, _addressDataLength *big.Int) (*types.Transaction, error) {
	return _TxBridge.contract.Transact(opts, "initialize", _token, _admin, _minAmount, _maxAmount, _chainSuffix, _addressPrefix, _addressDataLength)
}

// Initialize is a paid mutator transaction binding the contract method 0x983e49b3.
//
// Solidity: function initialize(address _token, address _admin, uint256 _minAmount, uint256 _maxAmount, string _chainSuffix, string _addressPrefix, uint256 _addressDataLength) returns()
func (_TxBridge *TxBridgeSession) Initialize(_token common.Address, _admin common.Address, _minAmount *big.Int, _maxAmount *big.Int, _chainSuffix string, _addressPrefix string, _addressDataLength *big.Int) (*types.Transaction, error) {
	return _TxBridge.Contract.Initialize(&_TxBridge.TransactOpts, _token, _admin, _minAmount, _maxAmount, _chainSuffix, _addressPrefix, _addressDataLength)
}

// Initialize is a paid mutator transaction binding the contract method 0x983e49b3.
//
// Solidity: function initialize(address _token, address _admin, uint256 _minAmount, uint256 _maxAmount, string _chainSuffix, string _addressPrefix, uint256 _addressDataLength) returns()
func (_TxBridge *TxBridgeTransactorSession) Initialize(_token common.Address, _admin common.Address, _minAmount *big.Int, _maxAmount *big.Int, _chainSuffix string, _addressPrefix string, _addressDataLength *big.Int) (*types.Transaction, error) {
	return _TxBridge.Contract.Initialize(&_TxBridge.TransactOpts, _token, _admin, _minAmount, _maxAmount, _chainSuffix, _addressPrefix, _addressDataLength)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_TxBridge *TxBridgeTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TxBridge.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_TxBridge *TxBridgeSession) Pause() (*types.Transaction, error) {
	return _TxBridge.Contract.Pause(&_TxBridge.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_TxBridge *TxBridgeTransactorSession) Pause() (*types.Transaction, error) {
	return _TxBridge.Contract.Pause(&_TxBridge.TransactOpts)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address callerConfirmation) returns()
func (_TxBridge *TxBridgeTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, callerConfirmation common.Address) (*types.Transaction, error) {
	return _TxBridge.contract.Transact(opts, "renounceRole", role, callerConfirmation)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address callerConfirmation) returns()
func (_TxBridge *TxBridgeSession) RenounceRole(role [32]byte, callerConfirmation common.Address) (*types.Transaction, error) {
	return _TxBridge.Contract.RenounceRole(&_TxBridge.TransactOpts, role, callerConfirmation)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address callerConfirmation) returns()
func (_TxBridge *TxBridgeTransactorSession) RenounceRole(role [32]byte, callerConfirmation common.Address) (*types.Transaction, error) {
	return _TxBridge.Contract.RenounceRole(&_TxBridge.TransactOpts, role, callerConfirmation)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_TxBridge *TxBridgeTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _TxBridge.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_TxBridge *TxBridgeSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _TxBridge.Contract.RevokeRole(&_TxBridge.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_TxBridge *TxBridgeTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _TxBridge.Contract.RevokeRole(&_TxBridge.TransactOpts, role, account)
}

// SetAddressDataLength is a paid mutator transaction binding the contract method 0x8d3cf6a5.
//
// Solidity: function setAddressDataLength(uint256 _addressDataLength) returns()
func (_TxBridge *TxBridgeTransactor) SetAddressDataLength(opts *bind.TransactOpts, _addressDataLength *big.Int) (*types.Transaction, error) {
	return _TxBridge.contract.Transact(opts, "setAddressDataLength", _addressDataLength)
}

// SetAddressDataLength is a paid mutator transaction binding the contract method 0x8d3cf6a5.
//
// Solidity: function setAddressDataLength(uint256 _addressDataLength) returns()
func (_TxBridge *TxBridgeSession) SetAddressDataLength(_addressDataLength *big.Int) (*types.Transaction, error) {
	return _TxBridge.Contract.SetAddressDataLength(&_TxBridge.TransactOpts, _addressDataLength)
}

// SetAddressDataLength is a paid mutator transaction binding the contract method 0x8d3cf6a5.
//
// Solidity: function setAddressDataLength(uint256 _addressDataLength) returns()
func (_TxBridge *TxBridgeTransactorSession) SetAddressDataLength(_addressDataLength *big.Int) (*types.Transaction, error) {
	return _TxBridge.Contract.SetAddressDataLength(&_TxBridge.TransactOpts, _addressDataLength)
}

// SetAddressPrefix is a paid mutator transaction binding the contract method 0xb8e4e61d.
//
// Solidity: function setAddressPrefix(string _addressPrefix) returns()
func (_TxBridge *TxBridgeTransactor) SetAddressPrefix(opts *bind.TransactOpts, _addressPrefix string) (*types.Transaction, error) {
	return _TxBridge.contract.Transact(opts, "setAddressPrefix", _addressPrefix)
}

// SetAddressPrefix is a paid mutator transaction binding the contract method 0xb8e4e61d.
//
// Solidity: function setAddressPrefix(string _addressPrefix) returns()
func (_TxBridge *TxBridgeSession) SetAddressPrefix(_addressPrefix string) (*types.Transaction, error) {
	return _TxBridge.Contract.SetAddressPrefix(&_TxBridge.TransactOpts, _addressPrefix)
}

// SetAddressPrefix is a paid mutator transaction binding the contract method 0xb8e4e61d.
//
// Solidity: function setAddressPrefix(string _addressPrefix) returns()
func (_TxBridge *TxBridgeTransactorSession) SetAddressPrefix(_addressPrefix string) (*types.Transaction, error) {
	return _TxBridge.Contract.SetAddressPrefix(&_TxBridge.TransactOpts, _addressPrefix)
}

// SetChainSuffix is a paid mutator transaction binding the contract method 0x2f880fe4.
//
// Solidity: function setChainSuffix(string _chainSuffix) returns()
func (_TxBridge *TxBridgeTransactor) SetChainSuffix(opts *bind.TransactOpts, _chainSuffix string) (*types.Transaction, error) {
	return _TxBridge.contract.Transact(opts, "setChainSuffix", _chainSuffix)
}

// SetChainSuffix is a paid mutator transaction binding the contract method 0x2f880fe4.
//
// Solidity: function setChainSuffix(string _chainSuffix) returns()
func (_TxBridge *TxBridgeSession) SetChainSuffix(_chainSuffix string) (*types.Transaction, error) {
	return _TxBridge.Contract.SetChainSuffix(&_TxBridge.TransactOpts, _chainSuffix)
}

// SetChainSuffix is a paid mutator transaction binding the contract method 0x2f880fe4.
//
// Solidity: function setChainSuffix(string _chainSuffix) returns()
func (_TxBridge *TxBridgeTransactorSession) SetChainSuffix(_chainSuffix string) (*types.Transaction, error) {
	return _TxBridge.Contract.SetChainSuffix(&_TxBridge.TransactOpts, _chainSuffix)
}

// SetMaxAmount is a paid mutator transaction binding the contract method 0x4fe47f70.
//
// Solidity: function setMaxAmount(uint256 _maxAmount) returns()
func (_TxBridge *TxBridgeTransactor) SetMaxAmount(opts *bind.TransactOpts, _maxAmount *big.Int) (*types.Transaction, error) {
	return _TxBridge.contract.Transact(opts, "setMaxAmount", _maxAmount)
}

// SetMaxAmount is a paid mutator transaction binding the contract method 0x4fe47f70.
//
// Solidity: function setMaxAmount(uint256 _maxAmount) returns()
func (_TxBridge *TxBridgeSession) SetMaxAmount(_maxAmount *big.Int) (*types.Transaction, error) {
	return _TxBridge.Contract.SetMaxAmount(&_TxBridge.TransactOpts, _maxAmount)
}

// SetMaxAmount is a paid mutator transaction binding the contract method 0x4fe47f70.
//
// Solidity: function setMaxAmount(uint256 _maxAmount) returns()
func (_TxBridge *TxBridgeTransactorSession) SetMaxAmount(_maxAmount *big.Int) (*types.Transaction, error) {
	return _TxBridge.Contract.SetMaxAmount(&_TxBridge.TransactOpts, _maxAmount)
}

// SetMinAmount is a paid mutator transaction binding the contract method 0x897b0637.
//
// Solidity: function setMinAmount(uint256 _minAmount) returns()
func (_TxBridge *TxBridgeTransactor) SetMinAmount(opts *bind.TransactOpts, _minAmount *big.Int) (*types.Transaction, error) {
	return _TxBridge.contract.Transact(opts, "setMinAmount", _minAmount)
}

// SetMinAmount is a paid mutator transaction binding the contract method 0x897b0637.
//
// Solidity: function setMinAmount(uint256 _minAmount) returns()
func (_TxBridge *TxBridgeSession) SetMinAmount(_minAmount *big.Int) (*types.Transaction, error) {
	return _TxBridge.Contract.SetMinAmount(&_TxBridge.TransactOpts, _minAmount)
}

// SetMinAmount is a paid mutator transaction binding the contract method 0x897b0637.
//
// Solidity: function setMinAmount(uint256 _minAmount) returns()
func (_TxBridge *TxBridgeTransactorSession) SetMinAmount(_minAmount *big.Int) (*types.Transaction, error) {
	return _TxBridge.Contract.SetMinAmount(&_TxBridge.TransactOpts, _minAmount)
}

// SetToken is a paid mutator transaction binding the contract method 0x144fa6d7.
//
// Solidity: function setToken(address _token) returns()
func (_TxBridge *TxBridgeTransactor) SetToken(opts *bind.TransactOpts, _token common.Address) (*types.Transaction, error) {
	return _TxBridge.contract.Transact(opts, "setToken", _token)
}

// SetToken is a paid mutator transaction binding the contract method 0x144fa6d7.
//
// Solidity: function setToken(address _token) returns()
func (_TxBridge *TxBridgeSession) SetToken(_token common.Address) (*types.Transaction, error) {
	return _TxBridge.Contract.SetToken(&_TxBridge.TransactOpts, _token)
}

// SetToken is a paid mutator transaction binding the contract method 0x144fa6d7.
//
// Solidity: function setToken(address _token) returns()
func (_TxBridge *TxBridgeTransactorSession) SetToken(_token common.Address) (*types.Transaction, error) {
	return _TxBridge.Contract.SetToken(&_TxBridge.TransactOpts, _token)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_TxBridge *TxBridgeTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TxBridge.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_TxBridge *TxBridgeSession) Unpause() (*types.Transaction, error) {
	return _TxBridge.Contract.Unpause(&_TxBridge.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_TxBridge *TxBridgeTransactorSession) Unpause() (*types.Transaction, error) {
	return _TxBridge.Contract.Unpause(&_TxBridge.TransactOpts)
}

// TxBridgeBridgeInitiatedIterator is returned from FilterBridgeInitiated and is used to iterate over the raw logs and unpacked data for BridgeInitiated events raised by the TxBridge contract.
type TxBridgeBridgeInitiatedIterator struct {
	Event *TxBridgeBridgeInitiated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TxBridgeBridgeInitiatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TxBridgeBridgeInitiated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TxBridgeBridgeInitiated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TxBridgeBridgeInitiatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TxBridgeBridgeInitiatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TxBridgeBridgeInitiated represents a BridgeInitiated event raised by the TxBridge contract.
type TxBridgeBridgeInitiated struct {
	From           common.Address
	TxchainAddress string
	Amount         *big.Int
	Timestamp      *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterBridgeInitiated is a free log retrieval operation binding the contract event 0xc8fc702ef75cbf37407ffc9dd8b13a87c16d721286a50b10789508f9ed97482f.
//
// Solidity: event BridgeInitiated(address indexed from, string txchainAddress, uint256 amount, uint256 timestamp)
func (_TxBridge *TxBridgeFilterer) FilterBridgeInitiated(opts *bind.FilterOpts, from []common.Address) (*TxBridgeBridgeInitiatedIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _TxBridge.contract.FilterLogs(opts, "BridgeInitiated", fromRule)
	if err != nil {
		return nil, err
	}
	return &TxBridgeBridgeInitiatedIterator{contract: _TxBridge.contract, event: "BridgeInitiated", logs: logs, sub: sub}, nil
}

// WatchBridgeInitiated is a free log subscription operation binding the contract event 0xc8fc702ef75cbf37407ffc9dd8b13a87c16d721286a50b10789508f9ed97482f.
//
// Solidity: event BridgeInitiated(address indexed from, string txchainAddress, uint256 amount, uint256 timestamp)
func (_TxBridge *TxBridgeFilterer) WatchBridgeInitiated(opts *bind.WatchOpts, sink chan<- *TxBridgeBridgeInitiated, from []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _TxBridge.contract.WatchLogs(opts, "BridgeInitiated", fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TxBridgeBridgeInitiated)
				if err := _TxBridge.contract.UnpackLog(event, "BridgeInitiated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBridgeInitiated is a log parse operation binding the contract event 0xc8fc702ef75cbf37407ffc9dd8b13a87c16d721286a50b10789508f9ed97482f.
//
// Solidity: event BridgeInitiated(address indexed from, string txchainAddress, uint256 amount, uint256 timestamp)
func (_TxBridge *TxBridgeFilterer) ParseBridgeInitiated(log types.Log) (*TxBridgeBridgeInitiated, error) {
	event := new(TxBridgeBridgeInitiated)
	if err := _TxBridge.contract.UnpackLog(event, "BridgeInitiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TxBridgeInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the TxBridge contract.
type TxBridgeInitializedIterator struct {
	Event *TxBridgeInitialized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TxBridgeInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TxBridgeInitialized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TxBridgeInitialized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TxBridgeInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TxBridgeInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TxBridgeInitialized represents a Initialized event raised by the TxBridge contract.
type TxBridgeInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_TxBridge *TxBridgeFilterer) FilterInitialized(opts *bind.FilterOpts) (*TxBridgeInitializedIterator, error) {

	logs, sub, err := _TxBridge.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &TxBridgeInitializedIterator{contract: _TxBridge.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_TxBridge *TxBridgeFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *TxBridgeInitialized) (event.Subscription, error) {

	logs, sub, err := _TxBridge.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TxBridgeInitialized)
				if err := _TxBridge.contract.UnpackLog(event, "Initialized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseInitialized is a log parse operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_TxBridge *TxBridgeFilterer) ParseInitialized(log types.Log) (*TxBridgeInitialized, error) {
	event := new(TxBridgeInitialized)
	if err := _TxBridge.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TxBridgePausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the TxBridge contract.
type TxBridgePausedIterator struct {
	Event *TxBridgePaused // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TxBridgePausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TxBridgePaused)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TxBridgePaused)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TxBridgePausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TxBridgePausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TxBridgePaused represents a Paused event raised by the TxBridge contract.
type TxBridgePaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_TxBridge *TxBridgeFilterer) FilterPaused(opts *bind.FilterOpts) (*TxBridgePausedIterator, error) {

	logs, sub, err := _TxBridge.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &TxBridgePausedIterator{contract: _TxBridge.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_TxBridge *TxBridgeFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *TxBridgePaused) (event.Subscription, error) {

	logs, sub, err := _TxBridge.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TxBridgePaused)
				if err := _TxBridge.contract.UnpackLog(event, "Paused", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePaused is a log parse operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_TxBridge *TxBridgeFilterer) ParsePaused(log types.Log) (*TxBridgePaused, error) {
	event := new(TxBridgePaused)
	if err := _TxBridge.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TxBridgeRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the TxBridge contract.
type TxBridgeRoleAdminChangedIterator struct {
	Event *TxBridgeRoleAdminChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TxBridgeRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TxBridgeRoleAdminChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TxBridgeRoleAdminChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TxBridgeRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TxBridgeRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TxBridgeRoleAdminChanged represents a RoleAdminChanged event raised by the TxBridge contract.
type TxBridgeRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_TxBridge *TxBridgeFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*TxBridgeRoleAdminChangedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _TxBridge.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &TxBridgeRoleAdminChangedIterator{contract: _TxBridge.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_TxBridge *TxBridgeFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *TxBridgeRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _TxBridge.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TxBridgeRoleAdminChanged)
				if err := _TxBridge.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleAdminChanged is a log parse operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_TxBridge *TxBridgeFilterer) ParseRoleAdminChanged(log types.Log) (*TxBridgeRoleAdminChanged, error) {
	event := new(TxBridgeRoleAdminChanged)
	if err := _TxBridge.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TxBridgeRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the TxBridge contract.
type TxBridgeRoleGrantedIterator struct {
	Event *TxBridgeRoleGranted // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TxBridgeRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TxBridgeRoleGranted)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TxBridgeRoleGranted)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TxBridgeRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TxBridgeRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TxBridgeRoleGranted represents a RoleGranted event raised by the TxBridge contract.
type TxBridgeRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_TxBridge *TxBridgeFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*TxBridgeRoleGrantedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _TxBridge.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &TxBridgeRoleGrantedIterator{contract: _TxBridge.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_TxBridge *TxBridgeFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *TxBridgeRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _TxBridge.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TxBridgeRoleGranted)
				if err := _TxBridge.contract.UnpackLog(event, "RoleGranted", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleGranted is a log parse operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_TxBridge *TxBridgeFilterer) ParseRoleGranted(log types.Log) (*TxBridgeRoleGranted, error) {
	event := new(TxBridgeRoleGranted)
	if err := _TxBridge.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TxBridgeRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the TxBridge contract.
type TxBridgeRoleRevokedIterator struct {
	Event *TxBridgeRoleRevoked // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TxBridgeRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TxBridgeRoleRevoked)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TxBridgeRoleRevoked)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TxBridgeRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TxBridgeRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TxBridgeRoleRevoked represents a RoleRevoked event raised by the TxBridge contract.
type TxBridgeRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_TxBridge *TxBridgeFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*TxBridgeRoleRevokedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _TxBridge.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &TxBridgeRoleRevokedIterator{contract: _TxBridge.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_TxBridge *TxBridgeFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *TxBridgeRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _TxBridge.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TxBridgeRoleRevoked)
				if err := _TxBridge.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleRevoked is a log parse operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_TxBridge *TxBridgeFilterer) ParseRoleRevoked(log types.Log) (*TxBridgeRoleRevoked, error) {
	event := new(TxBridgeRoleRevoked)
	if err := _TxBridge.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TxBridgeTokenUpdatedIterator is returned from FilterTokenUpdated and is used to iterate over the raw logs and unpacked data for TokenUpdated events raised by the TxBridge contract.
type TxBridgeTokenUpdatedIterator struct {
	Event *TxBridgeTokenUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TxBridgeTokenUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TxBridgeTokenUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TxBridgeTokenUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TxBridgeTokenUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TxBridgeTokenUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TxBridgeTokenUpdated represents a TokenUpdated event raised by the TxBridge contract.
type TxBridgeTokenUpdated struct {
	OldToken common.Address
	NewToken common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterTokenUpdated is a free log retrieval operation binding the contract event 0x0b1186973f810894b87ab0bfbee422fddcaad21b46dc705a561451bbb6bac117.
//
// Solidity: event TokenUpdated(address oldToken, address newToken)
func (_TxBridge *TxBridgeFilterer) FilterTokenUpdated(opts *bind.FilterOpts) (*TxBridgeTokenUpdatedIterator, error) {

	logs, sub, err := _TxBridge.contract.FilterLogs(opts, "TokenUpdated")
	if err != nil {
		return nil, err
	}
	return &TxBridgeTokenUpdatedIterator{contract: _TxBridge.contract, event: "TokenUpdated", logs: logs, sub: sub}, nil
}

// WatchTokenUpdated is a free log subscription operation binding the contract event 0x0b1186973f810894b87ab0bfbee422fddcaad21b46dc705a561451bbb6bac117.
//
// Solidity: event TokenUpdated(address oldToken, address newToken)
func (_TxBridge *TxBridgeFilterer) WatchTokenUpdated(opts *bind.WatchOpts, sink chan<- *TxBridgeTokenUpdated) (event.Subscription, error) {

	logs, sub, err := _TxBridge.contract.WatchLogs(opts, "TokenUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TxBridgeTokenUpdated)
				if err := _TxBridge.contract.UnpackLog(event, "TokenUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTokenUpdated is a log parse operation binding the contract event 0x0b1186973f810894b87ab0bfbee422fddcaad21b46dc705a561451bbb6bac117.
//
// Solidity: event TokenUpdated(address oldToken, address newToken)
func (_TxBridge *TxBridgeFilterer) ParseTokenUpdated(log types.Log) (*TxBridgeTokenUpdated, error) {
	event := new(TxBridgeTokenUpdated)
	if err := _TxBridge.contract.UnpackLog(event, "TokenUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TxBridgeUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the TxBridge contract.
type TxBridgeUnpausedIterator struct {
	Event *TxBridgeUnpaused // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TxBridgeUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TxBridgeUnpaused)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TxBridgeUnpaused)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TxBridgeUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TxBridgeUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TxBridgeUnpaused represents a Unpaused event raised by the TxBridge contract.
type TxBridgeUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_TxBridge *TxBridgeFilterer) FilterUnpaused(opts *bind.FilterOpts) (*TxBridgeUnpausedIterator, error) {

	logs, sub, err := _TxBridge.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &TxBridgeUnpausedIterator{contract: _TxBridge.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_TxBridge *TxBridgeFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *TxBridgeUnpaused) (event.Subscription, error) {

	logs, sub, err := _TxBridge.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TxBridgeUnpaused)
				if err := _TxBridge.contract.UnpackLog(event, "Unpaused", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUnpaused is a log parse operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_TxBridge *TxBridgeFilterer) ParseUnpaused(log types.Log) (*TxBridgeUnpaused, error) {
	event := new(TxBridgeUnpaused)
	if err := _TxBridge.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
