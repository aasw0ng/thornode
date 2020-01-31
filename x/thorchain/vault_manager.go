package thorchain

import (
	"errors"
	"fmt"

	"github.com/blang/semver"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/constants"
)

// const values used to emit events
const (
	EventTypeActiveVault   = "ActiveVault"
	EventTypeInactiveVault = "InactiveVault"
)

// VaultMgr is going to manage the vaults
type VaultMgr struct {
	k                   Keeper
	versionedTxOutStore VersionedTxOutStore
}

// NewVaultMgr create a new vault manager
func NewVaultMgr(k Keeper, versionedTxOutStore VersionedTxOutStore) *VaultMgr {
	return &VaultMgr{
		k:                   k,
		versionedTxOutStore: versionedTxOutStore,
	}
}

func (vm *VaultMgr) processGenesisSetup(ctx sdk.Context) error {
	if ctx.BlockHeight() != genesisBlockHeight {
		return nil
	}
	vaults, err := vm.k.GetAsgardVaults(ctx)
	if err != nil {
		return fmt.Errorf("fail to get vaults: %w", err)
	}
	if len(vaults) > 0 {
		ctx.Logger().Info("already have vault, no need to generate at genesis")
		return nil
	}
	active, err := vm.k.ListActiveNodeAccounts(ctx)
	if err != nil {
		return fmt.Errorf("fail to get all active node accounts")
	}
	if len(active) == 0 {
		return errors.New("no active accounts,cannot proceed")
	}
	if len(active) == 1 {
		vault := NewVault(0, ActiveVault, AsgardVault, active[0].PubKeySet.Secp256k1)
		vault.Membership = common.PubKeys{active[0].PubKeySet.Secp256k1}
		if err := vm.k.SetVault(ctx, vault); err != nil {
			return fmt.Errorf("fail to save vault: %w", err)
		}
	} else {
		// Trigger a keygen ceremony
		if err := vm.TriggerKeygen(ctx, active); err != nil {
			return fmt.Errorf("fail to trigger a keygen: %w", err)
		}
	}
	return nil
}

// EndBlock move funds from retiring asgard vaults
func (vm *VaultMgr) EndBlock(ctx sdk.Context, version semver.Version, constAccessor constants.ConstantValues) error {
	if ctx.BlockHeight() == genesisBlockHeight {
		return vm.processGenesisSetup(ctx)
	}

	migrateInterval := constAccessor.GetInt64Value(constants.FundMigrationInterval)

	retiring, err := vm.k.GetAsgardVaultsByStatus(ctx, RetiringVault)
	if err != nil {
		return err
	}

	active, err := vm.k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		return err
	}

	// if we have no active asgards to move funds to, don't move funds
	if len(active) == 0 {
		return nil
	}
	txOutStore, err := vm.versionedTxOutStore.GetTxOutStore(vm.k, version)
	if nil != err {
		ctx.Logger().Error("fail to get txout store", "error", err)
		return errBadVersion
	}
	for _, vault := range retiring {
		if !vault.HasFunds() {
			// no more funds to move, delete the vault
			fmt.Println("Vault IS DELETED")
			if err := vm.k.DeleteVault(ctx, vault.PubKey); err != nil {
				return err
			}
			continue
		}

		// move partial funds every 30 minutes
		if (ctx.BlockHeight()-vault.StatusSince)%migrateInterval == 0 {
			for _, coin := range vault.Coins {

				// determine which active asgard vault is the best to send
				// these coins to. We target the vault with the least amount of
				// this particular coin
				cn := active[0].GetCoin(coin.Asset)
				pk := active[0].PubKey
				for _, asgard := range active {
					if cn.Amount.GT(asgard.GetCoin(coin.Asset).Amount) {
						cn = asgard.GetCoin(coin.Asset)
						pk = asgard.PubKey
					}
				}

				// get address of asgard pubkey
				addr, err := pk.GetAddress(coin.Asset.Chain)
				if err != nil {
					return err
				}

				// figure the nth time, we've sent migration txs from this vault
				nth := (ctx.BlockHeight()-vault.StatusSince)/migrateInterval + 1

				// Default amount set to total remaining amount. Relies on the
				// signer, to successfully send these funds while respecting
				// gas requirements (so it'll actually send slightly less)
				amt := coin.Amount
				if nth < 5 { // migrate partial funds 4 times
					// each round of migration, we are increasing the amount 20%.
					// Round 1 = 20%
					// Round 2 = 40%
					// Round 3 = 60%
					// Round 4 = 80%
					// Round 5 = 100%
					amt = amt.MulUint64(uint64(nth)).QuoUint64(5)
				}

				toi := &TxOutItem{
					Chain:       coin.Asset.Chain,
					InHash:      common.BlankTxID,
					ToAddress:   addr,
					VaultPubKey: vault.PubKey,
					Coin: common.Coin{
						Asset:  coin.Asset,
						Amount: amt,
					},
					Memo: "migrate",
				}
				_, err = txOutStore.TryAddTxOutItem(ctx, toi)
				if nil != err {
					return err
				}
			}
		}
	}

	return nil
}

func (vm *VaultMgr) TriggerKeygen(ctx sdk.Context, nas NodeAccounts) error {
	keygen := make(common.PubKeys, len(nas))
	for i := range nas {
		keygen[i] = nas[i].PubKeySet.Secp256k1
	}
	keygens := NewKeygens(ctx.BlockHeight())
	keygens.Keygens = []common.PubKeys{keygen}
	return vm.k.SetKeygens(ctx, keygens)
}

func (vm *VaultMgr) RotateVault(ctx sdk.Context, vault Vault) error {
	fmt.Printf("Rotating vault... %d\n", len(vault.Membership))
	active, err := vm.k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		return err
	}

	// find vaults the new vault conflicts with, mark them as inactive
	for _, asgard := range active {
		for _, member := range asgard.Membership {
			if vault.Contains(member) {
				asgard.UpdateStatus(RetiringVault, ctx.BlockHeight())
				if err := vm.k.SetVault(ctx, asgard); err != nil {
					return err
				}

				ctx.EventManager().EmitEvent(
					sdk.NewEvent(EventTypeInactiveVault,
						sdk.NewAttribute("set asgard vault to inactive", asgard.PubKey.String())))
				break
			}
		}
	}

	// Update Node account membership
	for _, member := range vault.Membership {
		na, err := vm.k.GetNodeAccountByPubKey(ctx, member)
		if err != nil {
			return err
		}
		na.TryAddSignerPubKey(vault.PubKey)
		if err := vm.k.SetNodeAccount(ctx, na); err != nil {
			return err
		}
	}

	if err := vm.k.SetVault(ctx, vault); err != nil {
		return err
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(EventTypeActiveVault,
			sdk.NewAttribute("add new asgard vault", vault.PubKey.String())))
	return nil
}
