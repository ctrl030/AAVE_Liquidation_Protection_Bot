// Package service serves the frontend.
package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"

	"clients"
	"delegation"
	"erc20"
)

// Registration contains registration data passed from the client.
type Registration struct {
	Signature string `json:"signature"`
	Threshold string `json:"threshold"`
}

// Serve serves the frontend rooted at `path`. It provides the repayment address, `repAddr`, to the
// client so that it may approve withdrawal of ATokens and it provides the JSON serialized
// certificate, `cert`, to the client for signing. Signatures are notified via the callback `cb`.
func Serve(client *clients.Client, path string, repAddr common.Address, cert *delegation.Certificate, cb func(*Registration)) {
	router := gin.Default()
	router.Use(static.Serve("/", static.LocalFile(path, true)))

	api := router.Group("/api")
	api.GET("/state", func(ctx *gin.Context) {
		hexAddr := ctx.Query("address")
		log.Printf("got state query addr: %v", hexAddr)
		if !common.IsHexAddress(hexAddr) {
			ctx.AbortWithError(400, fmt.Errorf("%s is not a hex address", hexAddr))
			return
		}
		addr := common.HexToAddress(hexAddr)
		loan, err := client.Loan(ctx, addr)
		if err != nil {
			ctx.AbortWithError(400, err)
			return
		}
		amount, err := loan.Data(ctx, client)
		if err != nil {
			ctx.AbortWithError(400, err)
			return
		}

		threshold := float64(loan.LiquidationThreshold) / float64(10000)
		ctx.JSON(http.StatusOK, gin.H{
			"collateral-name":       loan.CollateralName,
			"collateral-address":    loan.Collateral.String(),
			"collateral-amount":     amount.CollateralAmount.String(),
			"a-token-address":       loan.AToken.String(),
			"debt-name":             loan.DebtName,
			"debt-address":          loan.Debt.String(),
			"debt-amount":           amount.DebtAmount.String(),
			"current-ratio":         amount.CurrentRatio.FloatString(10),
			"liquidation-threshold": fmt.Sprintf("%.4f", threshold),
			"contract-address":      repAddr.String(),
		})
	})

	api.GET("/abi", func(ctx *gin.Context) {
		switch name := ctx.Query("name"); name {
		case "erc20":
			ctx.DataFromReader(http.StatusOK, int64(len(erc20.Erc20ABI)), gin.MIMEJSON,
				strings.NewReader(erc20.Erc20ABI), nil)
		default:
			ctx.AbortWithError(400, fmt.Errorf("unknown ABI %s", name))
			return
		}
	})

	api.GET("/cert", func(ctx *gin.Context) {
		ctx.AsciiJSON(http.StatusOK, cert.TypedData())
	})

	api.POST("/register", func(ctx *gin.Context) {
		body, err := ioutil.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.AbortWithError(400, fmt.Errorf("getting register body contents: %w", err))
		}
		reg := &Registration{}
		if err := json.Unmarshal(body, reg); err != nil {
			ctx.AbortWithError(400, fmt.Errorf("couldn't parse body %s: %w", body, err))
		}
		go cb(reg) // Callback runs in a goroutine in case it blocks.
		ctx.Request.Body = ioutil.NopCloser(bytes.NewReader(body))
		ctx.Status(http.StatusOK)
	})

	router.Run(":3000")
}
