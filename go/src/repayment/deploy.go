package repayment

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"clients"
)

// Deploy deploys the contract using the bot account.
func Deploy(ctx context.Context, c *clients.Client) (*Repayment, common.Address, error) {
	var addr common.Address
	var r *Repayment
	if err := c.ExecuteAsBot(ctx, "deploying protection contract",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			var tx *types.Transaction
			var err error
			// Gas limit estimation fails for deployment. It crashes trying to resolve a transaction
			// recipient, but there's none for deployment.
			txr.GasLimit = uint64(9500000)
			addr, tx, r, err = DeployRepayment(txr, c.ETH())
			return tx, err
		}); err != nil {
		return nil, common.BytesToAddress(nil), err
	}
	return r, addr, nil
}
