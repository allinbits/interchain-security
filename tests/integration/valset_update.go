package integration

import (
	"cosmossdk.io/math"

	ccv "github.com/allinbits/interchain-security/x/ccv/types"
)

// TestPacketRoundtrip tests a CCV packet roundtrip when tokens are bonded on provider
func (s *CCVTestSuite) TestPacketRoundtrip() {
	s.SetupCCVChannel(s.path)
	s.SetupTransferChannel()

	// Bond some tokens on provider to change validator powers
	bondAmt := math.NewInt(1000000)
	delAddr := s.providerChain.SenderAccount.GetAddress()
	delegate(s, delAddr, bondAmt)

	// Send CCV packet to consumer at the end of the epoch
	s.nextEpoch()

	// Relay 1 VSC packet from provider to consumer
	relayAllCommittedPackets(s, s.providerChain, s.path, ccv.ProviderPortID, s.path.EndpointB.ChannelID, 1)
}
