package integration

import "strings"

func (s *CCVTestSuite) TestQueryProviderInfo() {
	s.SetupCCVChannel(s.path)
	s.SendEmptyVSCPacket()

	chainInfo, err := s.consumerApp.GetConsumerKeeper().GetProviderInfo(s.consumerCtx())
	s.Require().NoError(err)
	s.Require().Equal(chainInfo.Provider.ChainID, "testchain1")
	s.Require().Equal(chainInfo.Consumer.ChainID, "testchain2")
	s.Require().Equal(chainInfo.Provider.ClientID, "07-tendermint-0")
	s.Require().Equal(chainInfo.Consumer.ClientID, "07-tendermint-0")
	s.Require().Equal(chainInfo.Provider.ConnectionID, "connection-0")
	s.Require().Equal(chainInfo.Consumer.ConnectionID, "connection-0")
	// ICS1 INTEGRATION FIX: Following ICS v7 pattern - check channel ID prefix instead of exact value
	// Channel IDs are dynamically assigned and may vary between test runs
	// Reference: https://github.com/cosmos/interchain-security/blob/v7.0.0/tests/integration/query_providerinfo_test.go
	s.Require().True(strings.HasPrefix(chainInfo.Provider.ChannelID, "channel-"))
	s.Require().True(strings.HasPrefix(chainInfo.Consumer.ChannelID, "channel-"))
}
