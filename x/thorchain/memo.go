package thorchain

import (
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"

	"gitlab.com/thorchain/thornode/common"
)

// TXTYPE:STATE1:STATE2:STATE3:FINALMEMO

type (
	TxType    uint8
	adminType uint8
)

const (
	txUnknown TxType = iota
	txStake
	txUnstake
	txSwap
	txOutbound
	txAdd
	txGas
	txBond
	txLeave
	txYggdrasilFund
	txYggdrasilReturn
	txReserve
	txRefund
	txMigrate
	txRagnarok
)

var stringToTxTypeMap = map[string]TxType{
	"stake":      txStake,
	"st":         txStake,
	"+":          txStake,
	"withdraw":   txUnstake,
	"unstake":    txUnstake,
	"wd":         txUnstake,
	"-":          txUnstake,
	"swap":       txSwap,
	"s":          txSwap,
	"=":          txSwap,
	"outbound":   txOutbound,
	"add":        txAdd,
	"a":          txAdd,
	"%":          txAdd,
	"gas":        txGas,
	"g":          txGas,
	"$":          txGas,
	"bond":       txBond,
	"leave":      txLeave,
	"yggdrasil+": txYggdrasilFund,
	"yggdrasil-": txYggdrasilReturn,
	"reserve":    txReserve,
	"refund":     txRefund,
	"migrate":    txMigrate,
	"ragnarok":   txRagnarok,
}

var txToStringMap = map[TxType]string{
	txStake:           "stake",
	txUnstake:         "unstake",
	txSwap:            "swap",
	txOutbound:        "outbound",
	txRefund:          "refund",
	txAdd:             "add",
	txGas:             "gas",
	txBond:            "bond",
	txLeave:           "leave",
	txYggdrasilFund:   "yggdrasil+",
	txYggdrasilReturn: "yggdrasil-",
	txReserve:         "reserve",
	txMigrate:         "migrate",
	txRagnarok:        "ragnarok",
}

// converts a string into a txType
func stringToTxType(s string) (TxType, error) {
	// THORNode can support Abbreviated MEMOs , usually it is only one character
	sl := strings.ToLower(s)
	if t, ok := stringToTxTypeMap[sl]; ok {
		return t, nil
	}
	return txUnknown, fmt.Errorf("invalid tx type: %s", s)
}

// Check if two txTypes are the same
func (tx TxType) Equals(tx2 TxType) bool {
	return tx.String() == tx2.String()
}

// Converts a txType into a string
func (tx TxType) String() string {
	return txToStringMap[tx]
}

type Memo interface {
	IsType(tx TxType) bool
	GetType() TxType

	String() string
	GetAsset() common.Asset
	GetAmount() string
	GetDestination() common.Address
	GetSlipLimit() sdk.Uint
	GetKey() string
	GetValue() string
	GetTxID() common.TxID
	GetNodeAddress() sdk.AccAddress
	GetBlockHeight() int64
}

type MemoBase struct {
	TxType TxType
	Asset  common.Asset
}

type CreateMemo struct {
	MemoBase
}

type GasMemo struct {
	MemoBase
}

type AddMemo struct {
	MemoBase
}

type StakeMemo struct {
	MemoBase
	RuneAmount  string
	AssetAmount string
	Address     common.Address
}

type UnstakeMemo struct {
	MemoBase
	Amount string
}

type SwapMemo struct {
	MemoBase
	Destination common.Address
	SlipLimit   sdk.Uint
}

type AdminMemo struct {
	MemoBase
	Key   string
	Value string
	Type  adminType
}

type OutboundMemo struct {
	MemoBase
	TxID common.TxID
}

type RefundMemo struct {
	MemoBase
	TxID common.TxID
}

type BondMemo struct {
	MemoBase
	NodeAddress sdk.AccAddress
}

type LeaveMemo struct {
	MemoBase
}

type YggdrasilFundMemo struct {
	MemoBase
}

type YggdrasilReturnMemo struct {
	MemoBase
}

type ReserveMemo struct {
	MemoBase
}

type MigrateMemo struct {
	MemoBase
	BlockHeight int64
}

type RagnarokMemo struct {
	MemoBase
	BlockHeight int64
}

func NewRagnarokMemo(blockHeight int64) RagnarokMemo {
	return RagnarokMemo{
		BlockHeight: blockHeight,
	}
}

