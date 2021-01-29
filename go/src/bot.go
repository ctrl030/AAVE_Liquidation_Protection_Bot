package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"

	"abis"
	"erc20"
	"lendingpool"
	"protection"
	"wallets"
	"weth9"
)

const (
	// ethURI = "wss://mainnet.infura.io/ws/v3/92e15d2bbf9b41d286544a680c4b23d0"
	// ethURI = "wss://goerli.infura.io/ws/v3/92e15d2bbf9b41d286544a680c4b23d0"
	ethURI = "ws://localhost:8545"
)

var (
	// ethUSDAggregator is the Aggregator proxied by 0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419.
	ethUSDAggregator = common.HexToAddress("0x00c7A37B03690fb9f41b5C5AF8131735C7275446")
	// btcUSDAggregator is the Aggregator proxied by 0xF4030086522a5bEEa4988F8cA5B36dbC97BeE88c.
	btcUSDAggregator   = common.HexToAddress("0xF570deEffF684D964dc3E15E1F9414283E3f7419")
	wETH9Address       = common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")
	daiAddress         = common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F")
	aETHAddress        = common.HexToAddress("0x030bA81f1c18d280636F32af80b9AAd02Cf0854e")
	lendingPoolAddress = common.HexToAddress("0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9")

	oneInchTemplate = template.Must(template.New("1inch API").Parse(
		"https://api.1inch.exchange/v2.0/swap?" +
			"fromTokenAddress={{.From}}&toTokenAddress={{.To}}" +
			"&amount={{.Amount}}&fromAddress={{.OnBehalfOf}}&slippage={{.Slippage}}" +
			"&disableEstimate=true"))

	loanAmount *big.Int
)

func init() {
	var s bool
	loanAmount, s = new(big.Int).SetString("801000000000000000000", 10) // 801 DAI
	if !s {
		log.Fatalf("setting loan amount")
	}
}

func main() {
	eth, err := ethclient.Dial(ethURI)
	if err != nil {
		log.Fatalf("Error dialing %s: %v", ethURI, err)
	}
	defer eth.Close()

	ctx := context.Background()

	bot, err := wallets.NewWallet("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	if err != nil {
		log.Fatalf("Failed to create wallet: %v", err)
	}

	var contractAddress common.Address
	var p *protection.Protection
	if err = bot.Execute(ctx, eth, "deploying protection contract",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			var tx *types.Transaction
			var err error
			contractAddress, tx, p, err = protection.DeployProtection(txr, eth)
			return tx, err
		}); err != nil {
		log.Fatalf("Error: %v", err)
	}

	wETH, err := weth9.NewWeth9(wETH9Address, eth)
	if err != nil {
		log.Fatalf("Error creating WETH client: %v", err)
	}

	lp, err := lendingpool.NewLendingpool(lendingPoolAddress, eth)
	if err != nil {
		log.Fatalf("Error creating LendingPool client: %v", err)
	}

	aETH, err := erc20.NewErc20(aETHAddress, eth)
	if err != nil {
		log.Fatalf("Error creating aETH client: %v", err)
	}

	user, err := wallets.NewWallet("59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d")
	if err != nil {
		log.Fatalf("Error creating user wallet: %v", err)
	}

	if err = setupLoan(ctx, eth, wETH, lp, user); err != nil {
		log.Fatalf("Error setting up loan: %v", err)
	}
	log.Printf("loan succeeded")

	if err = registerProtection(ctx, eth, aETH, p, contractAddress, bot, user); err != nil {
		log.Fatalf("Error registering for protection: %v", err)
	}
	log.Printf("protection registration succeeded")

	if err = executeProtection(ctx, eth, aETH, p, contractAddress, bot, user); err != nil {
		log.Fatalf("Error executing protection: %v", err)
	}
	log.Printf("protection execution succeeded")

	listenAndWait(ctx, eth)
}

