package keeper_test

import (
	"fmt"
	"sort"
	"testing"

	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/stretchr/testify/require"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"

	abci "github.com/cometbft/cometbft/abci/types"
	tmprotocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"

	cryptotestutil "github.com/cosmos/interchain-security/v5/testutil/crypto"
	testkeeper "github.com/cosmos/interchain-security/v5/testutil/keeper"
	"github.com/cosmos/interchain-security/v5/x/ccv/provider/types"
	ccv "github.com/cosmos/interchain-security/v5/x/ccv/types"
)

const consumer = "consumer"

// TestValsetUpdateBlockHeight tests the getter, setter, and deletion methods for valset updates mapped to block height
func TestValsetUpdateBlockHeight(t *testing.T) {
	providerKeeper, ctx, ctrl, _ := testkeeper.GetProviderKeeperAndCtx(t, testkeeper.NewInMemKeeperParams(t))
	defer ctrl.Finish()

	blockHeight, found := providerKeeper.GetValsetUpdateBlockHeight(ctx, uint64(0))
	require.False(t, found)
	require.Zero(t, blockHeight)

	providerKeeper.SetValsetUpdateBlockHeight(ctx, uint64(1), uint64(2))
	blockHeight, found = providerKeeper.GetValsetUpdateBlockHeight(ctx, uint64(1))
	require.True(t, found)
	require.Equal(t, blockHeight, uint64(2))

	providerKeeper.DeleteValsetUpdateBlockHeight(ctx, uint64(1))
	blockHeight, found = providerKeeper.GetValsetUpdateBlockHeight(ctx, uint64(1))
	require.False(t, found)
	require.Zero(t, blockHeight)

	providerKeeper.SetValsetUpdateBlockHeight(ctx, uint64(1), uint64(2))
	providerKeeper.SetValsetUpdateBlockHeight(ctx, uint64(3), uint64(4))
	blockHeight, found = providerKeeper.GetValsetUpdateBlockHeight(ctx, uint64(3))
	require.True(t, found)
	require.Equal(t, blockHeight, uint64(4))
}

// TestGetAllValsetUpdateBlockHeights tests GetAllValsetUpdateBlockHeights behaviour correctness
func TestGetAllValsetUpdateBlockHeights(t *testing.T) {
	pk, ctx, ctrl, _ := testkeeper.GetProviderKeeperAndCtx(t, testkeeper.NewInMemKeeperParams(t))
	defer ctrl.Finish()

	cases := []types.ValsetUpdateIdToHeight{
		{
			ValsetUpdateId: 2,
			Height:         22,
		},
		{
			ValsetUpdateId: 1,
			Height:         11,
		},
		{
			// normal execution should not have two ValsetUpdateIdToHeight
			// with the same Height, but let's test it anyway
			ValsetUpdateId: 4,
			Height:         11,
		},
		{
			ValsetUpdateId: 3,
			Height:         33,
		},
	}
	expectedGetAllOrder := cases
	// sorting by ValsetUpdateId
	sort.Slice(expectedGetAllOrder, func(i, j int) bool {
		return expectedGetAllOrder[i].ValsetUpdateId < expectedGetAllOrder[j].ValsetUpdateId
	})

	for _, c := range cases {
		pk.SetValsetUpdateBlockHeight(ctx, c.ValsetUpdateId, c.Height)
	}

	// iterate and check all results are returned in the expected order
	result := pk.GetAllValsetUpdateBlockHeights(ctx)
	require.Len(t, result, len(cases))
	require.Equal(t, expectedGetAllOrder, result)
}

// TestSlashAcks tests the getter, setter, iteration, and deletion methods for stored slash acknowledgements
func TestSlashAcks(t *testing.T) {
	providerKeeper, ctx, ctrl, _ := testkeeper.GetProviderKeeperAndCtx(t, testkeeper.NewInMemKeeperParams(t))
	defer ctrl.Finish()

	chainID := consumer

	acks := providerKeeper.GetSlashAcks(ctx, chainID)
	require.Nil(t, acks)

	p := []string{"alice", "bob", "charlie"}
	providerKeeper.SetSlashAcks(ctx, chainID, p)

	acks = providerKeeper.GetSlashAcks(ctx, chainID)
	require.NotNil(t, acks)

	require.Len(t, acks, 3)
	slashAcks := providerKeeper.ConsumeSlashAcks(ctx, chainID)
	require.Len(t, slashAcks, 3)

	acks = providerKeeper.GetSlashAcks(ctx, chainID)
	require.Nil(t, acks)

	chains := []string{"c1", "c2", "c3"}

	for _, c := range chains {
		providerKeeper.SetSlashAcks(ctx, c, p)
	}

	for _, c := range chains {
		require.Equal(t, p, providerKeeper.GetSlashAcks(ctx, c))
		providerKeeper.DeleteSlashAcks(ctx, c)
		acks = providerKeeper.GetSlashAcks(ctx, c)
		require.Len(t, acks, 0)
	}
}

