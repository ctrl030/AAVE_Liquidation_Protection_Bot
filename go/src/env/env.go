// Package env contains environmental parameters.
package env

import (
	"github.com/ethereum/go-ethereum/common"
)

const (
	ethURI = "ws://localhost:8545"
	// The keys here are stable values used by local hardhat nodes.
	botKey  = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	userKey = "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
)

var (
	lendingPoolAddress = common.HexToAddress("0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9")
	weth9Address       = common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")
	daiAddress         = common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F")
)

// Params encapsulates environmental parameters.
type Params interface {
	ETHURI() string
	BotKey() string
	LendingPoolAddress() common.Address

	// TODO(greatfilter): the parameters below are only used in testing. Factor this out?
	UserKey() string
	WETH9Address() common.Address
	DaiAddress() common.Address
}

type localTestNet struct{}

func LocalTestNet() Params {
	return &localTestNet{}
}

func (_ *localTestNet) ETHURI() string {
	return "ws://localhost:8545"
}

func (_ *localTestNet) BotKey() string {
	return "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
}

func (_ *localTestNet) UserKey() string {
	return "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
}

func (_ *localTestNet) LendingPoolAddress() common.Address {
	return lendingPoolAddress
}

func (_ *localTestNet) WETH9Address() common.Address {
	return weth9Address
}

func (_ *localTestNet) DaiAddress() common.Address {
	return daiAddress
}
