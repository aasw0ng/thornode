package thorchain

import (
	"encoding/json"
	stdErrors "errors"
	"fmt"

	"github.com/blang/semver"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"gitlab.com/thorchain/thornode/constants"
)

// YggdrasilHandler is to process yggdrasil messages
// When thorchain fund yggdrasil pool , observer should observe two transactions
// 1. outbound tx from asgard vault
// 2. inbound tx to yggdrasil vault
// when yggdrasil pool return fund , observer should observe two transactions as well
// 1. outbound tx from yggdrasil vault
// 2. inbound tx to asgard vault
type YggdrasilHandler struct {
	keeper       Keeper
	txOutStore   VersionedTxOutStore
	validatorMgr VersionedValidatorManager
}

// NewYggdrasilHandler create a new Yggdrasil handler
func NewYggdrasilHandler(keeper Keeper, txOutStore VersionedTxOutStore, validatorMgr VersionedValidatorManager) YggdrasilHandler {
	return YggdrasilHandler{
		keeper:       keeper,
		txOutStore:   txOutStore,
		validatorMgr: validatorMgr,
	}
}

// Run execute the logic in Yggdrasil Handler
func (h YggdrasilHandler) Run(ctx sdk.Context, m sdk.Msg, version semver.Version, constAccessor constants.ConstantValues) sdk.Result {
	msg, ok := m.(MsgYggdrasil)
	if !ok {
		return errInvalidMessage.Result()
	}
	if err := h.validate(ctx, msg, version); err != nil {
		return err.Result()
	}
	return h.handle(ctx, msg, version, constAccessor)
}

func (h YggdrasilHandler) validate(ctx sdk.Context, msg MsgYggdrasil, version semver.Version) sdk.Error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg)
	}
	ctx.Logger().Error(errInvalidVersion.Error())
	return errBadVersion
}

func (h YggdrasilHandler) validateV1(ctx sdk.Context, msg MsgYggdrasil) sdk.Error {
	if err := msg.ValidateBasic(); err != nil {
		ctx.Logger().Error(err.Error())
		return err
	}

	if !isSignedByActiveNodeAccounts(ctx, h.keeper, msg.GetSigners()) {
		return sdk.ErrUnauthorized("not signed by an active node account")
	}
	return nil
}

func (h YggdrasilHandler) handle(ctx sdk.Context, msg MsgYggdrasil, version semver.Version, constAccessor constants.ConstantValues) sdk.Result {
	ctx.Logger().Info("receive MsgYggdrasil", "pubkey", msg.PubKey.String(), "add_funds", msg.AddFunds, "coins", msg.Coins)
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.handleV1(ctx, msg, version)
	} else {
		ctx.Logger().Error(errInvalidVersion.Error())
		return errBadVersion.Result()
	}
}

func (h YggdrasilHandler) handleV1(ctx sdk.Context, msg MsgYggdrasil, version semver.Version) sdk.Result {
	vault, err := h.keeper.GetVault(ctx, msg.PubKey)
	if err != nil && !stdErrors.Is(err, ErrVaultNotFound) {
		ctx.Logger().Error("fail to get yggdrasil", "error", err)
		return sdk.ErrInternal(err.Error()).Result()
	}
	if len(vault.Type) == 0 {
		vault.Status = ActiveVault
		vault.Type = YggdrasilVault
	}

	if msg.AddFunds {
		return h.handleYggdrasilFund(ctx, msg, vault)
	}
	return h.handleYggdrasilReturn(ctx, msg, vault, version)
}

func (h YggdrasilHandler) handleYggdrasilFund(ctx sdk.Context, msg MsgYggdrasil, vault Vault) sdk.Result {
	if vault.Type == AsgardVault {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("asgard_fund_yggdrasil",
				sdk.NewAttribute("pubkey", vault.PubKey.String()),
				sdk.NewAttribute("coins", msg.Coins.String()),
				sdk.NewAttribute("tx", msg.Tx.ID.String())))
	}
	if vault.Type == YggdrasilVault {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("yggdrasil_receive_fund",
				sdk.NewAttribute("pubkey", vault.PubKey.String()),
				sdk.NewAttribute("coins", msg.Coins.String()),
				sdk.NewAttribute("tx", msg.Tx.ID.String())))
		yggFundEvent := NewEventYggdrasil(msg.PubKey, msg.Coins, YggdrasilFund)
		yggBuf, err := json.Marshal(yggFundEvent)
		if err != nil {
			ctx.Logger().Error("fail to marshal yggdrasil fund event", "error", err)
			return sdk.ErrInternal("fail to marshal yggdrasil fund event to json").Result()
		}
		e := NewEvent(yggFundEvent.Type(), ctx.BlockHeight(), msg.Tx, yggBuf, EventSuccess)
		if err := h.keeper.UpsertEvent(ctx, e); err != nil {
			ctx.Logger().Error("fail to save events", "error", err)
			return sdk.ErrInternal("fail to save events").Result()
		}
	}
	// Yggdrasil usually comes from Asgard , Asgard --> Yggdrasil
	// It will be an outbound tx from Asgard pool , and it will be an Inbound tx form Yggdrasil pool
	// incoming fund will be added to Vault as part of ObservedTxInHandler
	// Yggdrasil handler doesn't need to do anything
	return sdk.Result{
		Code:      sdk.CodeOK,
		Codespace: DefaultCodespace,
	}
}

