package repayment

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"clients"
)

var (
	oneInchTemplate = template.Must(template.New("1inch API").Parse(
		"https://api.1inch.exchange/v2.0/swap?" +
			"fromTokenAddress={{.From}}&toTokenAddress={{.To}}" +
			"&amount={{.Amount}}&fromAddress={{.FromAddress}}&slippage={{.Slippage}}" +
			"&disableEstimate=true"))
)

// Execution encapsulates information needed to perform repayment.
type Execution struct {
	loan      *clients.Loan
	signature []byte
	calldata  []byte
}

// NewExecution prepares for repayment. `rAddr` is the address of the RepaymentExecutor contract
// and `signature` is the delegation certificate that approves the Bot to perform repayment.
func NewExecution(ctx context.Context, c *clients.Client, loan *clients.Loan, rAddr common.Address, signature []byte) (*Execution, error) {
	// This balance might be slightly less than the balance when the repayment executes, but only
	// by the amount of interest accumulated over 1 block. Since the 1inch API is off-chain, it's
	// not really possible to get the exact amount at the time of execution.
	balance, err := c.BalanceOf(ctx, loan.AToken, loan.User)
	if err != nil {
		return nil, fmt.Errorf("retrieving user collateral balance: %w", err)
	}

	buf := &strings.Builder{}
	if err := oneInchTemplate.Execute(buf, struct {
		From, To, FromAddress common.Address
		Amount                *big.Int
		Slippage              string
	}{
		loan.Collateral, loan.Debt, rAddr, balance, "1",
	}); err != nil {
		return nil, fmt.Errorf("preparing url: %w", err)
	}

	oneInchClient := http.Client{Timeout: time.Second * 10}

	req, err := http.NewRequest(http.MethodGet, buf.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("preparing 1inch request: %w", err)
	}
	req.Header.Set("User-Agent", "AAVE Liquidation Protection Bot")

	res, err := oneInchClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing 1inch request: %w", err)
	}
	if res.Body != nil {
		defer res.Body.Close()
	}
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading 1inch response: %w", err)
	}

	var parsed interface{}
	if err = json.NewDecoder(bytes.NewReader(content)).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding 1inch response %s: %w", content, err)
	}
	msg, ok := parsed.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("parsed 1inch response wasn't map[string]{interface}: %s", content)
	}
	tx, ok := msg["tx"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("tx wasn't map[string]{interface}: %s", content)
	}
	calldataHex, ok := tx["data"].(string)
	if !ok {
		return nil, fmt.Errorf("1inch response.tx.data wasn't a string: %s", content)
	}
	calldata, err := hex.DecodeString(calldataHex[2:])
	if err != nil {
		return nil, fmt.Errorf("decoding 1inch calldata %s: %w", content, err)
	}
	return &Execution{
		loan:      loan,
		signature: signature,
		calldata:  calldata,
	}, nil
}

// Execute executes repayment.
func (e *Execution) Execute(ctx context.Context, c *clients.Client, r *Repayment) error {
	if err := c.ExecuteAsBot(ctx, "executing repayment",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			return r.Execute(txr, e.loan.User, e.signature, e.loan.Collateral, e.loan.Debt, e.calldata)
		}); err != nil {
		return err
	}
	return nil
}
