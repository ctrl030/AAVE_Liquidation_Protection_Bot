// Package service serves the frontend.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"

	"clients"
	"delegation"
	"erc20"
	"repayment"
)

type rawRegistration struct {
	User      string `json:"user"`
	Signature string `json:"signature"`
	Threshold string `json:"threshold"`
}

// Deps contains dependencies needed to instantiate the service.
type Deps struct {
	Client *clients.Client

	RepAddr common.Address
	Rep     *repayment.Repayment

	// Root is the root path to statically served files.
	Root string

	// Cert is the unsiged bot delegation certificate that grants permission to the bot to execute
	// repayment.
	Cert *delegation.Certificate
}

// Service holds the service state.
type Service struct {
	client  *clients.Client
	repAddr common.Address
	rep     *repayment.Repayment
	cert    *delegation.Certificate
	router  *gin.Engine
	// users contains the actively monitored loans. It maps from user `common.Address` to
	// `*registration` values.
	users sync.Map
}

// New instantiates a new Service instance.
func New(deps Deps) (*Service, error) {
	s := &Service{
		client:  deps.Client,
		repAddr: deps.RepAddr,
		rep:     deps.Rep,
		cert:    deps.Cert,
		router:  gin.Default(),
	}

	s.router.Use(static.Serve("/", static.LocalFile(deps.Root, true)))
	api := s.router.Group("/api")

	api.GET("/state", func(ctx *gin.Context) {
		hexAddr := ctx.Query("address")
		log.Printf("got state query addr: %v", hexAddr)
		if !common.IsHexAddress(hexAddr) {
			ctx.AbortWithError(400, fmt.Errorf("%s is not a hex address", hexAddr))
			return
		}
		addr := common.HexToAddress(hexAddr)
		loan, err := deps.Client.Loan(ctx, addr)
		if err != nil {
			ctx.AbortWithError(400, err)
			return
		}

		amount, err := loan.Data(ctx, deps.Client)
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
			"contract-address":      deps.RepAddr.String(),
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
		ctx.AsciiJSON(http.StatusOK, deps.Cert.TypedData())
	})

	api.POST("/register", func(ctx *gin.Context) {
		body, err := ioutil.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.AbortWithError(400, fmt.Errorf("getting register body contents: %w", err))
		}
		rr := &rawRegistration{}
		if err := json.Unmarshal(body, rr); err != nil {
			ctx.AbortWithError(400, fmt.Errorf("couldn't parse body %s: %w", body, err))
		}

		reg, err := s.verify(ctx, rr)
		if err != nil {
			ctx.AbortWithError(400, fmt.Errorf("invalid registration: %w", err))
		}

		go s.process(reg)

		ctx.Request.Body = ioutil.NopCloser(bytes.NewReader(body))
		ctx.Status(http.StatusOK)
	})
	return s, nil
}

// Run runs the service. NB: this method does not return.
func (s *Service) Run() {
	s.router.Run(":3000")
}

type registration struct {
	user      common.Address
	signature []byte
	// threshold is the ratio at which to liquidate in units of 1/10000. Its value is uint16, but that
	// type is not supported by atomic.
	threshold int32

	runOnce sync.Once
}

func (s *Service) process(r *registration) {
	v, ok := s.users.Load(r.user)
	if !ok {
		var loaded bool
		v, loaded = s.users.LoadOrStore(r.user, r)
		if loaded {
			// Only the threshold value can change.
			atomic.StoreInt32(&(v.(*registration).threshold), r.threshold)
		}
	}
	ctx := context.Background()
	reg := v.(*registration)
	reg.runOnce.Do(func() {
		loan, err := s.client.Loan(ctx, reg.user)
		if err != nil {
			log.Fatalf("Error retrieving loan for %v: %v", reg.user, err)
		}
		for {
			data, err := loan.Data(ctx, s.client)
			if err != nil {
				// Logs an error message. The query will be retried on the next cycle.
				log.Printf("Error getting loan amounts for %v: %v", reg.user, err)
			} else {
				log.Printf("Collateral = %s %v, Debt = %s %v", loan.CollateralName, data.CollateralAmount, loan.DebtName, data.DebtAmount)
				threshold := uint16(atomic.LoadInt32(&reg.threshold))
				ratio := data.Ratio()
				if ratio >= threshold {
					log.Printf("ratio %d >= threshold %d, repaying", ratio, threshold)
					break
				}
				log.Printf("ratio %d < threshold %d", ratio, threshold)
			}
			time.Sleep(time.Second * 5)
		}
		exec, err := repayment.NewExecution(ctx, s.client, loan, s.repAddr, reg.signature)
		if err != nil {
			log.Fatalf("Error preparing repayment execution: %v", err)
		}
		if err := exec.Execute(ctx, s.client, s.rep); err != nil {
			log.Fatalf("Error executing repayment: %v", err)
		}
	})
}

func (s *Service) verify(ctx context.Context, r *rawRegistration) (*registration, error) {
	if !common.IsHexAddress(r.User) {
		return nil, fmt.Errorf("user was not a hex address: %v", r)
	}
	user := common.HexToAddress(r.User)

	// Verifies the signature is the user-signed delegation certificate for the bot (this process).
	sig, err := hexutil.Decode(r.Signature)
	if err != nil {
		return nil, fmt.Errorf("signature was not hex %v: %w", r, err)
	}
	// crypto.Ecrecover seems to expect 0, 1 instead of 27 or 28.
	switch sig[64] {
	case '\x1b':
		sig[64] = '\x00'
	case '\x1c':
		sig[64] = '\x01'
	}
	rpk, err := crypto.Ecrecover(s.cert.Hash().Bytes(), sig)
	if err != nil {
		return nil, fmt.Errorf("recovering signer %v: %w", r, err)
	}
	pubKey, err := crypto.UnmarshalPubkey(rpk)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling public key %v: %w", pubKey, err)
	}
	if signer := crypto.PubkeyToAddress(*pubKey); signer != user {
		return nil, fmt.Errorf("recovered signer %v didn't match user %v: %v", signer, user, r)
	}

	// Verifies the threshold value.
	thresholdF, err := strconv.ParseFloat(r.Threshold, 64)
	if err != nil {
		return nil, fmt.Errorf("threshold parse error %v: %w", r, err)
	}
	threshold := uint16(thresholdF * 10000)
	if threshold <= 0 {
		return nil, fmt.Errorf("threshold too small: %v", r)
	}
	loan, err := s.client.Loan(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("looking up loan for %v: %w", user, err)
	}
	if threshold >= loan.LiquidationThreshold {
		return nil, fmt.Errorf("threshold %v >= liquidation threshold %v", threshold, loan.LiquidationThreshold)
	}

	return &registration{
		user:      user,
		signature: sig,
		threshold: int32(threshold),
	}, nil
}
