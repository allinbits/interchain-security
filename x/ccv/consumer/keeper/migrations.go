package keeper

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Migrator is a struct for handling in-place store migrations.
// Note: This is a complete fork, so no migrations are implemented.
type Migrator struct {
	keeper     Keeper
	paramSpace paramtypes.Subspace
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper, paramspace paramtypes.Subspace) Migrator {
	return Migrator{keeper: keeper, paramSpace: paramspace}
}
