// Package clients contains code for managing Ethereum clients needed for the AAVE Lending Pool.
package clients

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"aggregator"
	"erc20"
	"lendingpool"
	"wallets"
	"weth9"
)

const (
	weth9Address = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
)

type aggregatorEntry struct {
	name       string
	aggregator *common.Address
}

var (
	// priceFeeds has references to chainlink aggregator addresses with prices in ETH keyed by
	// asset address hex strings.
	//
	// Map keys are hex strings for comparability.
	priceFeeds map[string]aggregatorEntry
)

func init() {
	priceFeeds = make(map[string]aggregatorEntry)
	for _, entry := range []struct {
		name, token, aggregator string
	}{
		{"USDT",
			"0xdAC17F958D2ee523a2206206994597C13D831ec7", "0xEe9F2375b4bdF6387aa8265dD4FB8F16512A1d46"},
		{"WBTC",
			"0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", "0xdeb288F737066589598e9214E782fa5A8eD689e8"},
		{"WETH9", weth9Address, ""}, // 1 by definition.
		{"YFI",
			"0x0bc529c00C6401aEF6D220BE8C6Ea1667F6Ad93e", "0x7c5d4F8345e66f68099581Db340cd65B078C41f4"},
		{"ZRXToken",
			"0xE41d2489571d322189246DaFA5ebDe1F4699F498", "0x2Da4983a622a8498bb1a21FaE9D8F6C664939962"},
		{"Uni",
			"0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984", "0xD6aA3D25116d8dA79Ea0246c4826EB951872e02e"},
		{"AAVE",
			"0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9", "0x6Df09E975c830ECae5bd4eD9d90f3A95a4f88012"},
		{"BAToken",
			"0x0D8775F648430679A709E98d2b0Cb6250d2887EF", "0x0d16d4528239e9ee52fa531af613AcdB23D88c94"},
		{"Binance USD",
			"0x4Fabb145d64652a948d72533023f6E7A623C7C53", "0x614715d2Af89E6EC99A233818275142cE88d1Cfd"},
		{"Dai",
			"0x6B175474E89094C44Da98b954EedeAC495271d0F", "0x773616E4d11A78F511299002da57A0a94577F1f4"},
		{"ENJToken",
			"0xF629cBd94d3791C9250152BD8dfBDF380E2a3B9c", "0x24D9aB51950F3d62E9144fdC2f3135DAA6Ce8D1B"},
		{"KyberNetworkCrystal",
			"0xdd974D5C2e2928deA5F71b9825b8b646686BD200", "0x656c0544eF4C98A6a98491833A89204Abb045d6b"},
		{"LinkToken",
			"0x514910771AF9Ca656af840dff83E8264EcF986CA", "0xDC530D9457755926550b59e8ECcdaE7624181557"},
		{"MANAToken",
			"0x0F5D2fB29fb7d3CFeE444a200298f468908cC942", "0x82A44D92D6c329826dc557c5E1Be6ebeC5D5FeB9"},
		{"DSToken", "0x9f8F72aA9304c8B593d555F12eF6589cC3A579A2", ""}, // No Chainlink feed.
		{"RepublicToken",
			"0x408e41876cCCDC0F92210600ef50372656052a38", "0xF1939BECE7708382b5fb5e559f630CB8B39a10ee"},
		{"Synthetix Network Token",
			"0xC011a73ee8576Fb46F5E1c5751cA3B9Fe0af2a6F", "0xF9A76ae7a1075Fe7d646b06fF05Bd48b9FA5582e"},
		{"Synth sUSD",
			"0x57Ab1ec28D129707052df4dF418D58a2D46d5f51", "0xb343e7a1aF578FA35632435243D814e7497622f7"},
		{"TrueUSD",
			"0x0000000000085d4780B73119b644AE5ecd22b376", "0x7aeCF1c19661d12E962b69eBC8f6b2E63a55C660"},
		{"USD Coin",
			"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", "0x64EaC61A2DFda2c3Fa04eED49AA33D021AeC8838"},
		{"Vyper_contract",
			"0xD533a949740bb3306d119CC777fa900bA034cd52", "0x8a12Be339B0cD1829b91Adc01977caa5E9ac121e"},
		{"Gemini dollar", "0x056Fd409E1d7A124BD7017459dFEa2F387b6d5Cd", ""}, // No Chainlink feed.
	} {
		var aggregator *common.Address
		if common.IsHexAddress(entry.aggregator) {
			addr := common.HexToAddress(entry.aggregator)
			aggregator = &addr
		}
		priceFeeds[entry.token] = aggregatorEntry{
			name:       entry.name,
			aggregator: aggregator,
		}
	}
}

