package wallet

import (
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/chain/address"
)

const (
	defaultWalletType = "bls"
)

type WalletAPI interface {
	WalletNew(context.Context, string) (address.Address, error)
}

type Wallet struct {
	api WalletAPI
}

func New(api WalletAPI) *Wallet {
	return &Wallet{
		api: api,
	}
}

// CreateAddr creates a new address in the Lotus owned single wallet.
// Currently it defaults to BLS wallet type.
// ToDo: Allow choosing wallet type?
func (w *Wallet) CreateAddr(ctx context.Context) (address.Address, error) {
	a, err := w.api.WalletNew(ctx, defaultWalletType)
	if err != nil {
		return address.Undef, fmt.Errorf("error when generating wallet address: %s", err)
	}
	return a, nil
}
