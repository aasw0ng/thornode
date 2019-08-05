package swapservice

import (
	"github.com/jpthor/cosmos-swap/x/swapservice/types"
)

const (
	ModuleName       = types.ModuleName
	RouterKey        = types.RouterKey
	StoreKey         = types.StoreKey
	DefaultCodespace = types.DefaultCodespace
)

var (
	NewPoolStruct   = types.NewPoolStruct
	NewMsgSetTxHash = types.NewMsgSetTxHash
	ModuleCdc       = types.ModuleCdc
	RegisterCodec   = types.RegisterCodec
)

type (
	MsgSetPoolData      = types.MsgSetPoolData
	MsgSetStakeData     = types.MsgSetStakeData
	MsgSetTxHash        = types.MsgSetTxHash
	MsgSwap             = types.MsgSwap
	QueryResPoolStructs = types.QueryResPoolStructs
	TxHash              = types.TxHash
	PoolStruct          = types.PoolStruct
)
