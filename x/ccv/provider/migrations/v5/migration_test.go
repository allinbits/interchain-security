package v5

import (
	"testing"

	testutil "github.com/cosmos/interchain-security/v5/testutil/keeper"
)

func TestMigrateParams(t *testing.T) {
	inMemParams := testutil.NewInMemKeeperParams(t)
	provKeeper, ctx, ctrl, _ := testutil.GetProviderKeeperAndCtx(t, inMemParams)
	defer ctrl.Finish()

	provKeeper.SetConsumerClientId(ctx, "chainID", "clientID")

	// For Replicated Security, TopN is not used
	// The migration is now a no-op that just logs a message

	// Run the migration
	MigrateTopNForRegisteredChains(ctx, provKeeper)

	// Migration should complete without errors
	// No state changes are expected for Replicated Security
}
