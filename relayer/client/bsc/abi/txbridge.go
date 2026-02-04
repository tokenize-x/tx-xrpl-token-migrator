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

// TXBridgeMetaData contains all meta data concerning the TXBridge contract.
var TXBridgeMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AccessControlBadConfirmation\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"neededRole\",\"type\":\"bytes32\"}],\"name\":\"AccessControlUnauthorizedAccount\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maximum\",\"type\":\"uint256\"}],\"name\":\"AmountAboveMaximum\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"minimum\",\"type\":\"uint256\"}],\"name\":\"AmountBelowMinimum\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyString\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EnforcedPause\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ExpectedPause\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidAmount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidInitialization\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidTXAddress\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotInitializing\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ReentrancyGuardReentrantCall\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroAddress\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"version\",\"type\":\"uint64\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"previousAdminRole\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"newAdminRole\",\"type\":\"bytes32\"}],\"name\":\"RoleAdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleGranted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"txAddress\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"SentToTXChain\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"oldToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newToken\",\"type\":\"address\"}],\"name\":\"TokenUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEFAULT_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OPERATOR_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"addressPrefix\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleAdmin\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"grantRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRole\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_admin\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_minAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_maxAmount\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"_addressPrefix\",\"type\":\"string\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"txAddress\",\"type\":\"string\"}],\"name\":\"isValidTXAddress\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"callerConfirmation\",\"type\":\"address\"}],\"name\":\"renounceRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"txAddress\",\"type\":\"string\"}],\"name\":\"sendToTXChain\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_addressPrefix\",\"type\":\"string\"}],\"name\":\"setAddressPrefix\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_maxAmount\",\"type\":\"uint256\"}],\"name\":\"setMaxAmount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_minAmount\",\"type\":\"uint256\"}],\"name\":\"setMinAmount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"}],\"name\":\"setToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token\",\"outputs\":[{\"internalType\":\"contractITXToken\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// TXBridgeABI is the input ABI used to generate the binding from.
// Deprecated: Use TXBridgeMetaData.ABI instead.
var TXBridgeABI = TXBridgeMetaData.ABI

// TXBridge is an auto generated Go binding around an Ethereum contract.
type TXBridge struct {
	TXBridgeCaller     // Read-only binding to the contract
	TXBridgeTransactor // Write-only binding to the contract
	TXBridgeFilterer   // Log filterer for contract events
}