func NewMigrateMemo(blockHeight int64) MigrateMemo {
	return MigrateMemo{
		BlockHeight: blockHeight,
	}
}

func NewOutboundMemo(txID common.TxID) OutboundMemo {
	return OutboundMemo{
		TxID: txID,
	}
}

// NewRefundMemo create a new RefundMemo
func NewRefundMemo(txID common.TxID) RefundMemo {
	return RefundMemo{
		TxID: txID,
	}
}

func ParseMemo(memo string) (Memo, error) {
	var err error
	noMemo := MemoBase{}
	if len(memo) == 0 {
		return noMemo, fmt.Errorf("memo can't be empty")
	}
	parts := strings.Split(memo, ":")
	tx, err := stringToTxType(parts[0])
	if err != nil {
		return noMemo, err
	}

	// list of memo types that do not contain an asset in their memo
	noAssetMemos := []TxType{
		txGas, txOutbound, txBond, txLeave, txRefund,
		txYggdrasilFund, txYggdrasilReturn, txReserve,
		txMigrate, txRagnarok,
	}
	hasAsset := true
	for _, memoType := range noAssetMemos {
		if tx == memoType {
			hasAsset = false
		}
	}

	var asset common.Asset
	if hasAsset {
		if len(parts) < 2 {
			return noMemo, fmt.Errorf("cannot parse given memo: length %d", len(parts))
		}
		var err error
		asset, err = common.NewAsset(parts[1])
		if err != nil {
			return noMemo, err
		}
	}

	switch tx {
	case txGas:
		return GasMemo{
			MemoBase: MemoBase{TxType: txGas},
		}, nil
	case txLeave:
		return LeaveMemo{
			MemoBase: MemoBase{TxType: txLeave},
		}, nil
	case txAdd:
		return AddMemo{
			MemoBase: MemoBase{TxType: txAdd, Asset: asset},
		}, nil

	case txStake:
		var addr common.Address
		if !asset.Chain.IsBNB() {
			if len(parts) < 3 {
				// cannot stake into a non BNB-based pool when THORNode don't have an
				// associated address
				return noMemo, fmt.Errorf("invalid stake. Cannot stake to a non BNB-based pool without providing an associated address")
			}
			addr, err = common.NewAddress(parts[2])
			if err != nil {
				return noMemo, err
			}
		}
		return StakeMemo{
			MemoBase: MemoBase{TxType: txStake, Asset: asset},
			Address:  addr,
		}, nil

	case txUnstake:
		if len(parts) < 2 {
			return noMemo, fmt.Errorf("invalid unstake memo")
		}
		var withdrawAmount string
		if len(parts) > 2 {
			withdrawAmount = parts[2]
			wa, err := sdk.ParseUint(withdrawAmount)
			if err != nil {
				return noMemo, err
			}
			if !wa.GT(sdk.ZeroUint()) || wa.GT(sdk.NewUint(MaxUnstakeBasisPoints)) {
				return noMemo, fmt.Errorf("withdraw amount :%s is invalid", withdrawAmount)
			}
		}
		return UnstakeMemo{
			MemoBase: MemoBase{TxType: txUnstake, Asset: asset},
			Amount:   withdrawAmount,
		}, err

	case txSwap:
		if len(parts) < 2 {
			return noMemo, fmt.Errorf("missing swap parameters: memo should in SWAP:SYMBOLXX-XXX:DESTADDR:TRADE-TARGET format")
		}
		// DESTADDR can be empty , if it is empty , it will swap to the sender address
		destination := common.NoAddress
		if len(parts) > 2 {
			if len(parts[2]) > 0 {
				destination, err = common.NewAddress(parts[2])
				if err != nil {
					return noMemo, err
				}
			}
		}
		// price limit can be empty , when it is empty , there is no price protection
		slip := sdk.ZeroUint()
		if len(parts) > 3 && len(parts[3]) > 0 {
			amount, err := sdk.ParseUint(parts[3])
			if err != nil {
				return noMemo, fmt.Errorf("swap price limit:%s is invalid", parts[3])
			}

			slip = amount
		}
		return SwapMemo{
			MemoBase:    MemoBase{TxType: txSwap, Asset: asset},
			Destination: destination,
			SlipLimit:   slip,
		}, err

	case txOutbound:
		if len(parts) < 2 {
			return noMemo, fmt.Errorf("not enough parameters")
		}
		txID, err := common.NewTxID(parts[1])
		return NewOutboundMemo(txID), err
	case txRefund:
		if len(parts) < 2 {
			return noMemo, fmt.Errorf("not enough parameters")
		}
		txID, err := common.NewTxID(parts[1])
		return NewRefundMemo(txID), err
	case txBond:
		if len(parts) < 2 {
			return noMemo, fmt.Errorf("not enough parameters")
		}
		addr, err := sdk.AccAddressFromBech32(parts[1])
		if err != nil {
			return noMemo, errors.Wrapf(err, "%s is an invalid thorchain address", parts[1])
		}
		return BondMemo{
			MemoBase:    MemoBase{TxType: txBond},
			NodeAddress: addr,
		}, nil
	case txYggdrasilFund:
		return YggdrasilFundMemo{
			MemoBase: MemoBase{TxType: txYggdrasilFund},
		}, nil
	case txYggdrasilReturn:
		return YggdrasilReturnMemo{
			MemoBase: MemoBase{TxType: txYggdrasilReturn},
		}, nil
	case txReserve:
		return ReserveMemo{
			MemoBase: MemoBase{TxType: txYggdrasilReturn},
		}, nil
	case txMigrate:
		if len(parts) < 2 {
			return noMemo, errors.New("not enough parameters")
		}
		blockHeight, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return noMemo, fmt.Errorf("fail to convert (%s) to a valid block height: %w", parts[1], err)
		}
		return MigrateMemo{
			MemoBase:    MemoBase{TxType: txMigrate},
			BlockHeight: blockHeight,
		}, nil
	case txRagnarok:
		if len(parts) < 2 {
			return noMemo, errors.New("not enough parameters")
		}
		blockHeight, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return noMemo, fmt.Errorf("fail to convert (%s) to a valid block height: %w", parts[1], err)
		}
		return RagnarokMemo{
			MemoBase:    MemoBase{TxType: txRagnarok},
			BlockHeight: blockHeight,
		}, nil
	default:
		return noMemo, fmt.Errorf("TxType not supported: %s", tx.String())
	}
}