func (h YggdrasilHandler) handleYggdrasilReturn(ctx sdk.Context, msg MsgYggdrasil, vault Vault, version semver.Version) sdk.Result {
	// observe an outbound tx from yggdrasil vault
	if vault.Type == YggdrasilVault {
		asgardVaults, err := h.keeper.GetAsgardVaultsByStatus(ctx, ActiveVault)
		if err != nil {
			ctx.Logger().Error("unable to get asgard vaults", "error", err)
			return sdk.ErrInternal("unable to get asgard vaults").Result()
		}
		isAsgardReceipient, err := asgardVaults.HasAddress(msg.Tx.Chain, msg.Tx.ToAddress)
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("unable to determinate whether %s is an Asgard vault", msg.Tx.ToAddress), "error", err)
			return sdk.ErrInternal("unable to check recipient against active Asgards").Result()
		}

		if !isAsgardReceipient {
			// not sending to asgard , slash the node account
			for _, c := range msg.Tx.Coins {
				if err := slashNodeAccount(ctx, h.keeper, msg.PubKey, c.Asset, c.Amount); err != nil {
					ctx.Logger().Error("fail to slash account for sending fund to a none asgard vault using yggdrasil-", "error", err)
					return sdk.ErrInternal("fail to slash account").Result()
				}
			}
		}

		na, err := h.keeper.GetNodeAccountByPubKey(ctx, msg.PubKey)
		if err != nil {
			ctx.Logger().Error("unable to get node account", "error", err)
			return sdk.ErrInternal(err.Error()).Result()
		}
		if na.Status == NodeActive {
			// node still active , no refund bond
			return sdk.Result{
				Code:      sdk.CodeOK,
				Codespace: DefaultCodespace,
			}
		}
		txOutStore, err := h.txOutStore.GetTxOutStore(h.keeper, version)
		if err != nil {
			ctx.Logger().Error("fail to get txout store", "error", err)
			return errBadVersion.Result()
		}
		if err := refundBond(ctx, msg.Tx, na, h.keeper, txOutStore); err != nil {
			ctx.Logger().Error("fail to refund bond", "error", err)
			return sdk.ErrInternal(err.Error()).Result()
		}
		return sdk.Result{
			Code:      sdk.CodeOK,
			Codespace: DefaultCodespace,
		}
	}

	// when vault.Type is asgard, that means this tx is observed on an asgard pool and it is an inbound tx
	if vault.Type == AsgardVault {
		yggFundEvent := NewEventYggdrasil(msg.PubKey, msg.Coins, YggdrasilReturn)
		yggBuf, err := json.Marshal(yggFundEvent)
		if err != nil {
			ctx.Logger().Error("fail to marshal yggdrasil fund event", "error", err)
			return sdk.ErrInternal("fail to marshal yggdrasil fund event to json").Result()
		}
		e := NewEvent(yggFundEvent.Type(), ctx.BlockHeight(), msg.Tx, yggBuf, EventSuccess)
		if err := h.keeper.UpsertEvent(ctx, e); err != nil {
			ctx.Logger().Error("fail to save events", "error", err)
			return sdk.ErrInternal("fail to save events").Result()
		}
		// Yggdrasil return fund back to Asgard
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("yggdrasil_return",
				sdk.NewAttribute("pubkey", vault.PubKey.String()),
				sdk.NewAttribute("coins", msg.Coins.String()),
				sdk.NewAttribute("tx", msg.Tx.ID.String())))
	}
	return sdk.Result{
		Code:      sdk.CodeOK,
		Codespace: DefaultCodespace,
	}
}