func setupLoan(ctx context.Context, eth *ethclient.Client, wETH *weth9.Weth9, lp *lendingpool.Lendingpool, user *wallets.Wallet) error {
	txr, err := user.NewTransactor(ctx, eth)
	if err != nil {
		return err
	}
	amount := new(big.Int).Mul(big.NewInt(1), big.NewInt(params.Ether))
	txr.Value = amount
	tx, err := wETH.Deposit(txr)
	if err != nil {
		return fmt.Errorf("wrapping ETH: %w", err)
	}
	r, err := eth.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return err
	}
	if r.Status != 1 {
		return fmt.Errorf("wrapping ETH transaction failed: %v", r)
	}

	if err = user.Execute(ctx, eth, "approving lending pool for WETH",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			return wETH.Approve(txr, lendingPoolAddress, amount)
		}); err != nil {
		return err
	}

	if err = user.Execute(ctx, eth, "depositing WETH collateral",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			return lp.Deposit(txr, wETH9Address, amount, user.Address, 0)
		}); err != nil {
		return err
	}

	if err = user.Execute(ctx, eth, "borrowing DAI",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			return lp.Borrow(txr, daiAddress, loanAmount, big.NewInt(1), 0, user.Address)
		}); err != nil {
		return err
	}

	return nil
}

func registerProtection(ctx context.Context, eth *ethclient.Client, aETH *erc20.Erc20, lp *protection.Protection, contractAddress common.Address, bot *wallets.Wallet, user *wallets.Wallet) error {
	if err := user.Execute(ctx, eth, "approving protection contract for aETH",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			return aETH.Approve(txr, contractAddress, big.NewInt(-1))
		}); err != nil {
		return err
	}

	if err := user.Execute(ctx, eth, "registering for protection",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			txr.Value = big.NewInt(9500000)
			return lp.Register(txr, wETH9Address, daiAddress, 0)
		}); err != nil {
		return err
	}

	return nil
}

func executeProtection(ctx context.Context, eth *ethclient.Client, aETH *erc20.Erc20, lp *protection.Protection, contractAddress common.Address, bot *wallets.Wallet, user *wallets.Wallet) error {
	balance, err := aETH.BalanceOf(&bind.CallOpts{Context: ctx}, user.Address)
	if err != nil {
		return fmt.Errorf("retrieving user collateral balance: %w", err)
	}

	buf := &strings.Builder{}
	if err := oneInchTemplate.Execute(buf, struct {
		From, To, OnBehalfOf common.Address
		Amount               *big.Int
		Slippage             string
	}{
		wETH9Address, daiAddress, contractAddress, balance, "1",
	}); err != nil {
		return fmt.Errorf("preparing url: %w", err)
	}

	oneInchClient := http.Client{Timeout: time.Second * 5}

	req, err := http.NewRequest(http.MethodGet, buf.String(), nil)
	if err != nil {
		return fmt.Errorf("preparing 1inch request: %w", err)
	}
	req.Header.Set("User-Agent", "AVVE Liquidation Protection Bot")

	res, err := oneInchClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing oneInch request: %w", err)
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	var parsed interface{}
	if err = json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("decoding oneInch response: %w", err)
	}
	calldataHex := parsed.(map[string]interface{})["tx"].(map[string]interface{})["data"].(string)
	calldata, err := hex.DecodeString(calldataHex[2:])
	if err != nil {
		return fmt.Errorf("decoding oneinch calldata: %w", err)
	}

	if err = bot.Execute(ctx, eth, "executing protection",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			return lp.Execute(txr, user.Address, calldata)
		}); err != nil {
		return err
	}

	return nil
}

func listenAndWait(ctx context.Context, eth *ethclient.Client) {
	fromBlock, err := eth.BlockNumber(ctx)
	if err != nil {
		log.Fatalf("Error getting block number: %v", err)
	}

	go listenAggregator(ctx, eth, ethUSDAggregator, fromBlock,
		func(l *types.Log, a *abis.AnswerUpdated) {
			log.Printf("Eth (%v): %+v", l.BlockHash, *a)
		})
	go listenAggregator(ctx, eth, btcUSDAggregator, fromBlock,
		func(l *types.Log, a *abis.AnswerUpdated) {
			log.Printf("BTC (%v): %+v", l.BlockHash, *a)
		})

	select {} // Waits forever.
}

func listenAggregator(ctx context.Context, eth *ethclient.Client, aggregator common.Address,
	latest uint64, cb func(*types.Log, *abis.AnswerUpdated)) {
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(latest)),
		Topics:    abis.AnswerUpdatedTopics,
		Addresses: []common.Address{aggregator},
	}

	logs := make(chan types.Log, 2)
	s, err := eth.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		log.Fatalf("Error subscribing %v: %v", query, err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-s.Err():
			log.Printf("Logs subscription error: %v", err)
			break
		case l := <-logs:
			a, err := abis.AnswerUpdatedFromLog(l)
			if err != nil {
				log.Printf("error unpacking event: %v", err)
				continue
			}
			cb(&l, a)
		}
	}
}
