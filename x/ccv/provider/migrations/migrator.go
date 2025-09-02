package migrations

import (
	storetypes "cosmossdk.io/store/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	providerkeeper "github.com/cosmos/interchain-security/v5/x/ccv/provider/keeper"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	providerKeeper providerkeeper.Keeper
	paramSpace     paramtypes.Subspace
	storeKey       storetypes.StoreKey
}

// NewMigrator returns a new Migrator.
func NewMigrator(
	providerKeeper providerkeeper.Keeper,
	paramSpace paramtypes.Subspace,
	storeKey storetypes.StoreKey,
) Migrator {
	return Migrator{
		providerKeeper: providerKeeper,
		paramSpace:     paramSpace,
		storeKey:       storeKey,
	}
}