// Client primarily encapsulates Lending Pool operations.
type Client struct {
	eth          *ethclient.Client
	bot          *wallets.Wallet
	weth9Address common.Address
	weth         *weth9.Weth9
	lpAddress    common.Address
	lp           *lendingpool.Lendingpool
	// tokens maps `common.Address` token addresses to `*erc20.Erc20` token instances.
	tokens *sync.Map
	// prices maps `common.Address` token addresses to `*erc20.Erc20` token instances.
	prices *sync.Map
}

// Params wraps parameters used to construct the client.
type Params struct {
	ETHURI             string
	BotKey             string
	WETH9Address       common.Address
	LendingPoolAddress common.Address
}

// NewClient initializes a new Client instance.
func NewClient(params *Params) (*Client, error) {
	eth, err := ethclient.Dial(params.ETHURI)
	if err != nil {
		return nil, fmt.Errorf("dialing %s: %w", params.ETHURI, err)
	}
	bot, err := wallets.NewWallet(params.BotKey)
	if err != nil {
		return nil, fmt.Errorf("bot wallet from key %s: %w", params.BotKey, err)
	}
	weth, err := weth9.NewWeth9(params.WETH9Address, eth)
	if err != nil {
		return nil, fmt.Errorf("creating WETH from %s: %w", params.WETH9Address, err)
	}
	lp, err := lendingpool.NewLendingpool(params.LendingPoolAddress, eth)
	if err != nil {
		return nil, fmt.Errorf("creating LendingPool from %s: %w", params.LendingPoolAddress, err)
	}

	return &Client{
		eth:          eth,
		bot:          bot,
		weth9Address: params.WETH9Address,
		weth:         weth,
		lpAddress:    params.LendingPoolAddress,
		lp:           lp,
		tokens:       new(sync.Map),
		prices:       new(sync.Map),
	}, nil
}

// ETH provides access to the underlying ETH client.
func (c *Client) ETH() *ethclient.Client {
	return c.eth
}

// Execute runs the transaction `t` using credentials of `from`.
func (c *Client) Execute(ctx context.Context, from *wallets.Wallet, desc string,
	t func(*bind.TransactOpts) (*types.Transaction, error)) error {
	txr, err := from.NewTransactor(ctx, c.eth)
	if err != nil {
		return fmt.Errorf("creating transactor for %s: %w", desc, err)
	}
	tx, err := t(txr)
	if err != nil {
		return fmt.Errorf("executing %s: %w", desc, err)
	}
	r, err := c.eth.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return fmt.Errorf("obtaining receipt for %s: %w", desc, err)
	}
	if r.Status != 1 {
		return fmt.Errorf("%s transaction failed: %v", desc, r)
	}
	return nil
}

// Execute performs the transaction `t` using the bot's credentials.
func (c *Client) ExecuteAsBot(ctx context.Context, desc string,
	t func(*bind.TransactOpts) (*types.Transaction, error)) error {
	return c.Execute(ctx, c.bot, desc, t)
}

// Token returns an Erc20 token interface for the given token address.
func (c *Client) Token(addr common.Address) (*erc20.Erc20, error) {
	v, ok := c.tokens.Load(addr)
	if !ok {
		var err error
		v, err = erc20.NewErc20(addr, c.eth)
		if err != nil {
			return nil, fmt.Errorf("erc20 client for %v: %w", addr, err)
		}
		v, _ = c.tokens.LoadOrStore(addr, v)
	}
	return v.(*erc20.Erc20), nil
}

