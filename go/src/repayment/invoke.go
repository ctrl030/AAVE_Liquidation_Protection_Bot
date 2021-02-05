package repayment

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"text/template"

	"github.com/ethereum/go-ethereum/accounts/abi"
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
	addressT abi.Type
	uintT    abi.Type
)

func init() {
	var err error
	addressT, err = abi.NewType("address", "", nil)
	if err != nil {
		log.Fatalf("Error creating address type: %v", err)
	}
	uintT, err = abi.NewType("uint256", "", nil)
	if err != nil {
		log.Fatalf("Error creating uint type: %v", err)
	}
}

// Execution encapsulates information needed to perform repayment.
type Execution struct {
	loan      *clients.Loan
	cAmount   *big.Int
	signature []byte
	calldata  []byte
}

// NewExecution prepares for repayment. `rAddr` is the address of the RepaymentExecutor contract
// and `signature` is the delegation certificate that approves the Bot to perform repayment.
func NewExecution(ctx context.Context, c *clients.Client, loan *clients.Loan, rAddr common.Address, signature []byte) (*Execution, error) {
	tx, cAmount, err := oneinch.Swap(ctx, c, loan, rAddr)
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
		cAmount:   cAmount,
		signature: signature,
		calldata:  calldata,
	}, nil
}

// Execute executes repayment. This should be called soon after `NewExecution` to avoid slippage.
func (e *Execution) Execute(ctx context.Context, c *clients.Client, r *Repayment) error {
	if err := c.ExecuteAsBot(ctx, "executing repayment",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			args := abi.Arguments{
				abi.Argument{Name: "_aToken", Type: addressT},
				abi.Argument{Name: "_cAsset", Type: addressT},
				abi.Argument{Name: "_cAmount", Type: uintT},
			}
			packed, err := args.Pack(e.loan.AToken, e.loan.Collateral, e.cAmount)
			if err != nil {
				return nil, fmt.Errorf("packing args: %w", err)
			}
			return r.Execute(txr, e.loan.User, e.signature, e.loan.StableDebt, e.loan.VariableDebt, e.loan.Debt, packed, e.calldata)
		}); err != nil {
		return err
	}
	return nil
}