// Base Functions
func (m MemoBase) String() string                 { return "" }
func (m MemoBase) GetType() TxType                { return m.TxType }
func (m MemoBase) IsType(tx TxType) bool          { return m.TxType.Equals(tx) }
func (m MemoBase) GetAsset() common.Asset         { return m.Asset }
func (m MemoBase) GetAmount() string              { return "" }
func (m MemoBase) GetDestination() common.Address { return "" }
func (m MemoBase) GetSlipLimit() sdk.Uint         { return sdk.ZeroUint() }
func (m MemoBase) GetKey() string                 { return "" }
func (m MemoBase) GetValue() string               { return "" }
func (m MemoBase) GetTxID() common.TxID           { return "" }
func (m MemoBase) GetNodeAddress() sdk.AccAddress { return sdk.AccAddress{} }
func (m MemoBase) GetBlockHeight() int64          { return 0 }

// Transaction Specific Functions
func (m UnstakeMemo) GetAmount() string            { return m.Amount }
func (m SwapMemo) GetDestination() common.Address  { return m.Destination }
func (m SwapMemo) GetSlipLimit() sdk.Uint          { return m.SlipLimit }
func (m AdminMemo) GetKey() string                 { return m.Key }
func (m AdminMemo) GetValue() string               { return m.Value }
func (m BondMemo) GetNodeAddress() sdk.AccAddress  { return m.NodeAddress }
func (m StakeMemo) GetDestination() common.Address { return m.Address }
func (m OutboundMemo) GetTxID() common.TxID        { return m.TxID }
func (m OutboundMemo) String() string {
	return fmt.Sprintf("OUTBOUND:%s", m.TxID.String())
}

// GetTxID return the relevant tx id in refund memo
func (m RefundMemo) GetTxID() common.TxID { return m.TxID }

// String implement fmt.Stringer
func (m RefundMemo) String() string {
	return fmt.Sprintf("REFUND:%s", m.TxID.String())
}

func (m MigrateMemo) String() string {
	return fmt.Sprintf("MIGRATE:%d", m.BlockHeight)
}

func (m MigrateMemo) GetBlockHeight() int64 {
	return m.BlockHeight
}

func (m RagnarokMemo) String() string {
	return fmt.Sprintf("RAGNAROK:%d", m.BlockHeight)
}

func (m RagnarokMemo) GetBlockHeight() int64 {
	return m.BlockHeight
}
