package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"text/template"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"

	"abis"
	"clients"
	"env"
	"erc20"
	"protection"
	"scenarios"
	"wallets"
)

// ethURI = "wss://mainnet.infura.io/ws/v3/92e15d2bbf9b41d286544a680c4b23d0"
// ethURI = "wss://goerli.infura.io/ws/v3/92e15d2bbf9b41d286544a680c4b23d0"

var (
	// ethUSDAggregator is the Aggregator proxied by 0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419.
	ethUSDAggregator = common.HexToAddress("0x00c7A37B03690fb9f41b5C5AF8131735C7275446")
	// btcUSDAggregator is the Aggregator proxied by 0xF4030086522a5bEEa4988F8cA5B36dbC97BeE88c.
	btcUSDAggregator = common.HexToAddress("0xF570deEffF684D964dc3E15E1F9414283E3f7419")
)

func main() {
	params := env.LocalTestNet()
	client, err := clients.NewClient(params)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	ctx := context.Background()

	listenPrices(ctx, client.ETH())

	// Serves ./views using gin.
	router := gin.Default()
	router.Use(static.Serve("/", static.LocalFile("../../ui/dist", true)))

	api := router.Group("/api")
	api.GET("/state", func(c *gin.Context) {
		hexAddr := c.Query("address")
		log.Printf("got state query addr: %v", hexAddr)
		if !common.IsHexAddress(hexAddr) {
			c.AbortWithError(400, fmt.Errorf("%s is not a hex address", hexAddr))
			return
		}
		addr := common.HexToAddress(hexAddr)
		loan, err := client.Loan(ctx, addr)
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
		amount, err := loan.Data(ctx, client)
		if err != nil {
			c.AbortWithError(400, err)
			return
		}

		threshold := float64(loan.LiquidationThreshold) / float64(10000)
		c.JSON(http.StatusOK, gin.H{
			"collateral-name":             loan.CollateralName,
			"collateral-address":          loan.Collateral.String(),
			"collateral-amount":           amount.CollateralAmount.String(),
			"a-token-address":             loan.AToken.String(),
			"debt-name":                   loan.DebtName,
			"debt-address":                loan.Debt.String(),
			"debt-amount":                 amount.DebtAmount.String(),
			"current-ratio":               amount.CurrentRatio.FloatString(10),
			"liquidation-threshold":       fmt.Sprintf("%.4f", threshold),
			"protection-contract-address": contractAddress.String(),
		})
	})
	api.GET("/abi", func(c *gin.Context) {
		switch name := c.Query("name"); name {
		case "erc20":
			c.DataFromReader(http.StatusOK, int64(len(erc20.Erc20ABI)), gin.MIMEJSON,
				strings.NewReader(erc20.Erc20ABI), nil)
		case "protection":
			c.DataFromReader(http.StatusOK, int64(len(protection.ProtectionABI)), gin.MIMEJSON,
				strings.NewReader(protection.ProtectionABI), nil)
		default:
			c.AbortWithError(400, fmt.Errorf("unknown ABI %s", name))
			return
		}
	})

	router.Run(":3000")
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
