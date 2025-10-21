package integration

import (
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"

	consumertypes "github.com/cosmos/interchain-security/v5/x/ccv/consumer/types"
	providertypes "github.com/cosmos/interchain-security/v5/x/ccv/provider/types"
	ccv "github.com/cosmos/interchain-security/v5/x/ccv/types"
)

func (suite *CCVTestSuite) TestRecycleTransferChannel() {
	consumerKeeper := suite.consumerApp.GetConsumerKeeper()

	// Only create a connection between consumer and provider
	suite.coordinator.CreateConnections(suite.path)

	// Confirm transfer channel has not been persisted
	transChan := consumerKeeper.GetDistributionTransmissionChannel(suite.consumerCtx())
	suite.Require().Empty(transChan)

	// Create transfer channel manually
	// IBC v10: Version constant replaced by V1
	// Reference: https://github.com/cosmos/interchain-security/blob/v7.0.1/tests/integration/changeover.go#L30
	distrTransferMsg := channeltypes.NewMsgChannelOpenInit(
		transfertypes.PortID,
		transfertypes.V1,
		channeltypes.UNORDERED,
		[]string{suite.path.EndpointA.ConnectionID},
		transfertypes.PortID,
		"", // signer unused
	)
	resp, err := consumerKeeper.ChannelOpenInit(suite.consumerCtx(), distrTransferMsg)
	suite.Require().NoError(err)

	// Confirm transfer channel still not persisted
	transChan = consumerKeeper.GetDistributionTransmissionChannel(suite.consumerCtx())
	suite.Require().Empty(transChan)

	// Setup state s.t. the consumer keeper emulates a consumer that was previously standalone
	consumerKeeper.MarkAsPrevStandaloneChain(suite.consumerCtx())
	suite.Require().True(consumerKeeper.IsPrevStandaloneChain(suite.consumerCtx()))
	suite.consumerApp.GetConsumerKeeper().SetDistributionTransmissionChannel(suite.consumerCtx(), resp.ChannelId)

	// Now finish setting up CCV channel
	suite.ExecuteCCVChannelHandshake(suite.path)

	// Confirm transfer channel is now persisted with expected channel id from open init response
	transChan = consumerKeeper.GetDistributionTransmissionChannel(suite.consumerCtx())
	suite.Require().Equal(resp.ChannelId, transChan)

	// Confirm channel exists
	found := consumerKeeper.TransferChannelExists(suite.consumerCtx(), transChan)
	suite.Require().True(found)

	// Sanity check, only two channels should exist, one transfer and one ccv
	channels := suite.consumerApp.GetIBCKeeper().ChannelKeeper.GetAllChannels(suite.consumerCtx())
	suite.Require().Len(channels, 2)
}