// TestAppendSlashAck tests the append method for stored slash acknowledgements
func TestAppendSlashAck(t *testing.T) {
	providerKeeper, ctx, ctrl, _ := testkeeper.GetProviderKeeperAndCtx(t, testkeeper.NewInMemKeeperParams(t))
	defer ctrl.Finish()

	p := []string{"alice", "bob", "charlie"}
	chains := []string{"c1", "c2"}
	providerKeeper.SetSlashAcks(ctx, chains[0], p)

	providerKeeper.AppendSlashAck(ctx, chains[0], p[0])
	acks := providerKeeper.GetSlashAcks(ctx, chains[0])
	require.NotNil(t, acks)
	require.Len(t, acks, len(p)+1)

	providerKeeper.AppendSlashAck(ctx, chains[1], p[0])
	acks = providerKeeper.GetSlashAcks(ctx, chains[1])
	require.NotNil(t, acks)
	require.Len(t, acks, 1)
}

// TestPendingVSCs tests the getter, appending, and deletion methods for stored pending VSCs
func TestPendingVSCs(t *testing.T) {
	providerKeeper, ctx, ctrl, _ := testkeeper.GetProviderKeeperAndCtx(t, testkeeper.NewInMemKeeperParams(t))
	defer ctrl.Finish()

	chainID := consumer

	pending := providerKeeper.GetPendingVSCPackets(ctx, chainID)
	require.Len(t, pending, 0)

	_, pks, _ := ibctesting.GenerateKeys(t, 4)
	var ppks [4]tmprotocrypto.PublicKey
	for i, pk := range pks {
		ppks[i], _ = cryptocodec.ToCmtProtoPublicKey(pk)
	}

	packetList := []ccv.ValidatorSetChangePacketData{
		{
			ValidatorUpdates: []abci.ValidatorUpdate{
				{PubKey: ppks[0], Power: 1},
				{PubKey: ppks[1], Power: 2},
			},
			ValsetUpdateId: 1,
		},
		{
			ValidatorUpdates: []abci.ValidatorUpdate{
				{PubKey: ppks[2], Power: 3},
			},
			ValsetUpdateId: 2,
		},
	}
	providerKeeper.AppendPendingVSCPackets(ctx, chainID, packetList...)

	packets := providerKeeper.GetPendingVSCPackets(ctx, chainID)
	require.Len(t, packets, 2)

	newPacket := ccv.ValidatorSetChangePacketData{
		ValidatorUpdates: []abci.ValidatorUpdate{
			{PubKey: ppks[3], Power: 4},
		},
		ValsetUpdateId: 3,
	}
	providerKeeper.AppendPendingVSCPackets(ctx, chainID, newPacket)
	vscs := providerKeeper.GetPendingVSCPackets(ctx, chainID)
	require.Len(t, vscs, 3)
	require.True(t, vscs[len(vscs)-1].ValsetUpdateId == 3)
	require.True(t, vscs[len(vscs)-1].GetValidatorUpdates()[0].PubKey.String() == ppks[3].String())

	providerKeeper.DeletePendingVSCPackets(ctx, chainID)
	pending = providerKeeper.GetPendingVSCPackets(ctx, chainID)
	require.Len(t, pending, 0)
}

// TestInitHeight tests the getter and setter methods for the stored block heights (on provider) when a given consumer chain was started
func TestInitHeight(t *testing.T) {
	providerKeeper, ctx, ctrl, _ := testkeeper.GetProviderKeeperAndCtx(t, testkeeper.NewInMemKeeperParams(t))
	defer ctrl.Finish()

	tc := []struct {
		chainID  string
		expected uint64
	}{
		{expected: 0, chainID: "chain"},
		{expected: 10, chainID: "chain1"},
		{expected: 12, chainID: "chain2"},
	}

	providerKeeper.SetInitChainHeight(ctx, tc[1].chainID, tc[1].expected)
	providerKeeper.SetInitChainHeight(ctx, tc[2].chainID, tc[2].expected)

	for _, tc := range tc {
		height, _ := providerKeeper.GetInitChainHeight(ctx, tc.chainID)
		require.Equal(t, tc.expected, height)
	}
}

