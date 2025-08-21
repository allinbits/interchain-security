package keeper_test

import (
	"fmt"
	"testing"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	cryptotestutil "github.com/cosmos/interchain-security/v5/testutil/crypto"
	testkeeper "github.com/cosmos/interchain-security/v5/testutil/keeper"
	"github.com/cosmos/interchain-security/v5/x/ccv/provider/types"
	ccvtypes "github.com/cosmos/interchain-security/v5/x/ccv/types"
	"github.com/stretchr/testify/require"
)

func TestQueryAllPairsValConAddrByConsumerChainID(t *testing.T) {
	chainID := consumer

	providerConsAddress, err := sdktypes.ConsAddressFromBech32("cosmosvalcons1wpex7anfv3jhystyv3eq20r35a")
	require.NoError(t, err)
	providerAddr := types.NewProviderConsAddress(providerConsAddress)

	consumerKey := cryptotestutil.NewCryptoIdentityFromIntSeed(1).TMProtoCryptoPublicKey()
	consumerAddr, err := ccvtypes.TMCryptoPublicKeyToConsAddr(consumerKey)
	require.NoError(t, err)

	pk, ctx, ctrl, _ := testkeeper.GetProviderKeeperAndCtx(t, testkeeper.NewInMemKeeperParams(t))
	defer ctrl.Finish()

	pk.SetValidatorConsumerPubKey(ctx, chainID, providerAddr, consumerKey)

	consumerPubKey, found := pk.GetValidatorConsumerPubKey(ctx, chainID, providerAddr)
	require.True(t, found, "consumer pubkey not found")
	require.NotEmpty(t, consumerPubKey, "consumer pubkey is empty")
	require.Equal(t, consumerPubKey, consumerKey)

	// Request is nil
	_, err = pk.QueryAllPairsValConAddrByConsumerChainID(ctx, nil)
	require.Error(t, err)

	// Request with chainId is empty
	_, err = pk.QueryAllPairsValConAddrByConsumerChainID(ctx, &types.QueryAllPairsValConAddrByConsumerChainIDRequest{})
	require.Error(t, err)

	// Request with chainId is invalid
	response, err := pk.QueryAllPairsValConAddrByConsumerChainID(ctx, &types.QueryAllPairsValConAddrByConsumerChainIDRequest{ChainId: "invalidChainId"})
	require.NoError(t, err)
	require.Equal(t, []*types.PairValConAddrProviderAndConsumer{}, response.PairValConAddr)

	// Request is valid
	response, err = pk.QueryAllPairsValConAddrByConsumerChainID(ctx, &types.QueryAllPairsValConAddrByConsumerChainIDRequest{ChainId: chainID})
	require.NoError(t, err)

	expectedResult := types.PairValConAddrProviderAndConsumer{
		ProviderAddress: providerConsAddress.String(),
		ConsumerAddress: consumerAddr.String(),
		ConsumerKey:     &consumerKey,
	}
	require.Equal(t, &consumerKey, response.PairValConAddr[0].ConsumerKey)
	require.Equal(t, &expectedResult, response.PairValConAddr[0])
}

func TestQueryConsumerValidators(t *testing.T) {
	chainID := "chainID"

	pk, ctx, ctrl, _ := testkeeper.GetProviderKeeperAndCtx(t, testkeeper.NewInMemKeeperParams(t))
	defer ctrl.Finish()

	req := types.QueryConsumerValidatorsRequest{
		ChainId: chainID,
	}

	// error returned from not-started chain
	_, err := pk.QueryConsumerValidators(ctx, &req)
	require.Error(t, err)

	providerAddr1 := types.NewProviderConsAddress([]byte("providerAddr1"))
	consumerKey1 := cryptotestutil.NewCryptoIdentityFromIntSeed(1).TMProtoCryptoPublicKey()
	consumerValidator1 := types.ConsumerValidator{ProviderConsAddr: providerAddr1.ToSdkConsAddr(), Power: 1, ConsumerPublicKey: &consumerKey1}

	providerAddr2 := types.NewProviderConsAddress([]byte("providerAddr2"))
	consumerKey2 := cryptotestutil.NewCryptoIdentityFromIntSeed(2).TMProtoCryptoPublicKey()
	consumerValidator2 := types.ConsumerValidator{ProviderConsAddr: providerAddr2.ToSdkConsAddr(), Power: 2, ConsumerPublicKey: &consumerKey2}

	expectedResponse := types.QueryConsumerValidatorsResponse{
		Validators: []*types.QueryConsumerValidatorsValidator{
			{
				ProviderAddress: providerAddr1.String(),
				ConsumerKey:     &consumerKey1,
				Power:           1,
			},
			{
				ProviderAddress: providerAddr2.String(),
				ConsumerKey:     &consumerKey2,
				Power:           2,
			},
		},
	}

	// set up the client id so the chain looks like it "started"
	pk.SetConsumerClientId(ctx, chainID, "clientID")
	pk.SetConsumerValSet(ctx, chainID, []types.ConsumerValidator{consumerValidator1, consumerValidator2})

	res, err := pk.QueryConsumerValidators(ctx, &req)
	require.NoError(t, err)
	require.Equal(t, &expectedResponse, res)
}

// TestGetConsumerChain tests GetConsumerChain behaviour correctness
func TestGetConsumerChain(t *testing.T) {
	pk, ctx, ctrl, _ := testkeeper.GetProviderKeeperAndCtx(t, testkeeper.NewInMemKeeperParams(t))
	defer ctrl.Finish()

	chainIDs := []string{"chain-1", "chain-2", "chain-3", "chain-4"}

	expectedGetAllOrder := []types.Chain{}
	for i, chainID := range chainIDs {
		clientID := fmt.Sprintf("client-%d", len(chainIDs)-i)
		pk.SetConsumerClientId(ctx, chainID, clientID)

		expectedGetAllOrder = append(expectedGetAllOrder,
			types.Chain{
				ChainId:  chainID,
				ClientId: clientID,
			})
	}

	for i, chainID := range pk.GetAllRegisteredAndProposedChainIDs(ctx) {
		c, err := pk.GetConsumerChain(ctx, chainID)
		require.NoError(t, err)
		require.Equal(t, expectedGetAllOrder[i], c)
	}
}