// TXBridgeCaller is an auto generated read-only Go binding around an Ethereum contract.
type TXBridgeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TXBridgeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TXBridgeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TXBridgeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TXBridgeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TXBridgeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TXBridgeSession struct {
	Contract     *TXBridge         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TXBridgeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TXBridgeCallerSession struct {
	Contract *TXBridgeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// TXBridgeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TXBridgeTransactorSession struct {
	Contract     *TXBridgeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// TXBridgeRaw is an auto generated low-level Go binding around an Ethereum contract.
type TXBridgeRaw struct {
	Contract *TXBridge // Generic contract binding to access the raw methods on
}

// TXBridgeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TXBridgeCallerRaw struct {
	Contract *TXBridgeCaller // Generic read-only contract binding to access the raw methods on
}

// TXBridgeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TXBridgeTransactorRaw struct {
	Contract *TXBridgeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTXBridge creates a new instance of TXBridge, bound to a specific deployed contract.
func NewTXBridge(address common.Address, backend bind.ContractBackend) (*TXBridge, error) {
	contract, err := bindTXBridge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TXBridge{TXBridgeCaller: TXBridgeCaller{contract: contract}, TXBridgeTransactor: TXBridgeTransactor{contract: contract}, TXBridgeFilterer: TXBridgeFilterer{contract: contract}}, nil
}

// NewTXBridgeCaller creates a new read-only instance of TXBridge, bound to a specific deployed contract.
func NewTXBridgeCaller(address common.Address, caller bind.ContractCaller) (*TXBridgeCaller, error) {
	contract, err := bindTXBridge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TXBridgeCaller{contract: contract}, nil
}

// NewTXBridgeTransactor creates a new write-only instance of TXBridge, bound to a specific deployed contract.
func NewTXBridgeTransactor(address common.Address, transactor bind.ContractTransactor) (*TXBridgeTransactor, error) {
	contract, err := bindTXBridge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TXBridgeTransactor{contract: contract}, nil
}

// NewTXBridgeFilterer creates a new log filterer instance of TXBridge, bound to a specific deployed contract.
func NewTXBridgeFilterer(address common.Address, filterer bind.ContractFilterer) (*TXBridgeFilterer, error) {
	contract, err := bindTXBridge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TXBridgeFilterer{contract: contract}, nil
}

// bindTXBridge binds a generic wrapper to an already deployed contract.
func bindTXBridge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TXBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TXBridge *TXBridgeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TXBridge.Contract.TXBridgeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TXBridge *TXBridgeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TXBridge.Contract.TXBridgeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TXBridge *TXBridgeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TXBridge.Contract.TXBridgeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TXBridge *TXBridgeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TXBridge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TXBridge *TXBridgeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TXBridge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TXBridge *TXBridgeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TXBridge.Contract.contract.Transact(opts, method, params...)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_TXBridge *TXBridgeCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _TXBridge.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_TXBridge *TXBridgeSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _TXBridge.Contract.DEFAULTADMINROLE(&_TXBridge.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_TXBridge *TXBridgeCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _TXBridge.Contract.DEFAULTADMINROLE(&_TXBridge.CallOpts)
}

// OPERATORROLE is a free data retrieval call binding the contract method 0xf5b541a6.
//
// Solidity: function OPERATOR_ROLE() view returns(bytes32)
func (_TXBridge *TXBridgeCaller) OPERATORROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _TXBridge.contract.Call(opts, &out, "OPERATOR_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// OPERATORROLE is a free data retrieval call binding the contract method 0xf5b541a6.
//
// Solidity: function OPERATOR_ROLE() view returns(bytes32)
func (_TXBridge *TXBridgeSession) OPERATORROLE() ([32]byte, error) {
	return _TXBridge.Contract.OPERATORROLE(&_TXBridge.CallOpts)
}

// OPERATORROLE is a free data retrieval call binding the contract method 0xf5b541a6.
//
// Solidity: function OPERATOR_ROLE() view returns(bytes32)
func (_TXBridge *TXBridgeCallerSession) OPERATORROLE() ([32]byte, error) {
	return _TXBridge.Contract.OPERATORROLE(&_TXBridge.CallOpts)
}

// AddressPrefix is a free data retrieval call binding the contract method 0xf0e0de8f.
//
// Solidity: function addressPrefix() view returns(string)
func (_TXBridge *TXBridgeCaller) AddressPrefix(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _TXBridge.contract.Call(opts, &out, "addressPrefix")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// AddressPrefix is a free data retrieval call binding the contract method 0xf0e0de8f.
//
// Solidity: function addressPrefix() view returns(string)
func (_TXBridge *TXBridgeSession) AddressPrefix() (string, error) {
	return _TXBridge.Contract.AddressPrefix(&_TXBridge.CallOpts)
}

// AddressPrefix is a free data retrieval call binding the contract method 0xf0e0de8f.
//
// Solidity: function addressPrefix() view returns(string)
func (_TXBridge *TXBridgeCallerSession) AddressPrefix() (string, error) {
	return _TXBridge.Contract.AddressPrefix(&_TXBridge.CallOpts)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_TXBridge *TXBridgeCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _TXBridge.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_TXBridge *TXBridgeSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _TXBridge.Contract.GetRoleAdmin(&_TXBridge.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_TXBridge *TXBridgeCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _TXBridge.Contract.GetRoleAdmin(&_TXBridge.CallOpts, role)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_TXBridge *TXBridgeCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _TXBridge.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_TXBridge *TXBridgeSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _TXBridge.Contract.HasRole(&_TXBridge.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_TXBridge *TXBridgeCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _TXBridge.Contract.HasRole(&_TXBridge.CallOpts, role, account)
}

// IsValidTXAddress is a free data retrieval call binding the contract method 0x061730e7.
//
// Solidity: function isValidTXAddress(string txAddress) view returns(bool)
func (_TXBridge *TXBridgeCaller) IsValidTXAddress(opts *bind.CallOpts, txAddress string) (bool, error) {
	var out []interface{}
	err := _TXBridge.contract.Call(opts, &out, "isValidTXAddress", txAddress)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsValidTXAddress is a free data retrieval call binding the contract method 0x061730e7.
//
// Solidity: function isValidTXAddress(string txAddress) view returns(bool)
func (_TXBridge *TXBridgeSession) IsValidTXAddress(txAddress string) (bool, error) {
	return _TXBridge.Contract.IsValidTXAddress(&_TXBridge.CallOpts, txAddress)
}

// IsValidTXAddress is a free data retrieval call binding the contract method 0x061730e7.
//
// Solidity: function isValidTXAddress(string txAddress) view returns(bool)
func (_TXBridge *TXBridgeCallerSession) IsValidTXAddress(txAddress string) (bool, error) {
	return _TXBridge.Contract.IsValidTXAddress(&_TXBridge.CallOpts, txAddress)
}

// MaxAmount is a free data retrieval call binding the contract method 0x5f48f393.
//
// Solidity: function maxAmount() view returns(uint256)
func (_TXBridge *TXBridgeCaller) MaxAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TXBridge.contract.Call(opts, &out, "maxAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxAmount is a free data retrieval call binding the contract method 0x5f48f393.
//
// Solidity: function maxAmount() view returns(uint256)
func (_TXBridge *TXBridgeSession) MaxAmount() (*big.Int, error) {
	return _TXBridge.Contract.MaxAmount(&_TXBridge.CallOpts)
}

// MaxAmount is a free data retrieval call binding the contract method 0x5f48f393.
//
// Solidity: function maxAmount() view returns(uint256)
func (_TXBridge *TXBridgeCallerSession) MaxAmount() (*big.Int, error) {
	return _TXBridge.Contract.MaxAmount(&_TXBridge.CallOpts)
}

// MinAmount is a free data retrieval call binding the contract method 0x9b2cb5d8.
//
// Solidity: function minAmount() view returns(uint256)
func (_TXBridge *TXBridgeCaller) MinAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TXBridge.contract.Call(opts, &out, "minAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinAmount is a free data retrieval call binding the contract method 0x9b2cb5d8.
//
// Solidity: function minAmount() view returns(uint256)
func (_TXBridge *TXBridgeSession) MinAmount() (*big.Int, error) {
	return _TXBridge.Contract.MinAmount(&_TXBridge.CallOpts)
}

// MinAmount is a free data retrieval call binding the contract method 0x9b2cb5d8.
//
// Solidity: function minAmount() view returns(uint256)
func (_TXBridge *TXBridgeCallerSession) MinAmount() (*big.Int, error) {
	return _TXBridge.Contract.MinAmount(&_TXBridge.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_TXBridge *TXBridgeCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _TXBridge.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_TXBridge *TXBridgeSession) Paused() (bool, error) {
	return _TXBridge.Contract.Paused(&_TXBridge.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_TXBridge *TXBridgeCallerSession) Paused() (bool, error) {
	return _TXBridge.Contract.Paused(&_TXBridge.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_TXBridge *TXBridgeCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _TXBridge.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_TXBridge *TXBridgeSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _TXBridge.Contract.SupportsInterface(&_TXBridge.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_TXBridge *TXBridgeCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _TXBridge.Contract.SupportsInterface(&_TXBridge.CallOpts, interfaceId)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_TXBridge *TXBridgeCaller) Token(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _TXBridge.contract.Call(opts, &out, "token")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_TXBridge *TXBridgeSession) Token() (common.Address, error) {
	return _TXBridge.Contract.Token(&_TXBridge.CallOpts)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_TXBridge *TXBridgeCallerSession) Token() (common.Address, error) {
	return _TXBridge.Contract.Token(&_TXBridge.CallOpts)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_TXBridge *TXBridgeTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _TXBridge.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_TXBridge *TXBridgeSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _TXBridge.Contract.GrantRole(&_TXBridge.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_TXBridge *TXBridgeTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _TXBridge.Contract.GrantRole(&_TXBridge.TransactOpts, role, account)
}

// Initialize is a paid mutator transaction binding the contract method 0x8e737af2.
//
// Solidity: function initialize(address _token, address _admin, uint256 _minAmount, uint256 _maxAmount, string _addressPrefix) returns()
func (_TXBridge *TXBridgeTransactor) Initialize(opts *bind.TransactOpts, _token common.Address, _admin common.Address, _minAmount *big.Int, _maxAmount *big.Int, _addressPrefix string) (*types.Transaction, error) {
	return _TXBridge.contract.Transact(opts, "initialize", _token, _admin, _minAmount, _maxAmount, _addressPrefix)
}

// Initialize is a paid mutator transaction binding the contract method 0x8e737af2.
//
// Solidity: function initialize(address _token, address _admin, uint256 _minAmount, uint256 _maxAmount, string _addressPrefix) returns()
func (_TXBridge *TXBridgeSession) Initialize(_token common.Address, _admin common.Address, _minAmount *big.Int, _maxAmount *big.Int, _addressPrefix string) (*types.Transaction, error) {
	return _TXBridge.Contract.Initialize(&_TXBridge.TransactOpts, _token, _admin, _minAmount, _maxAmount, _addressPrefix)
}

// Initialize is a paid mutator transaction binding the contract method 0x8e737af2.
//
// Solidity: function initialize(address _token, address _admin, uint256 _minAmount, uint256 _maxAmount, string _addressPrefix) returns()
func (_TXBridge *TXBridgeTransactorSession) Initialize(_token common.Address, _admin common.Address, _minAmount *big.Int, _maxAmount *big.Int, _addressPrefix string) (*types.Transaction, error) {
	return _TXBridge.Contract.Initialize(&_TXBridge.TransactOpts, _token, _admin, _minAmount, _maxAmount, _addressPrefix)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_TXBridge *TXBridgeTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TXBridge.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_TXBridge *TXBridgeSession) Pause() (*types.Transaction, error) {
	return _TXBridge.Contract.Pause(&_TXBridge.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_TXBridge *TXBridgeTransactorSession) Pause() (*types.Transaction, error) {
	return _TXBridge.Contract.Pause(&_TXBridge.TransactOpts)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address callerConfirmation) returns()
func (_TXBridge *TXBridgeTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, callerConfirmation common.Address) (*types.Transaction, error) {
	return _TXBridge.contract.Transact(opts, "renounceRole", role, callerConfirmation)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address callerConfirmation) returns()
func (_TXBridge *TXBridgeSession) RenounceRole(role [32]byte, callerConfirmation common.Address) (*types.Transaction, error) {
	return _TXBridge.Contract.RenounceRole(&_TXBridge.TransactOpts, role, callerConfirmation)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address callerConfirmation) returns()
func (_TXBridge *TXBridgeTransactorSession) RenounceRole(role [32]byte, callerConfirmation common.Address) (*types.Transaction, error) {
	return _TXBridge.Contract.RenounceRole(&_TXBridge.TransactOpts, role, callerConfirmation)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_TXBridge *TXBridgeTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _TXBridge.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_TXBridge *TXBridgeSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _TXBridge.Contract.RevokeRole(&_TXBridge.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_TXBridge *TXBridgeTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _TXBridge.Contract.RevokeRole(&_TXBridge.TransactOpts, role, account)
}

// SendToTXChain is a paid mutator transaction binding the contract method 0x2389b2bc.
//
// Solidity: function sendToTXChain(uint256 amount, string txAddress) returns()
func (_TXBridge *TXBridgeTransactor) SendToTXChain(opts *bind.TransactOpts, amount *big.Int, txAddress string) (*types.Transaction, error) {
	return _TXBridge.contract.Transact(opts, "sendToTXChain", amount, txAddress)
}

// SendToTXChain is a paid mutator transaction binding the contract method 0x2389b2bc.
//
// Solidity: function sendToTXChain(uint256 amount, string txAddress) returns()
func (_TXBridge *TXBridgeSession) SendToTXChain(amount *big.Int, txAddress string) (*types.Transaction, error) {
	return _TXBridge.Contract.SendToTXChain(&_TXBridge.TransactOpts, amount, txAddress)
}

// SendToTXChain is a paid mutator transaction binding the contract method 0x2389b2bc.
//
// Solidity: function sendToTXChain(uint256 amount, string txAddress) returns()
func (_TXBridge *TXBridgeTransactorSession) SendToTXChain(amount *big.Int, txAddress string) (*types.Transaction, error) {
	return _TXBridge.Contract.SendToTXChain(&_TXBridge.TransactOpts, amount, txAddress)
}

// SetAddressPrefix is a paid mutator transaction binding the contract method 0xb8e4e61d.
//
// Solidity: function setAddressPrefix(string _addressPrefix) returns()
func (_TXBridge *TXBridgeTransactor) SetAddressPrefix(opts *bind.TransactOpts, _addressPrefix string) (*types.Transaction, error) {
	return _TXBridge.contract.Transact(opts, "setAddressPrefix", _addressPrefix)
}

// SetAddressPrefix is a paid mutator transaction binding the contract method 0xb8e4e61d.
//
// Solidity: function setAddressPrefix(string _addressPrefix) returns()
func (_TXBridge *TXBridgeSession) SetAddressPrefix(_addressPrefix string) (*types.Transaction, error) {
	return _TXBridge.Contract.SetAddressPrefix(&_TXBridge.TransactOpts, _addressPrefix)
}

// SetAddressPrefix is a paid mutator transaction binding the contract method 0xb8e4e61d.
//
// Solidity: function setAddressPrefix(string _addressPrefix) returns()
func (_TXBridge *TXBridgeTransactorSession) SetAddressPrefix(_addressPrefix string) (*types.Transaction, error) {
	return _TXBridge.Contract.SetAddressPrefix(&_TXBridge.TransactOpts, _addressPrefix)
}

// SetMaxAmount is a paid mutator transaction binding the contract method 0x4fe47f70.
//
// Solidity: function setMaxAmount(uint256 _maxAmount) returns()
func (_TXBridge *TXBridgeTransactor) SetMaxAmount(opts *bind.TransactOpts, _maxAmount *big.Int) (*types.Transaction, error) {
	return _TXBridge.contract.Transact(opts, "setMaxAmount", _maxAmount)
}

// SetMaxAmount is a paid mutator transaction binding the contract method 0x4fe47f70.
//
// Solidity: function setMaxAmount(uint256 _maxAmount) returns()
func (_TXBridge *TXBridgeSession) SetMaxAmount(_maxAmount *big.Int) (*types.Transaction, error) {
	return _TXBridge.Contract.SetMaxAmount(&_TXBridge.TransactOpts, _maxAmount)
}

// SetMaxAmount is a paid mutator transaction binding the contract method 0x4fe47f70.
//
// Solidity: function setMaxAmount(uint256 _maxAmount) returns()
func (_TXBridge *TXBridgeTransactorSession) SetMaxAmount(_maxAmount *big.Int) (*types.Transaction, error) {
	return _TXBridge.Contract.SetMaxAmount(&_TXBridge.TransactOpts, _maxAmount)
}

// SetMinAmount is a paid mutator transaction binding the contract method 0x897b0637.
//
// Solidity: function setMinAmount(uint256 _minAmount) returns()
func (_TXBridge *TXBridgeTransactor) SetMinAmount(opts *bind.TransactOpts, _minAmount *big.Int) (*types.Transaction, error) {
	return _TXBridge.contract.Transact(opts, "setMinAmount", _minAmount)
}

// SetMinAmount is a paid mutator transaction binding the contract method 0x897b0637.
//
// Solidity: function setMinAmount(uint256 _minAmount) returns()
func (_TXBridge *TXBridgeSession) SetMinAmount(_minAmount *big.Int) (*types.Transaction, error) {
	return _TXBridge.Contract.SetMinAmount(&_TXBridge.TransactOpts, _minAmount)
}

// SetMinAmount is a paid mutator transaction binding the contract method 0x897b0637.
//
// Solidity: function setMinAmount(uint256 _minAmount) returns()
func (_TXBridge *TXBridgeTransactorSession) SetMinAmount(_minAmount *big.Int) (*types.Transaction, error) {
	return _TXBridge.Contract.SetMinAmount(&_TXBridge.TransactOpts, _minAmount)
}

// SetToken is a paid mutator transaction binding the contract method 0x144fa6d7.
//
// Solidity: function setToken(address _token) returns()
func (_TXBridge *TXBridgeTransactor) SetToken(opts *bind.TransactOpts, _token common.Address) (*types.Transaction, error) {
	return _TXBridge.contract.Transact(opts, "setToken", _token)
}

// SetToken is a paid mutator transaction binding the contract method 0x144fa6d7.
//
// Solidity: function setToken(address _token) returns()
func (_TXBridge *TXBridgeSession) SetToken(_token common.Address) (*types.Transaction, error) {
	return _TXBridge.Contract.SetToken(&_TXBridge.TransactOpts, _token)
}

// SetToken is a paid mutator transaction binding the contract method 0x144fa6d7.
//
// Solidity: function setToken(address _token) returns()
func (_TXBridge *TXBridgeTransactorSession) SetToken(_token common.Address) (*types.Transaction, error) {
	return _TXBridge.Contract.SetToken(&_TXBridge.TransactOpts, _token)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_TXBridge *TXBridgeTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TXBridge.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_TXBridge *TXBridgeSession) Unpause() (*types.Transaction, error) {
	return _TXBridge.Contract.Unpause(&_TXBridge.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_TXBridge *TXBridgeTransactorSession) Unpause() (*types.Transaction, error) {
	return _TXBridge.Contract.Unpause(&_TXBridge.TransactOpts)
}

// TXBridgeInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the TXBridge contract.
type TXBridgeInitializedIterator struct {
	Event *TXBridgeInitialized // Event containing the contract specifics and raw log

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
func (it *TXBridgeInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TXBridgeInitialized)
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
		it.Event = new(TXBridgeInitialized)
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
func (it *TXBridgeInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TXBridgeInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TXBridgeInitialized represents a Initialized event raised by the TXBridge contract.
type TXBridgeInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_TXBridge *TXBridgeFilterer) FilterInitialized(opts *bind.FilterOpts) (*TXBridgeInitializedIterator, error) {

	logs, sub, err := _TXBridge.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &TXBridgeInitializedIterator{contract: _TXBridge.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_TXBridge *TXBridgeFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *TXBridgeInitialized) (event.Subscription, error) {

	logs, sub, err := _TXBridge.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TXBridgeInitialized)
				if err := _TXBridge.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_TXBridge *TXBridgeFilterer) ParseInitialized(log types.Log) (*TXBridgeInitialized, error) {
	event := new(TXBridgeInitialized)
	if err := _TXBridge.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TXBridgePausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the TXBridge contract.
type TXBridgePausedIterator struct {
	Event *TXBridgePaused // Event containing the contract specifics and raw log

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
func (it *TXBridgePausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TXBridgePaused)
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
		it.Event = new(TXBridgePaused)
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
func (it *TXBridgePausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TXBridgePausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TXBridgePaused represents a Paused event raised by the TXBridge contract.
type TXBridgePaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_TXBridge *TXBridgeFilterer) FilterPaused(opts *bind.FilterOpts) (*TXBridgePausedIterator, error) {

	logs, sub, err := _TXBridge.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &TXBridgePausedIterator{contract: _TXBridge.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_TXBridge *TXBridgeFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *TXBridgePaused) (event.Subscription, error) {

	logs, sub, err := _TXBridge.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TXBridgePaused)
				if err := _TXBridge.contract.UnpackLog(event, "Paused", log); err != nil {
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
func (_TXBridge *TXBridgeFilterer) ParsePaused(log types.Log) (*TXBridgePaused, error) {
	event := new(TXBridgePaused)
	if err := _TXBridge.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TXBridgeRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the TXBridge contract.
type TXBridgeRoleAdminChangedIterator struct {
	Event *TXBridgeRoleAdminChanged // Event containing the contract specifics and raw log

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
func (it *TXBridgeRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TXBridgeRoleAdminChanged)
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
		it.Event = new(TXBridgeRoleAdminChanged)
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
func (it *TXBridgeRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TXBridgeRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TXBridgeRoleAdminChanged represents a RoleAdminChanged event raised by the TXBridge contract.
type TXBridgeRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_TXBridge *TXBridgeFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*TXBridgeRoleAdminChangedIterator, error) {

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

	logs, sub, err := _TXBridge.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &TXBridgeRoleAdminChangedIterator{contract: _TXBridge.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_TXBridge *TXBridgeFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *TXBridgeRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

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

	logs, sub, err := _TXBridge.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TXBridgeRoleAdminChanged)
				if err := _TXBridge.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
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
func (_TXBridge *TXBridgeFilterer) ParseRoleAdminChanged(log types.Log) (*TXBridgeRoleAdminChanged, error) {
	event := new(TXBridgeRoleAdminChanged)
	if err := _TXBridge.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TXBridgeRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the TXBridge contract.
type TXBridgeRoleGrantedIterator struct {
	Event *TXBridgeRoleGranted // Event containing the contract specifics and raw log

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
func (it *TXBridgeRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TXBridgeRoleGranted)
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
		it.Event = new(TXBridgeRoleGranted)
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
func (it *TXBridgeRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TXBridgeRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TXBridgeRoleGranted represents a RoleGranted event raised by the TXBridge contract.
type TXBridgeRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_TXBridge *TXBridgeFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*TXBridgeRoleGrantedIterator, error) {

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

	logs, sub, err := _TXBridge.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &TXBridgeRoleGrantedIterator{contract: _TXBridge.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_TXBridge *TXBridgeFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *TXBridgeRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _TXBridge.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TXBridgeRoleGranted)
				if err := _TXBridge.contract.UnpackLog(event, "RoleGranted", log); err != nil {
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
func (_TXBridge *TXBridgeFilterer) ParseRoleGranted(log types.Log) (*TXBridgeRoleGranted, error) {
	event := new(TXBridgeRoleGranted)
	if err := _TXBridge.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TXBridgeRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the TXBridge contract.
type TXBridgeRoleRevokedIterator struct {
	Event *TXBridgeRoleRevoked // Event containing the contract specifics and raw log

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
func (it *TXBridgeRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TXBridgeRoleRevoked)
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
		it.Event = new(TXBridgeRoleRevoked)
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
func (it *TXBridgeRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TXBridgeRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TXBridgeRoleRevoked represents a RoleRevoked event raised by the TXBridge contract.
type TXBridgeRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_TXBridge *TXBridgeFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*TXBridgeRoleRevokedIterator, error) {

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

	logs, sub, err := _TXBridge.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &TXBridgeRoleRevokedIterator{contract: _TXBridge.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_TXBridge *TXBridgeFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *TXBridgeRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _TXBridge.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TXBridgeRoleRevoked)
				if err := _TXBridge.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
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
func (_TXBridge *TXBridgeFilterer) ParseRoleRevoked(log types.Log) (*TXBridgeRoleRevoked, error) {
	event := new(TXBridgeRoleRevoked)
	if err := _TXBridge.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TXBridgeSentToTXChainIterator is returned from FilterSentToTXChain and is used to iterate over the raw logs and unpacked data for SentToTXChain events raised by the TXBridge contract.
type TXBridgeSentToTXChainIterator struct {
	Event *TXBridgeSentToTXChain // Event containing the contract specifics and raw log

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
func (it *TXBridgeSentToTXChainIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TXBridgeSentToTXChain)
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
		it.Event = new(TXBridgeSentToTXChain)
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
func (it *TXBridgeSentToTXChainIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TXBridgeSentToTXChainIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TXBridgeSentToTXChain represents a SentToTXChain event raised by the TXBridge contract.
type TXBridgeSentToTXChain struct {
	From      common.Address
	TxAddress string
	Amount    *big.Int
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterSentToTXChain is a free log retrieval operation binding the contract event 0xa0d66d4ed6a10cb5b9c30a9a41ba884bee23e383f96c44462f6e274f35eb5600.
//
// Solidity: event SentToTXChain(address indexed from, string txAddress, uint256 amount, uint256 timestamp)
func (_TXBridge *TXBridgeFilterer) FilterSentToTXChain(opts *bind.FilterOpts, from []common.Address) (*TXBridgeSentToTXChainIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _TXBridge.contract.FilterLogs(opts, "SentToTXChain", fromRule)
	if err != nil {
		return nil, err
	}
	return &TXBridgeSentToTXChainIterator{contract: _TXBridge.contract, event: "SentToTXChain", logs: logs, sub: sub}, nil
}

// WatchSentToTXChain is a free log subscription operation binding the contract event 0xa0d66d4ed6a10cb5b9c30a9a41ba884bee23e383f96c44462f6e274f35eb5600.
//
// Solidity: event SentToTXChain(address indexed from, string txAddress, uint256 amount, uint256 timestamp)
func (_TXBridge *TXBridgeFilterer) WatchSentToTXChain(opts *bind.WatchOpts, sink chan<- *TXBridgeSentToTXChain, from []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _TXBridge.contract.WatchLogs(opts, "SentToTXChain", fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TXBridgeSentToTXChain)
				if err := _TXBridge.contract.UnpackLog(event, "SentToTXChain", log); err != nil {
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

// ParseSentToTXChain is a log parse operation binding the contract event 0xa0d66d4ed6a10cb5b9c30a9a41ba884bee23e383f96c44462f6e274f35eb5600.
//
// Solidity: event SentToTXChain(address indexed from, string txAddress, uint256 amount, uint256 timestamp)
func (_TXBridge *TXBridgeFilterer) ParseSentToTXChain(log types.Log) (*TXBridgeSentToTXChain, error) {
	event := new(TXBridgeSentToTXChain)
	if err := _TXBridge.contract.UnpackLog(event, "SentToTXChain", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TXBridgeTokenUpdatedIterator is returned from FilterTokenUpdated and is used to iterate over the raw logs and unpacked data for TokenUpdated events raised by the TXBridge contract.
type TXBridgeTokenUpdatedIterator struct {
	Event *TXBridgeTokenUpdated // Event containing the contract specifics and raw log

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
func (it *TXBridgeTokenUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TXBridgeTokenUpdated)
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
		it.Event = new(TXBridgeTokenUpdated)
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
func (it *TXBridgeTokenUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TXBridgeTokenUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TXBridgeTokenUpdated represents a TokenUpdated event raised by the TXBridge contract.
type TXBridgeTokenUpdated struct {
	OldToken common.Address
	NewToken common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterTokenUpdated is a free log retrieval operation binding the contract event 0x0b1186973f810894b87ab0bfbee422fddcaad21b46dc705a561451bbb6bac117.
//
// Solidity: event TokenUpdated(address oldToken, address newToken)
func (_TXBridge *TXBridgeFilterer) FilterTokenUpdated(opts *bind.FilterOpts) (*TXBridgeTokenUpdatedIterator, error) {

	logs, sub, err := _TXBridge.contract.FilterLogs(opts, "TokenUpdated")
	if err != nil {
		return nil, err
	}
	return &TXBridgeTokenUpdatedIterator{contract: _TXBridge.contract, event: "TokenUpdated", logs: logs, sub: sub}, nil
}

// WatchTokenUpdated is a free log subscription operation binding the contract event 0x0b1186973f810894b87ab0bfbee422fddcaad21b46dc705a561451bbb6bac117.
//
// Solidity: event TokenUpdated(address oldToken, address newToken)
func (_TXBridge *TXBridgeFilterer) WatchTokenUpdated(opts *bind.WatchOpts, sink chan<- *TXBridgeTokenUpdated) (event.Subscription, error) {

	logs, sub, err := _TXBridge.contract.WatchLogs(opts, "TokenUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TXBridgeTokenUpdated)
				if err := _TXBridge.contract.UnpackLog(event, "TokenUpdated", log); err != nil {
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
func (_TXBridge *TXBridgeFilterer) ParseTokenUpdated(log types.Log) (*TXBridgeTokenUpdated, error) {
	event := new(TXBridgeTokenUpdated)
	if err := _TXBridge.contract.UnpackLog(event, "TokenUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TXBridgeUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the TXBridge contract.
type TXBridgeUnpausedIterator struct {
	Event *TXBridgeUnpaused // Event containing the contract specifics and raw log

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
func (it *TXBridgeUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TXBridgeUnpaused)
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
		it.Event = new(TXBridgeUnpaused)
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
func (it *TXBridgeUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TXBridgeUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TXBridgeUnpaused represents a Unpaused event raised by the TXBridge contract.
type TXBridgeUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_TXBridge *TXBridgeFilterer) FilterUnpaused(opts *bind.FilterOpts) (*TXBridgeUnpausedIterator, error) {

	logs, sub, err := _TXBridge.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &TXBridgeUnpausedIterator{contract: _TXBridge.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_TXBridge *TXBridgeFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *TXBridgeUnpaused) (event.Subscription, error) {

	logs, sub, err := _TXBridge.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TXBridgeUnpaused)
				if err := _TXBridge.contract.UnpackLog(event, "Unpaused", log); err != nil {
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
func (_TXBridge *TXBridgeFilterer) ParseUnpaused(log types.Log) (*TXBridgeUnpaused, error) {
	event := new(TXBridgeUnpaused)
	if err := _TXBridge.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
