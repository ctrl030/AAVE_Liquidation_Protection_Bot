package main

import (
	"context"
	"log"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"abis"
)

const (
	ethURI = "wss://mainnet.infura.io/ws/v3/92e15d2bbf9b41d286544a680c4b23d0"
	// ethURI = "wss://goerli.infura.io/ws/v3/92e15d2bbf9b41d286544a680c4b23d0"
)

var ethUSDAggregator common.Address
var btcUSDAggregator common.Address

func init() {
	// ethUSDAggregator is the Aggregator proxied by 0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419.
	ethUSDAggregator = common.HexToAddress("0x00c7A37B03690fb9f41b5C5AF8131735C7275446")
	// btcUSDAggregator is the Aggregator proxied by 0xF4030086522a5bEEa4988F8cA5B36dbC97BeE88c.
	btcUSDAggregator = common.HexToAddress("0xF570deEffF684D964dc3E15E1F9414283E3f7419")
}

func main() {
	eth, err := ethclient.Dial(ethURI)
	if err != nil {
		log.Fatalf("Error dialing %s: %v", ethURI, err)
	}
	defer eth.Close()

	ctx := context.Background()

	go listenAggregator(ctx, eth, ethUSDAggregator, func(l *types.Log, a *abis.AnswerUpdated) {
		log.Printf("Eth (%v): %+v", l.BlockHash, *a)
	})
	go listenAggregator(ctx, eth, btcUSDAggregator, func(l *types.Log, a *abis.AnswerUpdated) {
		log.Printf("BTC (%v): %+v", l.BlockHash, *a)
	})

	select {} // Waits forever.
}

func listenAggregator(ctx context.Context, eth *ethclient.Client, aggregator common.Address,
	cb func(*types.Log, *abis.AnswerUpdated)) {
	query := ethereum.FilterQuery{
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
