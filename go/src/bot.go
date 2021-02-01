package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"

	"abis"
	"clients"
	"erc20"
	"protection"
	"wallets"
)

const (
	// ethURI = "wss://mainnet.infura.io/ws/v3/92e15d2bbf9b41d286544a680c4b23d0"
	// ethURI = "wss://goerli.infura.io/ws/v3/92e15d2bbf9b41d286544a680c4b23d0"
	ethURI = "ws://localhost:8545"

	// The keys here appear to be stable values used by local hardhat nodes.
	botKey  = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	userKey = "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
)

var (
	// ethUSDAggregator is the Aggregator proxied by 0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419.
	ethUSDAggregator = common.HexToAddress("0x00c7A37B03690fb9f41b5C5AF8131735C7275446")
	// btcUSDAggregator is the Aggregator proxied by 0xF4030086522a5bEEa4988F8cA5B36dbC97BeE88c.
	btcUSDAggregator = common.HexToAddress("0xF570deEffF684D964dc3E15E1F9414283E3f7419")
	wETH9Address     = common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")
	daiAddress       = common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F")
	// This is the proxy address for tusd. The implementation address is:
	// 0xffc40F39806F1400d8278BfD33823705b5a4c196. Not sure which one AAVE uses.
	tusdAddress        = common.HexToAddress("0x0000000000085d4780B73119b644AE5ecd22b376")
	aETHAddress        = common.HexToAddress("0x030bA81f1c18d280636F32af80b9AAd02Cf0854e")
	lendingPoolAddress = common.HexToAddress("0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9")

	oneInchTemplate = template.Must(template.New("1inch API").Parse(
		"https://api.1inch.exchange/v2.0/swap?" +
			"fromTokenAddress={{.From}}&toTokenAddress={{.To}}" +
			"&amount={{.Amount}}&fromAddress={{.OnBehalfOf}}&slippage={{.Slippage}}" +
			"&disableEstimate=true"))

	loanAmount *big.Int

	ethAggregators = map[common.Address]common.Address{
		daiAddress: common.HexToAddress("0xd866A07Dea5Ee3c093e21d33660b5579C21F140b"),
		// Proxied by 0x3886BA987236181D98F2401c507Fb8BeA7871dF2.
		tusdAddress: common.HexToAddress("0x0c632eC5982e3A8dC116a02ebA7A419efec170B1"),
	}
)

func init() {
	var s bool
	loanAmount, s = new(big.Int).SetString("801000000000000000000", 10) // 801 DAI
	if !s {
		log.Fatalf("setting loan amount")
	}
}

func main() {
	client, err := clients.NewClient(&clients.Params{
		ETHURI:             ethURI,
		BotKey:             botKey,
		WETH9Address:       wETH9Address,
		LendingPoolAddress: lendingPoolAddress,
	})
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	ctx := context.Background()

	// Serves ./views using gin.
	router := gin.Default()
	router.Use(static.Serve("/", static.LocalFile("./views", true)))

	var contractAddress common.Address
	var p *protection.Protection
	if err := client.ExecuteAsBot(ctx, "deploying protection contract",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			var tx *types.Transaction
			var err error
			contractAddress, tx, p, err = protection.DeployProtection(txr, client.ETH())
			return tx, err
		}); err != nil {
		log.Fatalf("Error: %v", err)
	}

	user, err := wallets.NewWallet(userKey)
	if err != nil {
		log.Fatalf("Error creating user wallet: %v", err)
	}

	if err = setupLoan(ctx, client, user); err != nil {
		log.Fatalf("Error setting up loan: %v", err)
	}
	log.Printf("loan succeeded")

	/* if err = registerProtection(ctx, client, wETH9Address, daiAddress, p, contractAddress, user); err != nil {
		log.Fatalf("Error registering for protection: %v", err)
	}
	log.Printf("protection registration succeeded")

	aETH, err := client.Token(aETHAddress)
	if err != nil {
		log.Fatalf("Error creating aETH client: %v", err)
	}

	if err = executeProtection(ctx, client, aETH, p, contractAddress, user); err != nil {
		log.Fatalf("Error executing protection: %v", err)
	}
	log.Printf("protection execution succeeded") */

	listenPrices(ctx, client.ETH())

	api := router.Group("/api")
	api.GET("/state", func(c *gin.Context) {
		hexAddr := c.Query("address")
		log.Printf("got state query addr: %v", hexAddr)
		if !common.IsHexAddress(hexAddr) {
			c.AbortWithError(400, fmt.Errorf("%s is not a hex address", hexAddr))
			return
		}
		addr := common.HexToAddress(hexAddr)
		data, err := client.RetrieveLoanData(ctx, addr)
		if err != nil {
			c.AbortWithError(400, err)
			return
		}

		threshold := float64(data.LiquidationThreshold) / float64(10000)
		c.JSON(http.StatusOK, gin.H{
			"collateral-name":       data.CollateralName,
			"collateral-address":    data.Collateral.String(),
			"collateral-amount":     data.CollateralAmount.String(),
			"debt-name":             data.DebtName,
			"debt-address":          data.Debt.String(),
			"debt-amount":           data.DebtAmount.String(),
			"current-ratio":         data.CurrentRatio.FloatString(10),
			"liquidation-threshold": fmt.Sprintf("%.4f", threshold),
		})
	})

	router.Run(":3000")
}

