package v6

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	providerkeeper "github.com/cosmos/interchain-security/v5/x/ccv/provider/keeper"
	providertypes "github.com/cosmos/interchain-security/v5/x/ccv/provider/types"
)

// MigrateParams adds missing provider chain params to the param store.
func MigrateParams(ctx sdk.Context, paramsSubspace paramtypes.Subspace) {
	if !paramsSubspace.HasKeyTable() {
		paramsSubspace.WithKeyTable(providertypes.ParamKeyTable())
	}
	paramsSubspace.Set(ctx, providertypes.KeyNumberOfEpochsToStartReceivingRewards, providertypes.DefaultNumberOfEpochsToStartReceivingRewards)
}

func MigrateMinPowerInTopN(ctx sdk.Context, providerKeeper providerkeeper.Keeper) {
	// This migration is no longer needed for Replicated Security
	// All bonded validators participate, so there's no minimum power threshold
	providerKeeper.Logger(ctx).Info("MigrateMinPowerInTopN: Skipped - Replicated Security doesn't use Top N")
}
