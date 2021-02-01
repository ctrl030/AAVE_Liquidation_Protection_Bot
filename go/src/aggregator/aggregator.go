// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package aggregator

import (
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
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// AggregatorABI is the input ABI used to generate the binding from.
const AggregatorABI = "[{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"description\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint80\",\"name\":\"_roundId\",\"type\":\"uint80\"}],\"name\":\"getRoundData\",\"outputs\":[{\"internalType\":\"uint80\",\"name\":\"roundId\",\"type\":\"uint80\"},{\"internalType\":\"int256\",\"name\":\"answer\",\"type\":\"int256\"},{\"internalType\":\"uint256\",\"name\":\"startedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint80\",\"name\":\"answeredInRound\",\"type\":\"uint80\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestRoundData\",\"outputs\":[{\"internalType\":\"uint80\",\"name\":\"roundId\",\"type\":\"uint80\"},{\"internalType\":\"int256\",\"name\":\"answer\",\"type\":\"int256\"},{\"internalType\":\"uint256\",\"name\":\"startedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint80\",\"name\":\"answeredInRound\",\"type\":\"uint80\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

// Aggregator is an auto generated Go binding around an Ethereum contract.
type Aggregator struct {
	AggregatorCaller     // Read-only binding to the contract
	AggregatorTransactor // Write-only binding to the contract
	AggregatorFilterer   // Log filterer for contract events
}

// AggregatorCaller is an auto generated read-only Go binding around an Ethereum contract.
type AggregatorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AggregatorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AggregatorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AggregatorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AggregatorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AggregatorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AggregatorSession struct {
	Contract     *Aggregator       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AggregatorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AggregatorCallerSession struct {
	Contract *AggregatorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// AggregatorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AggregatorTransactorSession struct {
	Contract     *AggregatorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// AggregatorRaw is an auto generated low-level Go binding around an Ethereum contract.
type AggregatorRaw struct {
	Contract *Aggregator // Generic contract binding to access the raw methods on
}

// AggregatorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AggregatorCallerRaw struct {
	Contract *AggregatorCaller // Generic read-only contract binding to access the raw methods on
}

// AggregatorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AggregatorTransactorRaw struct {
	Contract *AggregatorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAggregator creates a new instance of Aggregator, bound to a specific deployed contract.
func NewAggregator(address common.Address, backend bind.ContractBackend) (*Aggregator, error) {
	contract, err := bindAggregator(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Aggregator{AggregatorCaller: AggregatorCaller{contract: contract}, AggregatorTransactor: AggregatorTransactor{contract: contract}, AggregatorFilterer: AggregatorFilterer{contract: contract}}, nil
}

// NewAggregatorCaller creates a new read-only instance of Aggregator, bound to a specific deployed contract.
func NewAggregatorCaller(address common.Address, caller bind.ContractCaller) (*AggregatorCaller, error) {
	contract, err := bindAggregator(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AggregatorCaller{contract: contract}, nil
}

// NewAggregatorTransactor creates a new write-only instance of Aggregator, bound to a specific deployed contract.
func NewAggregatorTransactor(address common.Address, transactor bind.ContractTransactor) (*AggregatorTransactor, error) {
	contract, err := bindAggregator(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AggregatorTransactor{contract: contract}, nil
}

// NewAggregatorFilterer creates a new log filterer instance of Aggregator, bound to a specific deployed contract.
func NewAggregatorFilterer(address common.Address, filterer bind.ContractFilterer) (*AggregatorFilterer, error) {
	contract, err := bindAggregator(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AggregatorFilterer{contract: contract}, nil
}

// bindAggregator binds a generic wrapper to an already deployed contract.
func bindAggregator(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(AggregatorABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Aggregator *AggregatorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Aggregator.Contract.AggregatorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Aggregator *AggregatorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Aggregator.Contract.AggregatorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Aggregator *AggregatorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Aggregator.Contract.AggregatorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Aggregator *AggregatorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Aggregator.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Aggregator *AggregatorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Aggregator.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Aggregator *AggregatorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Aggregator.Contract.contract.Transact(opts, method, params...)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_Aggregator *AggregatorCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_Aggregator *AggregatorSession) Decimals() (uint8, error) {
	return _Aggregator.Contract.Decimals(&_Aggregator.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_Aggregator *AggregatorCallerSession) Decimals() (uint8, error) {
	return _Aggregator.Contract.Decimals(&_Aggregator.CallOpts)
}

// Description is a free data retrieval call binding the contract method 0x7284e416.
//
// Solidity: function description() view returns(string)
func (_Aggregator *AggregatorCaller) Description(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "description")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Description is a free data retrieval call binding the contract method 0x7284e416.
//
// Solidity: function description() view returns(string)
func (_Aggregator *AggregatorSession) Description() (string, error) {
	return _Aggregator.Contract.Description(&_Aggregator.CallOpts)
}

// Description is a free data retrieval call binding the contract method 0x7284e416.
//
// Solidity: function description() view returns(string)
func (_Aggregator *AggregatorCallerSession) Description() (string, error) {
	return _Aggregator.Contract.Description(&_Aggregator.CallOpts)
}

// GetRoundData is a free data retrieval call binding the contract method 0x9a6fc8f5.
//
// Solidity: function getRoundData(uint80 _roundId) view returns(uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound)
func (_Aggregator *AggregatorCaller) GetRoundData(opts *bind.CallOpts, _roundId *big.Int) (struct {
	RoundId         *big.Int
	Answer          *big.Int
	StartedAt       *big.Int
	UpdatedAt       *big.Int
	AnsweredInRound *big.Int
}, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "getRoundData", _roundId)

	outstruct := new(struct {
		RoundId         *big.Int
		Answer          *big.Int
		StartedAt       *big.Int
		UpdatedAt       *big.Int
		AnsweredInRound *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.RoundId = out[0].(*big.Int)
	outstruct.Answer = out[1].(*big.Int)
	outstruct.StartedAt = out[2].(*big.Int)
	outstruct.UpdatedAt = out[3].(*big.Int)
	outstruct.AnsweredInRound = out[4].(*big.Int)

	return *outstruct, err

}

// GetRoundData is a free data retrieval call binding the contract method 0x9a6fc8f5.
//
// Solidity: function getRoundData(uint80 _roundId) view returns(uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound)
func (_Aggregator *AggregatorSession) GetRoundData(_roundId *big.Int) (struct {
	RoundId         *big.Int
	Answer          *big.Int
	StartedAt       *big.Int
	UpdatedAt       *big.Int
	AnsweredInRound *big.Int
}, error) {
	return _Aggregator.Contract.GetRoundData(&_Aggregator.CallOpts, _roundId)
}

// GetRoundData is a free data retrieval call binding the contract method 0x9a6fc8f5.
//
// Solidity: function getRoundData(uint80 _roundId) view returns(uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound)
func (_Aggregator *AggregatorCallerSession) GetRoundData(_roundId *big.Int) (struct {
	RoundId         *big.Int
	Answer          *big.Int
	StartedAt       *big.Int
	UpdatedAt       *big.Int
	AnsweredInRound *big.Int
}, error) {
	return _Aggregator.Contract.GetRoundData(&_Aggregator.CallOpts, _roundId)
}

// LatestRoundData is a free data retrieval call binding the contract method 0xfeaf968c.
//
// Solidity: function latestRoundData() view returns(uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound)
func (_Aggregator *AggregatorCaller) LatestRoundData(opts *bind.CallOpts) (struct {
	RoundId         *big.Int
	Answer          *big.Int
	StartedAt       *big.Int
	UpdatedAt       *big.Int
	AnsweredInRound *big.Int
}, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "latestRoundData")

	outstruct := new(struct {
		RoundId         *big.Int
		Answer          *big.Int
		StartedAt       *big.Int
		UpdatedAt       *big.Int
		AnsweredInRound *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.RoundId = out[0].(*big.Int)
	outstruct.Answer = out[1].(*big.Int)
	outstruct.StartedAt = out[2].(*big.Int)
	outstruct.UpdatedAt = out[3].(*big.Int)
	outstruct.AnsweredInRound = out[4].(*big.Int)

	return *outstruct, err

}

// LatestRoundData is a free data retrieval call binding the contract method 0xfeaf968c.
//
// Solidity: function latestRoundData() view returns(uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound)
func (_Aggregator *AggregatorSession) LatestRoundData() (struct {
	RoundId         *big.Int
	Answer          *big.Int
	StartedAt       *big.Int
	UpdatedAt       *big.Int
	AnsweredInRound *big.Int
}, error) {
	return _Aggregator.Contract.LatestRoundData(&_Aggregator.CallOpts)
}

// LatestRoundData is a free data retrieval call binding the contract method 0xfeaf968c.
//
// Solidity: function latestRoundData() view returns(uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound)
func (_Aggregator *AggregatorCallerSession) LatestRoundData() (struct {
	RoundId         *big.Int
	Answer          *big.Int
	StartedAt       *big.Int
	UpdatedAt       *big.Int
	AnsweredInRound *big.Int
}, error) {
	return _Aggregator.Contract.LatestRoundData(&_Aggregator.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(uint256)
func (_Aggregator *AggregatorCaller) Version(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(uint256)
func (_Aggregator *AggregatorSession) Version() (*big.Int, error) {
	return _Aggregator.Contract.Version(&_Aggregator.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(uint256)
func (_Aggregator *AggregatorCallerSession) Version() (*big.Int, error) {
	return _Aggregator.Contract.Version(&_Aggregator.CallOpts)
}