// Aggregator returns an `aggregator.Aggregator` instance for the given hex string token address.
func (c *Client) Aggregator(addr string) (*aggregator.Aggregator, error) {
	v, ok := c.prices.Load(addr)
	if !ok {
		entry := priceFeeds[addr]
		if entry.aggregator == nil {
			return nil, fmt.Errorf("no aggregator for %s (%s)", addr, entry.name)
		}
		var err error
		v, err = aggregator.NewAggregator(*entry.aggregator, c.eth)
		if err != nil {
			return nil, fmt.Errorf("aggregator client for %s: %w", addr, err)
		}
		v, _ = c.prices.LoadOrStore(addr, v)
	}
	return v.(*aggregator.Aggregator), nil
}

// PriceOf returns the price of the asset at hex address `addr` in ETH.
func (c *Client) PriceOf(ctx context.Context, addr string) (*big.Int, *big.Int, error) {
	var price *big.Int
	var decimals uint8
	if addr == weth9Address {
		price = big.NewInt(1)
		decimals = 0
	} else {
		agg, err := c.Aggregator(addr)
		if err != nil {
			return nil, nil, fmt.Errorf("getting aggregator for %s: %w", addr, err)
		}
		decimals, err = agg.Decimals(&bind.CallOpts{Context: ctx})
		if err != nil {
			return nil, nil, fmt.Errorf("getting decimals for %s: %w", addr, err)
		}
		data, err := agg.LatestRoundData(&bind.CallOpts{Context: ctx})
		if err != nil {
			return nil, nil, fmt.Errorf("getting price data for %s: %w", addr, err)
		}
		price = data.Answer
	}
	decFactor := big.NewInt(1)
	for i := uint8(0); i < decimals; i++ {
		decFactor = decFactor.Mul(decFactor, big.NewInt(10))
	}
	return price, decFactor, nil
}

// DepositETH deposits ETH into the lending pool from the given wallet. Used for testing.
func (c *Client) DepositETH(ctx context.Context, from *wallets.Wallet, amount *big.Int) error {
	txr, err := from.NewTransactor(ctx, c.eth)
	if err != nil {
		return fmt.Errorf("depositing WETH: %w", err)
	}
	txr.Value = amount // Setting the value means we can't user the higher-level `Execute` method.
	tx, err := c.weth.Deposit(txr)
	if err != nil {
		return fmt.Errorf("wrapping ETH: %w", err)
	}
	r, err := c.eth.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return err
	}
	if r.Status != 1 {
		return fmt.Errorf("wrapping ETH transaction failed: %v", r)
	}

	if err = c.Execute(ctx, from, "approving lending pool for WETH",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			return c.weth.Approve(txr, c.lpAddress, amount)
		}); err != nil {
		return err
	}

	if err = c.Execute(ctx, from, "depositing WETH collateral",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			return c.lp.Deposit(txr, c.weth9Address, amount, from.Address, 0)
		}); err != nil {
		return err
	}

	return nil
}

// Borrow borrows the given asset for the given wallet at the stable rate. Used for testing.
func (c *Client) Borrow(ctx context.Context, onBehalfOf *wallets.Wallet, asset common.Address, amount *big.Int) error {
	return c.Execute(ctx, onBehalfOf, fmt.Sprintf("borrowing %v", asset),
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			return c.lp.Borrow(txr, asset, amount, big.NewInt(1), 0, onBehalfOf.Address)
		})
}

// LoanData contains information about a loan.
type LoanData struct {
	Collateral           common.Address
	CollateralName       string
	CollateralAmount     *big.Int
	AToken               common.Address
	DebtName             string
	Debt                 common.Address
	DebtAmount           *big.Int
	CurrentRatio         *big.Rat
	LiquidationThreshold uint16
	ConfiguredRatio      *big.Int
}

