package mocks

import (
	"errors"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"gitlab.com/thorchain/bepswap/common"

	"gitlab.com/thorchain/bepswap/statechain/x/swapservice/types"
)

// MockPoolStorage implements PoolStorage interface, thus we can mock the error cases
type MockPoolStorage struct {
}

func (mps MockPoolStorage) PoolExist(ctx sdk.Context, ticker common.Ticker) bool {
	if ticker.Equals(common.Ticker("NOTEXIST")) {
		return false
	}
	return true
}

func (mps MockPoolStorage) GetPool(ctx sdk.Context, ticker common.Ticker) types.Pool {
	if ticker.Equals(common.Ticker("NOTEXIST")) {
		return types.Pool{}
	} else {
		return types.Pool{
			BalanceRune:  sdk.NewUint(100).MulUint64(types.One),
			BalanceToken: sdk.NewUint(100).MulUint64(types.One),
			PoolUnits:    sdk.NewUint(100).MulUint64(types.One),
			Status:       types.Enabled,
			Ticker:       ticker,
		}
	}
}

func (mps MockPoolStorage) SetPool(ctx sdk.Context, ps types.Pool) {}

func (mps MockPoolStorage) GetStakerPool(ctx sdk.Context, stakerID common.BnbAddress) (types.StakerPool, error) {
	if strings.EqualFold(stakerID.String(), "NOTEXISTSTAKER") {
		return types.StakerPool{}, errors.New("you asked for it")
	}
	return types.NewStakerPool(stakerID), nil
}

func (mps MockPoolStorage) SetStakerPool(ctx sdk.Context, stakerID common.BnbAddress, sp types.StakerPool) {

}

func (mps MockPoolStorage) GetPoolStaker(ctx sdk.Context, ticker common.Ticker) (types.PoolStaker, error) {
	if ticker.Equals(common.Ticker("NOTEXISTSTICKER")) {
		return types.PoolStaker{}, errors.New("you asked for it")
	}
	return types.NewPoolStaker(ticker, sdk.NewUint(100)), nil
}

func (mps MockPoolStorage) SetPoolStaker(ctx sdk.Context, ticker common.Ticker, ps types.PoolStaker) {}

func (mps MockPoolStorage) GetAdminConfigValue(ctx sdk.Context, key types.AdminConfigKey, bnb common.BnbAddress) (string, error) {
	return "FOOBAR", nil
}
func (mps MockPoolStorage) GetAdminConfigStakerAmtInterval(ctx sdk.Context, bnb common.BnbAddress) common.Amount {
	return common.NewAmountFromFloat(100)
}

func (mps MockPoolStorage) AddIncompleteEvents(ctx sdk.Context, event types.Event) {}