func TestGetAllRegisteredConsumerChainIDs(t *testing.T) {
	pk, ctx, ctrl, _ := testkeeper.GetProviderKeeperAndCtx(t, testkeeper.NewInMemKeeperParams(t))
	defer ctrl.Finish()

	chainIDs := []string{"chain-2", "chain-1", "chain-4", "chain-3"}
	// GetAllRegisteredConsumerChainIDs iterates over chainID in lexicographical order
	expectedChainIDs := []string{"chain-1", "chain-2", "chain-3", "chain-4"}

	for i, chainID := range chainIDs {
		clientID := fmt.Sprintf("client-%d", len(chainIDs)-i)
		pk.SetConsumerClientId(ctx, chainID, clientID)
	}

	result := pk.GetAllRegisteredConsumerChainIDs(ctx)
	require.Len(t, result, len(chainIDs))
	require.Equal(t, expectedChainIDs, result)
}

// TestSetSlashLog tests slash log getter and setter methods
func TestSetSlashLog(t *testing.T) {
	providerKeeper, ctx, ctrl, _ := testkeeper.GetProviderKeeperAndCtx(t, testkeeper.NewInMemKeeperParams(t))
	defer ctrl.Finish()

	addrWithDoubleSigns := cryptotestutil.NewCryptoIdentityFromIntSeed(1).ProviderConsAddress()
	addrWithoutDoubleSigns := cryptotestutil.NewCryptoIdentityFromIntSeed(2).ProviderConsAddress()

	providerKeeper.SetSlashLog(ctx, addrWithDoubleSigns)
	require.True(t, providerKeeper.GetSlashLog(ctx, addrWithDoubleSigns))
	require.False(t, providerKeeper.GetSlashLog(ctx, addrWithoutDoubleSigns))
}

func TestSetProposedConsumerChains(t *testing.T) {
	providerKeeper, ctx, ctrl, _ := testkeeper.GetProviderKeeperAndCtx(t, testkeeper.NewInMemKeeperParams(t))
	defer ctrl.Finish()

	tests := []struct {
		chainID    string
		proposalID uint64
	}{
		{chainID: "1", proposalID: 1},
		{chainID: "some other ID", proposalID: 12},
		{chainID: "some other other chain ID", proposalID: 123},
		{chainID: "", proposalID: 1234},
	}

	for _, test := range tests {
		providerKeeper.SetProposedConsumerChain(ctx, test.chainID, test.proposalID)
		cID, _ := providerKeeper.GetProposedConsumerChain(ctx, test.proposalID)
		require.Equal(t, cID, test.chainID)
	}
}

func TestDeleteProposedConsumerChainInStore(t *testing.T) {
	providerKeeper, ctx, ctrl, _ := testkeeper.GetProviderKeeperAndCtx(t, testkeeper.NewInMemKeeperParams(t))
	defer ctrl.Finish()

	tests := []struct {
		chainID          string
		proposalID       uint64
		deleteProposalID uint64
		ok               bool
	}{
		{chainID: "1", proposalID: 1, deleteProposalID: 1, ok: true},
		{chainID: "", proposalID: 12, deleteProposalID: 12, ok: true},
		{chainID: "1", proposalID: 0, deleteProposalID: 1, ok: false},
	}
	for _, test := range tests {
		providerKeeper.SetProposedConsumerChain(ctx, test.chainID, test.proposalID)
		providerKeeper.DeleteProposedConsumerChainInStore(ctx, test.deleteProposalID)
		cID, found := providerKeeper.GetProposedConsumerChain(ctx, test.proposalID)
		if test.ok {
			require.False(t, found)
		} else {
			require.Equal(t, cID, test.chainID)
		}
	}
}

func TestGetAllProposedConsumerChainIDs(t *testing.T) {
	providerKeeper, ctx, ctrl, _ := testkeeper.GetProviderKeeperAndCtx(t, testkeeper.NewInMemKeeperParams(t))
	defer ctrl.Finish()
	tests := [][]types.ProposedChain{
		{},
		{
			{
				ChainID:    "1",
				ProposalID: 1,
			},
		},
		{
			{
				ChainID:    "1",
				ProposalID: 1,
			},
			{
				ChainID:    "2",
				ProposalID: 2,
			},
			{
				ChainID:    "",
				ProposalID: 3,
			},
		},
	}

	for _, test := range tests {
		for _, tc := range test {
			providerKeeper.SetProposedConsumerChain(ctx, tc.ChainID, tc.ProposalID)
		}

		chains := providerKeeper.GetAllProposedConsumerChainIDs(ctx)

		sort.Slice(chains, func(i, j int) bool {
			return chains[i].ProposalID < chains[j].ProposalID
		})
		sort.Slice(test, func(i, j int) bool {
			return test[i].ProposalID < test[j].ProposalID
		})
		require.Equal(t, chains, test)

		for _, tc := range test {
			providerKeeper.DeleteProposedConsumerChainInStore(ctx, tc.ProposalID)
		}
	}
}