func setupLoan(ctx context.Context, client *clients.Client, user *wallets.Wallet) error {
	cAmount := new(big.Int).Mul(big.NewInt(1), big.NewInt(params.Ether))
	if err := client.DepositETH(ctx, user, cAmount); err != nil {
		return fmt.Errorf("setting up loan: %w", err)
	}

	// An Ether is 1e18 Wei and Dai is expressed at the same ratio, so the same constant can be used
	// here.
	dAmount := new(big.Int).Mul(big.NewInt(500), big.NewInt(params.Ether))
	if err := client.Borrow(ctx, user, daiAddress, dAmount); err != nil {
		return fmt.Errorf("setting up load: %w", err)
	}
	return nil
}

func registerProtection(ctx context.Context, client *clients.Client, cAsset common.Address, dAsset common.Address, lp *protection.Protection, contractAddress common.Address, user *wallets.Wallet) error {
	aToken, err := client.AToken(ctx, cAsset)
	if err != nil {
		return fmt.Errorf("registering protection for %v: %w", user, err)
	}

	if err := client.Execute(ctx, user, fmt.Sprintf("approving protection contract for %v", cAsset),
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			return aToken.Approve(txr, contractAddress, big.NewInt(-1))
		}); err != nil {
		return err
	}

	if err := client.Execute(ctx, user, "registering for protection",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			txr.Value = big.NewInt(9500000)
			// XXX use a non-bogus value here.
			return lp.Register(txr, cAsset, dAsset, 0)
		}); err != nil {
		return err
	}

	return nil
}

func executeProtection(ctx context.Context, client *clients.Client, aETH *erc20.Erc20, lp *protection.Protection, contractAddress common.Address, user *wallets.Wallet) error {
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

	oneInchClient := http.Client{Timeout: time.Second * 10}

	req, err := http.NewRequest(http.MethodGet, buf.String(), nil)
	if err != nil {
		return fmt.Errorf("preparing 1inch request: %w", err)
	}
	req.Header.Set("User-Agent", "AVVE Liquidation Protection Bot")

	res, err := oneInchClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing 1inch request: %w", err)
	}
	if res.Body != nil {
		defer res.Body.Close()
	}
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading 1inch response: %w", err)
	}

	var parsed interface{}
	if err = json.NewDecoder(bytes.NewReader(content)).Decode(&parsed); err != nil {
		return fmt.Errorf("decoding 1inch response %s: %w", content, err)
	}
	calldataHex := parsed.(map[string]interface{})["tx"].(map[string]interface{})["data"].(string)
	calldata, err := hex.DecodeString(calldataHex[2:])
	if err != nil {
		return fmt.Errorf("decoding 1inch calldata %s: %w", content, err)
	}

	if err = client.ExecuteAsBot(ctx, "executing protection",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			return lp.Execute(txr, user.Address, calldata)
		}); err != nil {
		return err
	}

	return nil
}

func listenPrices(ctx context.Context, eth *ethclient.Client) {
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
