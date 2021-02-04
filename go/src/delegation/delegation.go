// Package delegation creates certificates to attest approval.
//
// For explanation, see:
// https://medium.com/alpineintel/issuing-and-verifying-eip-712-challenges-with-go-32635ca78aaf
package delegation

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core"
)

const (
	appName = "AAVE Liquidation Protection Bot"
	salt    = "SU%N6gmumvj.A{@B,SdWXtVgg(Bof9SA"
)

type Certificate struct {
	delegate common.Address
	data     *core.TypedData
}

// New generates a new certificate for the given delegate.
func New(delegate common.Address) (*Certificate, error) {
	data := &core.TypedData{
		Types: core.Types{
			"Delegate": []core.Type{
				{Name: "delegate", Type: "address"},
			},
			"EIP712Domain": []core.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "salt", Type: "string"},
			},
		},
		PrimaryType: "Delegate",
		Domain: core.TypedDataDomain{
			Name:    appName,
			Version: "1",
			ChainId: math.NewHexOrDecimal256(1337), // Mainnet
			Salt:    salt,
		},
		Message: core.TypedDataMessage{
			"delegate": delegate.Hex(),
		},
	}
	return &Certificate{
		delegate: delegate,
		data:     data,
	}, nil
}

// TypedData returns the typed data needing a signature.
func (c *Certificate) TypedData() *core.TypedData {
	return c.data
}

// Hash returns the EIP-712 hash for the delegate.
func (c *Certificate) Hash() (common.Hash, error) {
	domainHash, err := c.data.HashStruct("EIP712Domain", c.data.Domain.Map())
	if err != nil {
		return common.BytesToHash(nil), fmt.Errorf("hashing domain for %v: %w", c.delegate, err)
	}
	dataHash, err := c.data.HashStruct(c.data.PrimaryType, c.data.Message)
	if err != nil {
		return common.BytesToHash(nil), fmt.Errorf("hashing message for %v: %w", c.delegate, err)
	}
	var buf bytes.Buffer
	buf.WriteByte('\x19')
	buf.WriteByte('\x01')
	buf.Write(domainHash)
	buf.Write(dataHash)

	return crypto.Keccak256Hash(buf.Bytes()), nil
}
