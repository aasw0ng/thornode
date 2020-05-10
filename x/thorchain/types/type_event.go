package types

import (
	"encoding/json"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"gitlab.com/thorchain/thornode/common"
)

// Events bt
type Event struct {
	ID     int64           `json:"id"`
	Height int64           `json:"height"`
	Type   string          `json:"type"`
	InTx   common.Tx       `json:"in_tx"`
	OutTxs common.Txs      `json:"out_txs"`
	Fee    common.Fee      `json:"fee"`
	Event  json.RawMessage `json:"event"`
	Status EventStatus     `json:"status"`
}

const (
	SwapEventType    = `swap`
	StakeEventType   = `stake`
	UnstakeEventType = `unstake`
	AddEventType     = `add`
	PoolEventType    = `pool`
	RewardEventType  = `rewards`
	RefundEventType  = `refund`
	BondEventType    = `bond`
	GasEventType     = `gas`
	ReserveEventType = `reserve`
	SlashEventType   = `slash`
	ErrataEventType  = `errata`
)

type PoolMod struct {
	Asset    common.Asset `json:"asset"`
	RuneAmt  sdk.Uint     `json:"rune_amt"`
	RuneAdd  bool         `json:"rune_add"`
	AssetAmt sdk.Uint     `json:"asset_amt"`
	AssetAdd bool         `json:"asset_add"`
}

type PoolMods []PoolMod

func NewPoolMod(asset common.Asset, runeAmt sdk.Uint, runeAdd bool, assetAmt sdk.Uint, assetAdd bool) PoolMod {
	return PoolMod{
		Asset:    asset,
		RuneAmt:  runeAmt,
		RuneAdd:  runeAdd,
		AssetAmt: assetAmt,
		AssetAdd: assetAdd,
	}
}

// NewEvent create a new  event
func NewEvent(typ string, ht int64, inTx common.Tx, evt json.RawMessage, status EventStatus) Event {
	return Event{
		Height: ht,
		Type:   typ,
		InTx:   inTx,
		Event:  evt,
		Status: status,
		Fee: common.Fee{
			Coins:      common.Coins{},
			PoolDeduct: sdk.ZeroUint(),
		},
	}
}

// Empty determinate whether the event is empty
func (evt Event) Empty() bool {
	return evt.InTx.ID.IsEmpty()
}

// Events is a slice of events
type Events []Event

// PopByInHash Pops an event out of the event list by hash ID
func (evts Events) PopByInHash(txID common.TxID) (found, events Events) {
	for _, evt := range evts {
		if evt.InTx.ID.Equals(txID) {
			found = append(found, evt)
		} else {
			events = append(events, evt)
		}
	}
	return
}

// EventSwap event for swap action
type EventSwap struct {
	Pool               common.Asset `json:"pool"`
	PriceTarget        sdk.Uint     `json:"price_target"`
	TradeSlip          sdk.Uint     `json:"trade_slip"`
	LiquidityFee       sdk.Uint     `json:"liquidity_fee"`
	LiquidityFeeInRune sdk.Uint     `json:"liquidity_fee_in_rune"`
	//  the following two field is trying to make events change backward compatible
	// very soon we don't need to save this event to key value store anymore , it will be removed then
	InTx   common.Tx `json:"-"` // this is the Tx that cause the swap to happen, it is a double swap , then the txid will be blank
	OutTxs common.Tx `json:"-"` // this field will need temporary
}

// NewEventSwap create a new swap event
func NewEventSwap(pool common.Asset, priceTarget, fee, tradeSlip, liquidityFeeInRune sdk.Uint, inTx common.Tx) EventSwap {
	return EventSwap{
		Pool:               pool,
		PriceTarget:        priceTarget,
		TradeSlip:          tradeSlip,
		LiquidityFee:       fee,
		LiquidityFeeInRune: liquidityFeeInRune,
		InTx:               inTx,
	}
}

// Type return a string that represent the type, it should not duplicated with other event
func (e EventSwap) Type() string {
	return SwapEventType
}

func (e EventSwap) Events() (sdk.Events, error) {
	evt := sdk.NewEvent(e.Type(),
		sdk.NewAttribute("pool", e.Pool.String()),
		sdk.NewAttribute("price_target", e.PriceTarget.String()),
		sdk.NewAttribute("trade_slip", e.TradeSlip.String()),
		sdk.NewAttribute("liquidity_fee", e.LiquidityFee.String()),
		sdk.NewAttribute("liquidity_fee_in_rune", e.LiquidityFeeInRune.String()),
	)
	evt = evt.AppendAttributes(e.InTx.ToAttributes()...)
	return sdk.Events{evt}, nil
}

// EventStake stake event
type EventStake struct {
	Pool       common.Asset `json:"pool"`
	StakeUnits sdk.Uint     `json:"stake_units"`
	TxIn       common.Tx    `json:"-"`
}

// NewEventStake create a new stake event
func NewEventStake(pool common.Asset, su sdk.Uint, txIn common.Tx) EventStake {
	return EventStake{
		Pool:       pool,
		StakeUnits: su,
		TxIn:       txIn,
	}
}

