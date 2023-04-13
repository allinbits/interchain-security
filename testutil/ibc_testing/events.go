package ibc_testing

import (
	"strconv"

	abci "github.com/cometbft/cometbft/abci/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

// FIXME copied from legacy_ibc_testing/core/events.go bc of ReconstructPacketFromEvent functions
// Should we keep or replace that function?

// ReconstructPacketFromEvent recreates a packet from an appropriate provided event
func ReconstructPacketFromEvent(event abci.Event) (packet types.Packet, err error) {
	attrMap := make(map[string][]byte)
	for _, attr := range event.Attributes {
		attrMap[string(attr.Key)] = attr.Value
	}

	sequence, err := strconv.Atoi(string(attrMap[string(types.AttributeKeySequence)]))
	if err != nil {
		return packet, err
	}
	timeoutTimestamp, err := strconv.Atoi(string(attrMap[string(types.AttributeKeyTimeoutTimestamp)]))
	if err != nil {
		return packet, err
	}
	timeoutHeight, err := clienttypes.ParseHeight(string(attrMap[string(types.AttributeKeyTimeoutHeight)]))
	if err != nil {
		return packet, err
	}
	return types.NewPacket(
		attrMap[string(types.AttributeKeyData)], // data
		uint64(sequence),
		string(attrMap[string(types.AttributeKeySrcPort)]),    // sourcePort,
		string(attrMap[string(types.AttributeKeySrcChannel)]), // sourceChannel,
		string(attrMap[string(types.AttributeKeyDstPort)]),    // destinationPort,
		string(attrMap[string(types.AttributeKeyDstChannel)]), // destinationChannel string,
		timeoutHeight,
		uint64(timeoutTimestamp),
	), nil
}
