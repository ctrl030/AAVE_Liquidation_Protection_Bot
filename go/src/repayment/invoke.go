package repayment

import (
	"context"
	"encoding/hex"
	"fmt"
	"text/template"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"clients"
	"oneinch"
)

var (
	oneInchSwapTemplate = template.Must(template.New("1inch API").Parse(
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
	tx, err := oneinch.Swap(ctx, c, loan, rAddr)
	if err != nil {
		return nil, fmt.Errorf("preparing swap execution: %w", err)
	}
	calldataHex, ok := tx["data"].(string)
	if !ok {
		return nil, fmt.Errorf("1inch response.tx.data wasn't a string: %v", tx)
	}
	calldata, err := hex.DecodeString(calldataHex[2:])
	if err != nil {
		return nil, fmt.Errorf("decoding 1inch calldata %v: %w", tx, err)
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