// Type return the event type
func (e EventStake) Type() string {
	return StakeEventType
}

func (e EventStake) Events() (sdk.Events, error) {
	evt := sdk.NewEvent(e.Type(),
		sdk.NewAttribute("pool", e.Pool.String()),
		sdk.NewAttribute("stake_units", e.StakeUnits.String()))
	evt = evt.AppendAttributes(e.TxIn.ToAttributes()...)
	return sdk.Events{
		evt,
	}, nil
}

// EventUnstake represent unstake
type EventUnstake struct {
	Pool        common.Asset `json:"pool"`
	StakeUnits  sdk.Uint     `json:"stake_units"`
	BasisPoints int64        `json:"basis_points"` // 1 ==> 10,0000
	Asymmetry   sdk.Dec      `json:"asymmetry"`    // -1.0 <==> 1.0
	InTx        common.Tx    `json:"-"`
}

// NewEventUnstake create a new unstake event
func NewEventUnstake(pool common.Asset, su sdk.Uint, basisPts int64, asym sdk.Dec, inTx common.Tx) EventUnstake {
	return EventUnstake{
		Pool:        pool,
		StakeUnits:  su,
		BasisPoints: basisPts,
		Asymmetry:   asym,
		InTx:        inTx,
	}
}

// Type return the unstake event type
func (e EventUnstake) Type() string {
	return UnstakeEventType
}

// Events
func (e EventUnstake) Events() (sdk.Events, error) {
	evt := sdk.NewEvent(e.Type(),
		sdk.NewAttribute("pool", e.Pool.String()),
		sdk.NewAttribute("stake_units", e.StakeUnits.String()),
		sdk.NewAttribute("basis_points", strconv.FormatInt(e.BasisPoints, 10)),
		sdk.NewAttribute("asymmetry", e.Asymmetry.String()))
	evt = evt.AppendAttributes(e.InTx.ToAttributes()...)
	return sdk.Events{evt}, nil
}

// EventAdd represent add operation
type EventAdd struct {
	Pool common.Asset `json:"pool"`
	InTx common.Tx    `json:"-"`
}

// NewEventAdd create a new add event
func NewEventAdd(pool common.Asset, inTx common.Tx) EventAdd {
	return EventAdd{
		Pool: pool,
		InTx: inTx,
	}
}

// Type return add event type
func (e EventAdd) Type() string {
	return AddEventType
}

// Events get all events
func (e EventAdd) Events() (sdk.Events, error) {
	evt := sdk.NewEvent(e.Type(),
		sdk.NewAttribute("pool", e.Pool.String()))
	evt = evt.AppendAttributes(e.InTx.ToAttributes()...)
	return sdk.Events{evt}, nil
}

// EventPool represent pool change event
type EventPool struct {
	Pool   common.Asset `json:"pool"`
	Status PoolStatus   `json:"status"`
}

// NewEventPool create a new pool change event
func NewEventPool(pool common.Asset, status PoolStatus) EventPool {
	return EventPool{
		Pool:   pool,
		Status: status,
	}
}

// Type return pool event type
func (e EventPool) Type() string {
	return PoolEventType
}

// Events provide an instance of sdk.Events
func (e EventPool) Events() (sdk.Events, error) {
	return sdk.Events{
		sdk.NewEvent(e.Type(),
			sdk.NewAttribute("pool", e.Pool.String()),
			sdk.NewAttribute("pool_status", e.Status.String())),
	}, nil
}

// PoolAmt pool asset amount
type PoolAmt struct {
	Asset  common.Asset `json:"asset"`
	Amount int64        `json:"amount"`
}

// EventRewards reward event
type EventRewards struct {
	BondReward  sdk.Uint  `json:"bond_reward"`
	PoolRewards []PoolAmt `json:"pool_rewards"`
}

// NewEventRewards create a new reward event
func NewEventRewards(bondReward sdk.Uint, poolRewards []PoolAmt) EventRewards {
	return EventRewards{
		BondReward:  bondReward,
		PoolRewards: poolRewards,
	}
}

// Type return reward event type
func (e EventRewards) Type() string {
	return RewardEventType
}

func (e EventRewards) Events() (sdk.Events, error) {
	evt := sdk.NewEvent(e.Type(),
		sdk.NewAttribute("bond_reward", e.BondReward.String()),
	)
	for _, item := range e.PoolRewards {
		evt = evt.AppendAttributes(sdk.NewAttribute(item.Asset.String(), strconv.FormatInt(item.Amount, 10)))
	}
	return sdk.Events{evt}, nil
}

// NewEventRefund create a new EventRefund
func NewEventRefund(code sdk.CodeType, reason string) EventRefund {
	return EventRefund{
		Code:   code,
		Reason: reason,
	}
}

// EventRefund represent a refund activity , and contains the reason why it get refund
type EventRefund struct {
	Code   sdk.CodeType `json:"code"`
	Reason string       `json:"reason"`
}