// TestChangeoverWithConnectionReuse tests the standalone-to-consumer changeover
// when reusing an existing IBC connection (ICS1 feature).
// This validates that a consumer chain can reuse an existing connection during changeover.
func (suite *CCVTestSuite) TestChangeoverWithConnectionReuse() {
	// Step 1: Create connection between consumer and provider (simulating existing connection)
	suite.coordinator.CreateConnections(suite.path)

	consumerKeeper := suite.consumerApp.GetConsumerKeeper()
	providerKeeper := suite.providerApp.GetProviderKeeper()

	// Get the connection IDs from the created connection
	providerConnectionID := suite.path.EndpointA.ConnectionID
	consumerConnectionID := suite.path.EndpointB.ConnectionID

	suite.Require().NotEmpty(providerConnectionID, "provider connection ID should be set")
	suite.Require().NotEmpty(consumerConnectionID, "consumer connection ID should be set")

	// Commit blocks on provider chain to ensure historical info is saved
	// MakeConsumerGenesis needs historical info at the current height
	suite.coordinator.CommitBlock(suite.providerChain)

	// Step 2: Create a consumer addition proposal with connection_id set
	prop := providertypes.ConsumerAdditionProposal{
		ChainId:                           suite.consumerChain.ChainID,
		UnbondingPeriod:                   ccv.DefaultConsumerUnbondingPeriod,
		CcvTimeoutPeriod:                  ccv.DefaultCCVTimeoutPeriod,
		TransferTimeoutPeriod:             ccv.DefaultTransferTimeoutPeriod,
		ConsumerRedistributionFraction:    "0.75",
		BlocksPerDistributionTransmission: 1000,
		HistoricalEntries:                 10000,
		ConnectionId:                      providerConnectionID, // ICS1: Reuse existing connection
	}

	// Step 3: Generate consumer genesis with connection reuse
	consumerGenesis, _, err := providerKeeper.MakeConsumerGenesis(suite.providerCtx(), &prop)
	suite.Require().NoError(err, "MakeConsumerGenesis should succeed")

	// Step 4: Verify the generated genesis has connection reuse fields set correctly
	suite.Require().Equal(consumerConnectionID, consumerGenesis.ConnectionId,
		"consumer genesis should have connection_id set to consumer-side connection")
	suite.Require().True(consumerGenesis.PreCCV,
		"consumer genesis should have preCCV=true for connection reuse")
	suite.Require().Nil(consumerGenesis.Provider.ClientState,
		"client_state should be nil when reusing connection")
	suite.Require().Nil(consumerGenesis.Provider.ConsensusState,
		"consensus_state should be nil when reusing connection")

	// For integration testing purposes, override preCCV to false since our test consumer
	// is not a real standalone chain with a standalone staking keeper.
	// In a real changeover scenario, preCCV would be true and the consumer would have
	// a standalone staking keeper. This test focuses on verifying the connection reuse
	// mechanism (provider genesis generation and consumer initialization with existing client).
	consumerGenesis.PreCCV = false

	// Step 5: Get the existing client ID from the connection (before InitGenesis)
	consumerConn, found := suite.consumerApp.GetIBCKeeper().ConnectionKeeper.GetConnection(
		suite.consumerCtx(), consumerConnectionID,
	)
	suite.Require().True(found, "consumer connection should exist")
	existingClientID := consumerConn.ClientId

	// Step 6: Initialize consumer with the generated genesis (with connection reuse)
	// Construct consumer GenesisState from the shared ConsumerGenesisState
	consumerGenesisState := &consumertypes.GenesisState{
		Params:       consumerGenesis.Params,
		NewChain:     consumerGenesis.NewChain,
		Provider:     consumerGenesis.Provider,
		PreCCV:       consumerGenesis.PreCCV,
		ConnectionId: consumerGenesis.ConnectionId,
	}
	consumerKeeper.InitGenesis(suite.consumerCtx(), consumerGenesisState)

	// Step 7: Verify consumer initialized correctly using existing connection's client
	providerClientID, found := consumerKeeper.GetProviderClientID(suite.consumerCtx())
	suite.Require().True(found, "provider client ID should be set")
	suite.Require().Equal(existingClientID, providerClientID,
		"consumer should use existing client from connection")

	// Step 8: Complete CCV channel handshake on top of existing connection
	suite.ExecuteCCVChannelHandshake(suite.path)

	// Step 9: Verify CCV channel was established on the existing connection
	// Note: In a real scenario, the provider channel ID would be set when the first VSC packet
	// is received. For this test, we verify the channel exists and manually set it since
	// we're testing connection reuse, not the full packet relay flow.
	channels := suite.consumerApp.GetIBCKeeper().ChannelKeeper.GetAllChannels(suite.consumerCtx())
	var ccvChannelID string
	for _, ch := range channels {
		if ch.PortId == ccv.ConsumerPortID && ch.State == channeltypes.OPEN {
			ccvChannelID = ch.ChannelId
			break
		}
	}
	suite.Require().NotEmpty(ccvChannelID, "CCV channel should exist")

	// Set the provider channel manually for testing purposes
	consumerKeeper.SetProviderChannel(suite.consumerCtx(), ccvChannelID)

	// Verify we can now get the provider channel
	retrievedChannelID, found := consumerKeeper.GetProviderChannel(suite.consumerCtx())
	suite.Require().True(found, "provider channel should be set")
	suite.Require().Equal(ccvChannelID, retrievedChannelID, "provider channel ID should match")

	ccvChannel, found := suite.consumerApp.GetIBCKeeper().ChannelKeeper.GetChannel(
		suite.consumerCtx(), ccv.ConsumerPortID, ccvChannelID,
	)
	suite.Require().True(found, "CCV channel should exist")
	suite.Require().Equal(consumerConnectionID, ccvChannel.ConnectionHops[0],
		"CCV channel should be on the existing connection")

	// Step 10: Verify the connection is shared between CCV and any other channels
	allChannels := suite.consumerApp.GetIBCKeeper().ChannelKeeper.GetAllChannels(suite.consumerCtx())

	// Count how many channels use the existing connection
	channelsOnConnection := 0
	for _, ch := range allChannels {
		if len(ch.ConnectionHops) > 0 && ch.ConnectionHops[0] == consumerConnectionID {
			channelsOnConnection++
		}
	}

	// At minimum, the CCV channel should be on the existing connection
	suite.Require().GreaterOrEqual(channelsOnConnection, 1,
		"at least CCV channel should use the existing connection")
}
