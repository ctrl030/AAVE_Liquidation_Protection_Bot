// Package scenarios contains code relating to test scenario setup.
package scenarios

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"clients"
	"wallets"
)

// SetupLoan deposits 1 ETH from the given account and borrows 500 Dai.
func SetupLoan(ctx context.Context, c *clients.Client, user *wallets.Wallet) error {
	cAmount := new(big.Int).Mul(big.NewInt(100), big.NewInt(params.Ether))
	if err := c.DepositETH(ctx, user, cAmount); err != nil {
		return fmt.Errorf("setting up loan: %w", err)
	}

	// An Ether is 1e18 Wei and Dai is expressed at the same ratio, so the same constant can be used
	// here.
	dAmount := new(big.Int).Mul(big.NewInt(80000), big.NewInt(params.Ether))
	if err := c.Borrow(ctx, user, c.DaiAddress(), dAmount); err != nil {
		return fmt.Errorf("setting up loan: %w", err)
	}
	return nil
}

// Approve allows the contract to redeem the collateral to pay back the debt.
func ApproveAToken(ctx context.Context, c *clients.Client, user *wallets.Wallet, contract common.Address) error {
	aToken, err := c.AToken(ctx, c.WETH9Address())
	if err != nil {
		return fmt.Errorf("looking up aToken for approval: %w", err)
	}
	if err := c.Execute(ctx, user, "approving protection contract",
		func(txr *bind.TransactOpts) (*types.Transaction, error) {
			return aToken.Approve(txr, contract, big.NewInt(-1))
		}); err != nil {
		return err
	}
	return nil
}