// Type return reward event type
func (e EventRefund) Type() string {
	return RefundEventType
}

type BondType string

const (
	BondPaid     BondType = `bond_paid`
	BondReturned BondType = `bond_returned`
)

// EventBond bond paid or returned event
type EventBond struct {
	Amount   sdk.Uint `json:"amount"`
	BondType BondType `json:"bond_type"`
}

// Type return bond event Type
func (e EventBond) Type() string {
	return BondEventType
}

// NewEventBond create a new Bond Events
func NewEventBond(amount sdk.Uint, bondType BondType) EventBond {
	return EventBond{
		Amount:   amount,
		BondType: bondType,
	}
}

type GasType string

type GasPool struct {
	Asset    common.Asset `json:"asset"`
	AssetAmt sdk.Uint     `json:"asset_amt"`
	RuneAmt  sdk.Uint     `json:"rune_amt"`
	Count    int64        `json:"transaction_count"`
}

// EventGas represent the events happened in thorchain related to Gas
type EventGas struct {
	Pools []GasPool `json:"pools"`
}

// NewEventGas create a new EventGas instance
func NewEventGas() *EventGas {
	return &EventGas{
		Pools: []GasPool{},
	}
}

// UpsertGasPool update the Gas Pools hold by EventGas instance
// if the given gasPool already exist, then it merge the gasPool with internal one , otherwise add it to the list
func (e *EventGas) UpsertGasPool(pool GasPool) {
	for i, p := range e.Pools {
		if p.Asset == pool.Asset {
			e.Pools[i].RuneAmt = p.RuneAmt.Add(pool.RuneAmt)
			e.Pools[i].AssetAmt = p.AssetAmt.Add(pool.AssetAmt)
			return
		}
	}
	e.Pools = append(e.Pools, pool)
}

// Type return event type
func (e *EventGas) Type() string {
	return GasEventType
}

func (e *EventGas) Events() (sdk.Events, error) {
	events := make(sdk.Events, 0, len(e.Pools))
	for _, item := range e.Pools {
		evt := sdk.NewEvent(e.Type(),
			sdk.NewAttribute("asset", item.Asset.String()),
			sdk.NewAttribute("asset_amt", item.AssetAmt.String()),
			sdk.NewAttribute("rune_amt", item.RuneAmt.String()),
			sdk.NewAttribute("transaction_count", strconv.FormatInt(item.Count, 10)))
		events = append(events, evt)
	}
	return events, nil
}

// EventReserve Reserve event type
type EventReserve struct {
	ReserveContributor ReserveContributor `json:"reserve_contributor"`
	InTx               common.Tx          `json:"-"`
}

// NewEventReserve create a new instance of EventReserve
func NewEventReserve(contributor ReserveContributor, inTx common.Tx) EventReserve {
	return EventReserve{
		ReserveContributor: contributor,
		InTx:               inTx,
	}
}

func (e EventReserve) Type() string {
	return ReserveEventType
}

func (e EventReserve) Events() (sdk.Events, error) {
	evt := sdk.NewEvent(e.Type(),
		sdk.NewAttribute("contributor_address", e.ReserveContributor.Address.String()),
		sdk.NewAttribute("amount", e.ReserveContributor.Amount.String()),
	)
	evt = evt.AppendAttributes(e.InTx.ToAttributes()...)
	return sdk.Events{
		evt,
	}, nil
}

// EventSlash represent a change in pool balance which caused by slash a node account
type EventSlash struct {
	Pool        common.Asset `json:"pool"`
	SlashAmount []PoolAmt    `json:"slash_amount"`
}

func NewEventSlash(pool common.Asset, slashAmount []PoolAmt) EventSlash {
	return EventSlash{
		Pool:        pool,
		SlashAmount: slashAmount,
	}
}

// Type return slash event type
func (e EventSlash) Type() string {
	return SlashEventType
}

// EventErrata represent a change in pool balance which caused by an errata transaction
type EventErrata struct {
	TxID  common.TxID `json:"tx_id"`
	Pools PoolMods    `json:"pools"`
}

func NewEventErrata(txID common.TxID, pools PoolMods) EventErrata {
	return EventErrata{
		TxID:  txID,
		Pools: pools,
	}
}

// Type return slash event type
func (e EventErrata) Type() string {
	return ErrataEventType
}

// Events
func (e EventErrata) Events() (sdk.Events, error) {
	events := make(sdk.Events, 0, len(e.Pools))
	for _, item := range e.Pools {
		evt := sdk.NewEvent(e.Type(),
			sdk.NewAttribute("in_tx_id", e.TxID.String()),
			sdk.NewAttribute("asset", item.Asset.String()),
			sdk.NewAttribute("rune_amt", item.RuneAmt.String()),
			sdk.NewAttribute("rune_add", strconv.FormatBool(item.RuneAdd)),
			sdk.NewAttribute("asset_amt", item.AssetAmt.String()),
			sdk.NewAttribute("asset_add", strconv.FormatBool(item.AssetAdd)))
		events = append(events, evt)
	}
	return events, nil
}
