// Package wallet contains basic functionality for test-account wallets.
package wallets

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Wallet encapsulates information specific to a wallet.
type Wallet struct {
	Address common.Address
	// TODO(greafilter): this should be moved into a signing service for better security.
	privateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

// NewWallet creates a new wallet.
func NewWallet(privateKey string) (*Wallet, error) {
	ret := &Wallet{}
	var err error
	ret.privateKey, err = crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, fmt.Errorf("converting %s to ECDSA: %w", privateKey, err)
	}
	ret.PublicKey = ret.privateKey.Public().(*ecdsa.PublicKey)
	ret.Address = crypto.PubkeyToAddress(*ret.PublicKey)
	return ret, nil
}

// NewTransactor creates a transactor from this wallet with some default settings.
func (w *Wallet) NewTransactor(ctx context.Context, ct bind.ContractTransactor) (*bind.TransactOpts, error) {
	nonce, err := ct.PendingNonceAt(ctx, w.Address)
	if err != nil {
		return nil, fmt.Errorf("obtaining pending nonce: %w", err)
	}
	gasPrice, err := ct.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting suggested gas price: %w", err)
	}

	tx := bind.NewKeyedTransactor(w.privateKey)
	tx.Nonce = big.NewInt(int64(nonce))
	tx.GasLimit = uint64(9500000)
	tx.GasPrice = gasPrice
	return tx, nil
}

// Execute performs the given transaction and checks its receipt.
func (w *Wallet) Execute(ctx context.Context, eth *ethclient.Client, desc string,
	t func(*bind.TransactOpts) (*types.Transaction, error)) error {
	txr, err := w.NewTransactor(ctx, eth)
	if err != nil {
		return fmt.Errorf("creating transactor for %s: %w", desc, err)
	}
	tx, err := t(txr)
	if err != nil {
		return fmt.Errorf("executing %s: %w", desc, err)
	}
	r, err := eth.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return fmt.Errorf("obtaining receipt for %s: %w", desc, err)
	}
	if r.Status != 1 {
		return fmt.Errorf("%s transaction failed: %v", desc, r)
	}
	return nil
}

// Sign signs a hash using this wallet.
func (w *Wallet) Sign(hash common.Hash) ([]byte, error) {
	signature, err := crypto.Sign(hash.Bytes(), w.privateKey)
	if err != nil {
		return nil, fmt.Errorf("signing with %v: %w", w.Address, err)
	}
	return signature, nil
}
