package keeper

import (
	"fmt"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/interchain-security/x/ccv/consumer/types"
	"github.com/cosmos/interchain-security/x/ccv/utils"
	abci "github.com/tendermint/tendermint/abci/types"
)

// ApplyCCValidatorChanges applies the given changes to the cross-chain validators states
func (k Keeper) ApplyCCValidatorChanges(ctx sdk.Context, changes []abci.ValidatorUpdate) {
	for _, change := range changes {
		addr := utils.GetChangePubKeyAddress(change)
		val, found := k.GetCCValidator(ctx, addr)

		// set new validator bonded
		if !found {
			consAddr := sdk.ConsAddress(addr)
			if change.Power < 1 {
				panic(fmt.Errorf("new validator bonded with zero voting power: %s", consAddr))
			}
			// convert validator pubkey from TM proto to SDK crytpo type
			pubkey, err := cryptocodec.FromTmProtoPublicKey(change.GetPubKey())
			if err != nil {
				panic(err)
			}
			ccVal, err := types.NewCCValidator(addr, change.Power, pubkey)
			if err != nil {
				panic(err)
			}

			k.SetCCValidator(ctx, ccVal)
			k.AfterValidatorBonded(ctx, consAddr, nil)
			continue
		}

		// remove unbonding existing-validators
		if change.Power < 1 {
			k.DeleteCCValidator(ctx, addr)
			continue
		}

		// update existing validators power
		val.Power = change.Power
		k.SetCCValidator(ctx, val)
	}
}

// IterateValidators - unimplemented on CCV keeper
func (k Keeper) IterateValidators(sdk.Context, func(index int64, validator stakingtypes.ValidatorI) (stop bool)) {
	panic("unimplemented on CCV keeper")
}

// Validator - unimplemented on CCV keeper
func (k Keeper) Validator(ctx sdk.Context, addr sdk.ValAddress) stakingtypes.ValidatorI {
	panic("unimplemented on CCV keeper")
}

// IsJailed returns the outstanding slashing flag for the given validator adddress
func (k Keeper) IsValidatorJailed(ctx sdk.Context, addr sdk.ConsAddress) bool {
	return k.OutstandingDowntime(ctx, addr)
}

// ValidatorByConsAddr returns an empty validator
func (k Keeper) ValidatorByConsAddr(sdk.Context, sdk.ConsAddress) stakingtypes.ValidatorI {
	return stakingtypes.Validator{}
}

// Slash sends a slashing request to the provider chain
func (k Keeper) Slash(ctx sdk.Context, addr sdk.ConsAddress, infractionHeight, power int64, _ sdk.Dec, infraction stakingtypes.InfractionType) {
	if infraction == stakingtypes.InfractionEmpty {
		return
	}

	k.SendSlashPacket(
		ctx,
		abci.Validator{
			Address: addr.Bytes(),
			Power:   power},
		// get VSC ID for infraction height
		k.GetHeightValsetUpdateID(ctx, uint64(infractionHeight)),
		infraction,
	)
}

// Jail - unimplemented on CCV keeper
func (k Keeper) Jail(ctx sdk.Context, addr sdk.ConsAddress) {}

// Unjail - unimplemented on CCV keeper
func (k Keeper) Unjail(sdk.Context, sdk.ConsAddress) {}

// Delegation - unimplemented on CCV keeper
func (k Keeper) Delegation(sdk.Context, sdk.AccAddress, sdk.ValAddress) stakingtypes.DelegationI {
	panic("unimplemented on CCV keeper")
}

// MaxValidators - unimplemented on CCV keeper
func (k Keeper) MaxValidators(sdk.Context) uint32 {
	panic("unimplemented on CCV keeper")
}

// GetHistoricalInfo gets the historical info at a given height
func (k Keeper) GetHistoricalInfo(ctx sdk.Context, height int64) (stakingtypes.HistoricalInfo, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetHistoricalInfoKey(height)

	value := store.Get(key)
	if value == nil {
		return stakingtypes.HistoricalInfo{}, false
	}

	return stakingtypes.MustUnmarshalHistoricalInfo(k.cdc, value), true
}

// SetHistoricalInfo sets the historical info at a given height
func (k Keeper) SetHistoricalInfo(ctx sdk.Context, height int64, hi *stakingtypes.HistoricalInfo) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetHistoricalInfoKey(height)
	value := k.cdc.MustMarshal(hi)

	store.Set(key, value)
}

// DeleteHistoricalInfo deletes the historical info at a given height
func (k Keeper) DeleteHistoricalInfo(ctx sdk.Context, height int64) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetHistoricalInfoKey(height)

	store.Delete(key)
}

// TrackHistoricalInfo saves the latest historical-info and deletes the oldest
// heights that are below pruning height
func (k Keeper) TrackHistoricalInfo(ctx sdk.Context) {
	entryNum := types.HistoricalEntries

	// Prune store to ensure we only have parameter-defined historical entries.
	// In most cases, this will involve removing a single historical entry.
	// In the rare scenario when the historical entries gets reduced to a lower value k'
	// from the original value k. k - k' entries must be deleted from the store.
	// Since the entries to be deleted are always in a continuous range, we can iterate
	// over the historical entries starting from the most recent version to be pruned
	// and then return at the first empty entry.
	for i := ctx.BlockHeight() - int64(entryNum); i >= 0; i-- {
		_, found := k.GetHistoricalInfo(ctx, i)
		if found {
			k.DeleteHistoricalInfo(ctx, i)
		} else {
			break
		}
	}

	// if there is no need to persist historicalInfo, return
	if entryNum == 0 {
		return
	}

	// Create HistoricalInfo struct
	lastVals := []stakingtypes.Validator{}
	for _, v := range k.GetAllCCValidator(ctx) {
		pk, err := v.ConsPubKey()
		if err != nil {
			panic("invalid validator key")
		}
		val, err := stakingtypes.NewValidator(nil, pk, stakingtypes.Description{})
		if err != nil {
			panic("invalid validator key")
		}
		// Is it required ?
		val.Status = stakingtypes.Bonded
		val.Tokens = sdk.TokensFromConsensusPower(v.Power, sdk.DefaultPowerReduction)
		lastVals = append(lastVals, val)
	}

	historicalEntry := stakingtypes.NewHistoricalInfo(ctx.BlockHeader(), lastVals, sdk.DefaultPowerReduction)

	// Set latest HistoricalInfo at current height
	k.SetHistoricalInfo(ctx, ctx.BlockHeight(), &historicalEntry)
}