// RetrieveLoanData retrieves loan information for the given user.
func (c *Client) RetrieveLoanData(ctx context.Context, u common.Address) (*LoanData, error) {
	reserves, err := c.lp.GetReservesList(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("getting reserves list: %v", err)
	}
	config, err := c.lp.GetUserConfiguration(&bind.CallOpts{Context: ctx}, u)
	if err != nil {
		return nil, fmt.Errorf("getting user configuration: %v", err)
	}
	var collateral []common.Address
	var debt []common.Address
	for i, asset := range reserves {
		if config.Data.Bit(2*i+1) > 0 {
			collateral = append(collateral, asset)
		}
		if config.Data.Bit(2*i) > 0 {
			debt = append(debt, asset)
		}
	}
	if len(collateral) != 1 || len(debt) != 1 {
		return nil, fmt.Errorf(`only 1 collateral asset and 1 debt asset supported
collateral assets: %v, debt assets: %v`, collateral, debt)
	}

	cInfo, err := c.lp.GetReserveData(&bind.CallOpts{Context: ctx}, collateral[0])
	if err != nil {
		return nil, fmt.Errorf("retrieving reserve data for %v: %v", collateral[0], err)
	}

	configBytes := cInfo.Configuration.Data.Bytes() // `Bytes` returns big endian.
	threshold := binary.BigEndian.Uint16(configBytes[len(configBytes)-4 : len(configBytes)-2])

	cAmount, err := c.BalanceOf(ctx, cInfo.ATokenAddress, u)
	if err != nil {
		return nil, fmt.Errorf("balance for user %v: %w", u, err)
	}

	dInfo, err := c.lp.GetReserveData(&bind.CallOpts{Context: ctx}, debt[0])
	if err != nil {
		return nil, fmt.Errorf("retrieving reserve data for %v: %v", debt[0], err)
	}

	sdAmount, err := c.BalanceOf(ctx, dInfo.StableDebtTokenAddress, u)
	if err != nil {
		return nil, fmt.Errorf("retrieving stable debt balance for %v: %w", u, err)
	}

	vdAmount, err := c.BalanceOf(ctx, dInfo.VariableDebtTokenAddress, u)
	if err != nil {
		return nil, fmt.Errorf("retrieving variable debt balance for %v: %w", u, err)
	}

	dAmount := new(big.Int).Add(sdAmount, vdAmount)

	cPrice, cFactor, err := c.PriceOf(ctx, collateral[0].Hex())
	if err != nil {
		return nil, fmt.Errorf("converting collateral %v to eth: %w", collateral[0], err)
	}
	dPrice, dFactor, err := c.PriceOf(ctx, debt[0].Hex())
	if err != nil {
		return nil, fmt.Errorf("converting debt %v to eth: %w", debt[0], err)
	}

	cEthAmount := new(big.Int).Mul(cAmount, cPrice)
	dEthAmount := new(big.Int).Mul(dAmount, dPrice)

	ratio := new(big.Rat).SetFrac(dEthAmount, cEthAmount)
	ratio = ratio.Mul(ratio, new(big.Rat).SetFrac(cFactor, dFactor))

	return &LoanData{
		Collateral:           collateral[0],
		CollateralName:       priceFeeds[collateral[0].Hex()].name,
		CollateralAmount:     cAmount,
		AToken:               cInfo.ATokenAddress,
		Debt:                 debt[0],
		DebtName:             priceFeeds[debt[0].Hex()].name,
		DebtAmount:           sdAmount.Add(sdAmount, vdAmount),
		CurrentRatio:         ratio,
		LiquidationThreshold: threshold,
	}, nil
}

// BalanceOf returns the balance of the given ERC20 asset in the given wallet.
func (c *Client) BalanceOf(ctx context.Context, asset common.Address, u common.Address) (*big.Int, error) {
	token, err := c.Token(asset)
	if err != nil {
		return nil, fmt.Errorf("getting token %v: %w", asset, err)
	}
	amount, err := token.BalanceOf(&bind.CallOpts{Context: ctx}, u)
	if err != nil {
		return nil, fmt.Errorf("querying balance of token %v: %w", asset, u, err)
	}
	return amount, nil
}

// AToken returns the ERC20 token of the AToken for the given asset.
func (c *Client) AToken(ctx context.Context, asset common.Address) (*erc20.Erc20, error) {
	data, err := c.lp.GetReserveData(&bind.CallOpts{Context: ctx}, asset)
	if err != nil {
		return nil, fmt.Errorf("retrieving reserve data for %v: %v", asset, err)
	}
	token, err := c.Token(data.ATokenAddress)
	if err != nil {
		return nil, fmt.Errorf("getting AToken for %v: %w", asset, err)
	}
	return token, nil
}
