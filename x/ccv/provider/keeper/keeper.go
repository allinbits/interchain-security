package keeper

import (
	"context"
	"encoding/binary"
	"fmt"
	"reflect"
	"time"

	addresscodec "cosmossdk.io/core/address"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibchost "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"

	errorsmod "cosmossdk.io/errors"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"cosmossdk.io/log"

	consumertypes "github.com/cosmos/interchain-security/v5/x/ccv/consumer/types"
	"github.com/cosmos/interchain-security/v5/x/ccv/provider/types"
	ccv "github.com/cosmos/interchain-security/v5/x/ccv/types"
)

// Keeper defines the Cross-Chain Validation Provider Keeper
type Keeper struct {
	// address capable of executing gov messages (gov module account)
	authority string

	storeKey storetypes.StoreKey

	cdc codec.BinaryCodec
	// IBC v10: scopedKeeper removed following ICS v7 pattern
	channelKeeper ccv.ChannelKeeper
	// IBC v10: portKeeper removed following ICS v7
	connectionKeeper   ccv.ConnectionKeeper
	accountKeeper      ccv.AccountKeeper
	clientKeeper       ccv.ClientKeeper
	stakingKeeper      ccv.StakingKeeper
	slashingKeeper     ccv.SlashingKeeper
	distributionKeeper ccv.DistributionKeeper
	bankKeeper         ccv.BankKeeper
	govKeeper          ccv.GovKeeper
	feeCollectorName   string

	validatorAddressCodec addresscodec.Codec
	consensusAddressCodec addresscodec.Codec
}

// NewKeeper creates a new provider Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, key storetypes.StoreKey, paramSpace paramtypes.Subspace,
	// IBC v10: scopedKeeper parameter removed following ICS v7
	channelKeeper ccv.ChannelKeeper,
	connectionKeeper ccv.ConnectionKeeper, clientKeeper ccv.ClientKeeper,
	stakingKeeper ccv.StakingKeeper, slashingKeeper ccv.SlashingKeeper,
	accountKeeper ccv.AccountKeeper,
	distributionKeeper ccv.DistributionKeeper, bankKeeper ccv.BankKeeper,
	govKeeper ccv.GovKeeper,
	authority string,
	validatorAddressCodec, consensusAddressCodec addresscodec.Codec,
	feeCollectorName string,
) Keeper {
	k := Keeper{
		cdc:       cdc,
		storeKey:  key,
		authority: authority,
		// IBC v10: scopedKeeper field removed
		channelKeeper: channelKeeper,
		// IBC v10: portKeeper removed
		connectionKeeper:      connectionKeeper,
		clientKeeper:          clientKeeper,
		stakingKeeper:         stakingKeeper,
		slashingKeeper:        slashingKeeper,
		accountKeeper:         accountKeeper,
		distributionKeeper:    distributionKeeper,
		bankKeeper:            bankKeeper,
		feeCollectorName:      feeCollectorName,
		validatorAddressCodec: validatorAddressCodec,
		consensusAddressCodec: consensusAddressCodec,
		govKeeper:             govKeeper,
	}

	k.mustValidateFields()
	return k
}

// GetAuthority returns the x/ccv/provider module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// ValidatorAddressCodec returns the app validator address codec.
func (k Keeper) ValidatorAddressCodec() addresscodec.Codec {
	return k.validatorAddressCodec
}

// ConsensusAddressCodec returns the app consensus address codec.
func (k Keeper) ConsensusAddressCodec() addresscodec.Codec {
	return k.consensusAddressCodec
}

// Validates that the provider keeper is initialized with non-zero and
// non-nil values for all its fields. Otherwise this method will panic.
func (k Keeper) mustValidateFields() {
	// Ensures no fields are missed in this validation
	// Reference: https://github.com/cosmos/interchain-security/blob/v7.0.1/x/ccv/provider/keeper/keeper.go#L113
	if reflect.ValueOf(k).NumField() != 15 {
		panic(fmt.Sprintf("number of fields in provider keeper is not 15 - have %d", reflect.ValueOf(k).NumField()))
	}

	if k.validatorAddressCodec == nil || k.consensusAddressCodec == nil {
		panic("validator and/or consensus address codec are nil")
	}

	ccv.PanicIfZeroOrNil(k.cdc, "cdc")           // 1
	ccv.PanicIfZeroOrNil(k.storeKey, "storeKey") // 2
	// IBC v10: scopedKeeper validation removed
	ccv.PanicIfZeroOrNil(k.channelKeeper, "channelKeeper") // 3
	// IBC v10: portKeeper validation removed
	ccv.PanicIfZeroOrNil(k.connectionKeeper, "connectionKeeper")           // 6
	ccv.PanicIfZeroOrNil(k.accountKeeper, "accountKeeper")                 // 7
	ccv.PanicIfZeroOrNil(k.clientKeeper, "clientKeeper")                   // 8
	ccv.PanicIfZeroOrNil(k.stakingKeeper, "stakingKeeper")                 // 9
	ccv.PanicIfZeroOrNil(k.slashingKeeper, "slashingKeeper")               // 10
	ccv.PanicIfZeroOrNil(k.distributionKeeper, "distributionKeeper")       // 11
	ccv.PanicIfZeroOrNil(k.bankKeeper, "bankKeeper")                       // 12
	ccv.PanicIfZeroOrNil(k.feeCollectorName, "feeCollectorName")           // 13
	ccv.PanicIfZeroOrNil(k.authority, "authority")                         // 14
	ccv.PanicIfZeroOrNil(k.validatorAddressCodec, "validatorAddressCodec") // 15
	ccv.PanicIfZeroOrNil(k.consensusAddressCodec, "consensusAddressCodec") // 16

	// this can be nil in tests
	// ccv.PanicIfZeroOrNil(k.govKeeper, "govKeeper")                         // 17
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+ibchost.ModuleName+"-"+types.ModuleName)
}

// IsBound checks if the CCV module is already bound to the desired port
// IBC v10: Capabilities removed, always return true as port binding is handled differently
func (k Keeper) IsBound(ctx sdk.Context, portID string) bool {
	// In IBC v10, port binding is handled internally
	return true
}

// IBC v10: BindPort removed following ICS v7 - using SetPort instead

// GetPort returns the portID for the CCV module. Used in ExportGenesis
func (k Keeper) GetPort(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	return string(store.Get(types.PortKey()))
}

// SetPort sets the portID for the CCV module. Used in InitGenesis
func (k Keeper) SetPort(ctx sdk.Context, portID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.PortKey(), []byte(portID))
}

// IBC v10: Capability methods removed - no longer needed

// SetChainToChannel sets the mapping from a consumer chainID to the CCV channel ID for that consumer chain.
func (k Keeper) SetChainToChannel(ctx sdk.Context, chainID, channelID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ChainToChannelKey(chainID), []byte(channelID))
}

// GetChainToChannel gets the CCV channelID for the given consumer chainID
func (k Keeper) GetChainToChannel(ctx sdk.Context, chainID string) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ChainToChannelKey(chainID))
	if bz == nil {
		return "", false
	}
	return string(bz), true
}

// DeleteChainToChannel deletes the CCV channel ID for the given consumer chain ID
func (k Keeper) DeleteChainToChannel(ctx sdk.Context, chainID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.ChainToChannelKey(chainID))
}

// SetProposedConsumerChain stores a consumer chainId corresponding to a submitted consumer addition proposal
// This consumer chainId is deleted once the voting period for the proposal ends.
func (k Keeper) SetProposedConsumerChain(ctx sdk.Context, chainID string, proposalID uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ProposedConsumerChainKey(proposalID), []byte(chainID))
}

// GetProposedConsumerChain returns the proposed chainID for the given consumerAddition proposal ID.
// This method is only used for testing.
func (k Keeper) GetProposedConsumerChain(ctx sdk.Context, proposalID uint64) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	consumerChain := store.Get(types.ProposedConsumerChainKey(proposalID))
	if consumerChain != nil {
		return string(consumerChain), true
	}
	return "", false
}

// DeleteProposedConsumerChainInStore deletes the consumer chainID from store
// which is in gov consumerAddition proposal
func (k Keeper) DeleteProposedConsumerChainInStore(ctx sdk.Context, proposalID uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.ProposedConsumerChainKey(proposalID))
}

// GetAllProposedConsumerChainIDs returns the proposed chainID of all gov consumerAddition proposals that are still in the voting period.
func (k Keeper) GetAllProposedConsumerChainIDs(ctx sdk.Context) []types.ProposedChain {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{types.ProposedConsumerChainByteKey})
	defer iterator.Close()

	proposedChains := []types.ProposedChain{}
	for ; iterator.Valid(); iterator.Next() {
		proposalID, err := types.ParseProposedConsumerChainKey(types.ProposedConsumerChainByteKey, iterator.Key())
		if err != nil {
			panic(fmt.Errorf("proposed chains cannot be parsed: %w", err))
		}

		proposedChains = append(proposedChains, types.ProposedChain{
			ChainID:    string(iterator.Value()),
			ProposalID: proposalID,
		})

	}

	return proposedChains
}

// GetAllPendingConsumerChainIDs gets pending consumer chains have not reach spawn time
func (k Keeper) GetAllPendingConsumerChainIDs(ctx sdk.Context) []string {
	chainIDs := []string{}
	props := k.GetAllPendingConsumerAdditionProps(ctx)
	for _, prop := range props {
		chainIDs = append(chainIDs, prop.ChainId)
	}

	return chainIDs
}

// GetAllRegisteredConsumerChainIDs gets all of the consumer chain IDs, for which the provider module
// created IBC clients. Consumer chains with created clients are also referred to as registered.
//
// Note that the registered consumer chains are stored under keys with the following format:
// ChainToClientBytePrefix | chainID
// Thus, the returned array is in ascending order of chainIDs.
func (k Keeper) GetAllRegisteredConsumerChainIDs(ctx sdk.Context) []string {
	chainIDs := []string{}

	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{types.ChainToClientBytePrefix})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// remove 1 byte prefix from key to retrieve chainID
		chainID := string(iterator.Key()[1:])
		chainIDs = append(chainIDs, chainID)
	}

	return chainIDs
}

// SetChannelToChain sets the mapping from the CCV channel ID to the consumer chainID.
func (k Keeper) SetChannelToChain(ctx sdk.Context, channelID, chainID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ChannelToChainKey(channelID), []byte(chainID))
}

// GetChannelToChain gets the consumer chainID for a given CCV channelID
func (k Keeper) GetChannelToChain(ctx sdk.Context, channelID string) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ChannelToChainKey(channelID))
	if bz == nil {
		return "", false
	}
	return string(bz), true
}

// DeleteChannelToChain deletes the consumer chain ID for a given CCV channelID
func (k Keeper) DeleteChannelToChain(ctx sdk.Context, channelID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.ChannelToChainKey(channelID))
}

// GetAllChannelToChains gets all channel to chain mappings. If a mapping exists,
// then the CCV channel to that consumer chain is established.
//
// Note that mapping from CCV channel IDs to consumer chainIDs
// is stored under keys with the following format:
// ChannelToChainBytePrefix | channelID
// Thus, the returned array is in ascending order of channelIDs.
func (k Keeper) GetAllChannelToChains(ctx sdk.Context) (channels []types.ChannelToChain) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{types.ChannelToChainBytePrefix})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// remove prefix from key to retrieve channelID
		channelID := string(iterator.Key()[1:])
		chainID := string(iterator.Value())

		channels = append(channels, types.ChannelToChain{
			ChannelId: channelID,
			ChainId:   chainID,
		})
	}

	return channels
}

func (k Keeper) SetConsumerGenesis(ctx sdk.Context, chainID string, gen ccv.ConsumerGenesisState) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := gen.Marshal()
	if err != nil {
		return err
	}
	store.Set(types.ConsumerGenesisKey(chainID), bz)

	return nil
}

func (k Keeper) GetConsumerGenesis(ctx sdk.Context, chainID string) (ccv.ConsumerGenesisState, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ConsumerGenesisKey(chainID))
	if bz == nil {
		return ccv.ConsumerGenesisState{}, false
	}

	var data ccv.ConsumerGenesisState
	if err := data.Unmarshal(bz); err != nil {
		// An error here would indicate something is very wrong,
		// the ConsumerGenesis is assumed to be correctly serialized in SetConsumerGenesis.
		panic(fmt.Errorf("consumer genesis could not be unmarshaled: %w", err))
	}
	return data, true
}

func (k Keeper) DeleteConsumerGenesis(ctx sdk.Context, chainID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.ConsumerGenesisKey(chainID))
}

// VerifyConsumerChain verifies that the chain trying to connect on the channel handshake
// is the expected consumer chain.
func (k Keeper) VerifyConsumerChain(ctx sdk.Context, channelID string, connectionHops []string) error {
	if len(connectionHops) != 1 {
		return errorsmod.Wrap(channeltypes.ErrTooManyConnectionHops, "must have direct connection to provider chain")
	}
	connectionID := connectionHops[0]
	clientID, tmClient, err := k.getUnderlyingClient(ctx, connectionID)
	if err != nil {
		return err
	}
	ccvClientId, found := k.GetConsumerClientId(ctx, tmClient.ChainId)
	if !found {
		return errorsmod.Wrapf(ccv.ErrClientNotFound, "cannot find client for consumer chain %s", tmClient.ChainId)
	}
	if ccvClientId != clientID {
		return errorsmod.Wrapf(types.ErrInvalidConsumerClient, "CCV channel must be built on top of CCV client. expected %s, got %s", ccvClientId, clientID)
	}

	// Verify that there isn't already a CCV channel for the consumer chain
	if prevChannel, ok := k.GetChainToChannel(ctx, tmClient.ChainId); ok {
		return errorsmod.Wrapf(ccv.ErrDuplicateChannel, "CCV channel with ID: %s already created for consumer chain %s", prevChannel, tmClient.ChainId)
	}
	return nil
}

// SetConsumerChain ensures that the consumer chain has not already been
// set by a different channel, and then sets the consumer chain mappings
// in keeper, and set the channel status to validating.
// If there is already a CCV channel between the provider and consumer
// chain then close the channel, so that another channel can be made.
//
// SetConsumerChain is called by OnChanOpenConfirm.
func (k Keeper) SetConsumerChain(ctx sdk.Context, channelID string) error {
	channel, ok := k.channelKeeper.GetChannel(ctx, ccv.ProviderPortID, channelID)
	if !ok {
		return errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "channel not found for channel ID: %s", channelID)
	}
	if len(channel.ConnectionHops) != 1 {
		return errorsmod.Wrap(channeltypes.ErrTooManyConnectionHops, "must have direct connection to consumer chain")
	}
	connectionID := channel.ConnectionHops[0]
	clientID, tmClient, err := k.getUnderlyingClient(ctx, connectionID)
	if err != nil {
		return err
	}
	// Verify that there isn't already a CCV channel for the consumer chain
	chainID := tmClient.ChainId
	if prevChannelID, ok := k.GetChainToChannel(ctx, chainID); ok {
		return errorsmod.Wrapf(ccv.ErrDuplicateChannel, "CCV channel with ID: %s already created for consumer chain %s", prevChannelID, chainID)
	}

	// the CCV channel is established:
	// - set channel mappings
	k.SetChainToChannel(ctx, chainID, channelID)
	k.SetChannelToChain(ctx, channelID, chainID)
	// - set current block height for the consumer chain initialization
	k.SetInitChainHeight(ctx, chainID, uint64(ctx.BlockHeight()))

	// emit event on successful addition
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			ccv.EventTypeChannelEstablished,
			sdk.NewAttribute(sdk.AttributeKeyModule, consumertypes.ModuleName),
			sdk.NewAttribute(ccv.AttributeChainID, chainID),
			sdk.NewAttribute(conntypes.AttributeKeyClientID, clientID),
			sdk.NewAttribute(channeltypes.AttributeKeyChannelID, channelID),
			sdk.NewAttribute(conntypes.AttributeKeyConnectionID, connectionID),
		),
	)
	return nil
}

// Retrieves the underlying client state corresponding to a connection ID.
func (k Keeper) getUnderlyingClient(ctx sdk.Context, connectionID string) (
	clientID string, tmClient *ibctmtypes.ClientState, err error,
) {
	conn, ok := k.connectionKeeper.GetConnection(ctx, connectionID)
	if !ok {
		return "", nil, errorsmod.Wrapf(conntypes.ErrConnectionNotFound,
			"connection not found for connection ID: %s", connectionID)
	}
	clientID = conn.ClientId
	clientState, ok := k.clientKeeper.GetClientState(ctx, clientID)
	if !ok {
		return "", nil, errorsmod.Wrapf(clienttypes.ErrClientNotFound,
			"client not found for client ID: %s", conn.ClientId)
	}
	tmClient, ok = clientState.(*ibctmtypes.ClientState)
	if !ok {
		return "", nil, errorsmod.Wrapf(clienttypes.ErrInvalidClientType,
			"invalid client type. expected %s, got %s", ibchost.Tendermint, clientState.ClientType())
	}
	return clientID, tmClient, nil
}

// chanCloseInit defines a wrapper function for the channel Keeper's function
// IBC v10: Capabilities have been removed
func (k Keeper) chanCloseInit(ctx sdk.Context, channelID string) error {
	return k.channelKeeper.ChanCloseInit(ctx, ccv.ProviderPortID, channelID)
}

func (k Keeper) IncrementValidatorSetUpdateId(ctx sdk.Context) {
	validatorSetUpdateId := k.GetValidatorSetUpdateId(ctx)
	k.SetValidatorSetUpdateId(ctx, validatorSetUpdateId+1)
}

func (k Keeper) SetValidatorSetUpdateId(ctx sdk.Context, valUpdateID uint64) {
	store := ctx.KVStore(k.storeKey)

	// Convert back into bytes for storage
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, valUpdateID)

	store.Set(types.ValidatorSetUpdateIdKey(), bz)
}

func (k Keeper) GetValidatorSetUpdateId(ctx sdk.Context) (validatorSetUpdateId uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ValidatorSetUpdateIdKey())

	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// SetValsetUpdateBlockHeight sets the block height for a given valset update id
func (k Keeper) SetValsetUpdateBlockHeight(ctx sdk.Context, valsetUpdateId, blockHeight uint64) {
	store := ctx.KVStore(k.storeKey)
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, blockHeight)
	store.Set(types.ValsetUpdateBlockHeightKey(valsetUpdateId), heightBytes)
}

// GetValsetUpdateBlockHeight gets the block height for a given valset update id
func (k Keeper) GetValsetUpdateBlockHeight(ctx sdk.Context, valsetUpdateId uint64) (uint64, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ValsetUpdateBlockHeightKey(valsetUpdateId))
	if bz == nil {
		return 0, false
	}
	return binary.BigEndian.Uint64(bz), true
}

// GetAllValsetUpdateBlockHeights gets all the block heights for all valset updates
//
// Note that the mapping from vscIDs to block heights is stored under keys with the following format:
// ValsetUpdateBlockHeightBytePrefix | vscID
// Thus, the returned array is in ascending order of vscIDs.
func (k Keeper) GetAllValsetUpdateBlockHeights(ctx sdk.Context) (valsetUpdateBlockHeights []types.ValsetUpdateIdToHeight) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{types.ValsetUpdateBlockHeightBytePrefix})

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		valsetUpdateId := binary.BigEndian.Uint64(iterator.Key()[1:])
		height := binary.BigEndian.Uint64(iterator.Value())

		valsetUpdateBlockHeights = append(valsetUpdateBlockHeights, types.ValsetUpdateIdToHeight{
			ValsetUpdateId: valsetUpdateId,
			Height:         height,
		})
	}

	return valsetUpdateBlockHeights
}

// DeleteValsetUpdateBlockHeight deletes the block height value for a given vaset update id
func (k Keeper) DeleteValsetUpdateBlockHeight(ctx sdk.Context, valsetUpdateId uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.ValsetUpdateBlockHeightKey(valsetUpdateId))
}

// SetSlashAcks sets the slash acks under the given chain ID
//
// TODO: SlashAcks should be persisted as a list of ConsumerConsAddr types, not strings.
// See https://github.com/cosmos/interchain-security/issues/728
func (k Keeper) SetSlashAcks(ctx sdk.Context, chainID string, acks []string) {
	store := ctx.KVStore(k.storeKey)

	sa := types.SlashAcks{
		Addresses: acks,
	}
	bz, err := sa.Marshal()
	if err != nil {
		// An error here would indicate something is very wrong,
		// sa is instantiated in this method and should be able to be marshaled.
		panic(fmt.Errorf("failed to marshal SlashAcks: %w", err))
	}
	store.Set(types.SlashAcksKey(chainID), bz)
}

// GetSlashAcks returns the slash acks stored under the given chain ID
//
// TODO: SlashAcks should be persisted as a list of ConsumerConsAddr types, not strings.
// See https://github.com/cosmos/interchain-security/issues/728
func (k Keeper) GetSlashAcks(ctx sdk.Context, chainID string) []string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.SlashAcksKey(chainID))
	if bz == nil {
		return nil
	}
	var acks types.SlashAcks
	if err := acks.Unmarshal(bz); err != nil {
		// An error here would indicate something is very wrong,
		// the SlashAcks are assumed to be correctly serialized in SetSlashAcks.
		panic(fmt.Errorf("failed to unmarshal SlashAcks: %w", err))
	}

	return acks.GetAddresses()
}

// ConsumeSlashAcks empties and returns the slash acks for a given chain ID
func (k Keeper) ConsumeSlashAcks(ctx sdk.Context, chainID string) (acks []string) {
	acks = k.GetSlashAcks(ctx, chainID)
	if len(acks) < 1 {
		return
	}
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.SlashAcksKey(chainID))
	return
}

// DeleteSlashAcks deletes the slash acks for a given chain ID
func (k Keeper) DeleteSlashAcks(ctx sdk.Context, chainID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.SlashAcksKey(chainID))
}

// AppendSlashAck appends the given slash ack to the given chain ID slash acks in store
func (k Keeper) AppendSlashAck(ctx sdk.Context, chainID,
	ack string, // TODO: consumer cons addr should be accepted here, see https://github.com/cosmos/interchain-security/issues/728
) {
	acks := k.GetSlashAcks(ctx, chainID)
	acks = append(acks, ack)
	k.SetSlashAcks(ctx, chainID, acks)
}

// SetInitChainHeight sets the provider block height when the given consumer chain was initiated
func (k Keeper) SetInitChainHeight(ctx sdk.Context, chainID string, height uint64) {
	store := ctx.KVStore(k.storeKey)
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)

	store.Set(types.InitChainHeightKey(chainID), heightBytes)
}

// GetInitChainHeight returns the provider block height when the given consumer chain was initiated
func (k Keeper) GetInitChainHeight(ctx sdk.Context, chainID string) (uint64, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.InitChainHeightKey(chainID))
	if bz == nil {
		return 0, false
	}

	return binary.BigEndian.Uint64(bz), true
}

// DeleteInitChainHeight deletes the block height value for which the given consumer chain's channel was established
func (k Keeper) DeleteInitChainHeight(ctx sdk.Context, chainID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.InitChainHeightKey(chainID))
}

// GetPendingVSCPackets returns the list of pending ValidatorSetChange packets stored under chain ID
func (k Keeper) GetPendingVSCPackets(ctx sdk.Context, chainID string) []ccv.ValidatorSetChangePacketData {
	var packets types.ValidatorSetChangePackets

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PendingVSCsKey(chainID))
	if bz == nil {
		return []ccv.ValidatorSetChangePacketData{}
	}
	if err := packets.Unmarshal(bz); err != nil {
		// An error here would indicate something is very wrong,
		// the PendingVSCPackets are assumed to be correctly serialized in AppendPendingVSCPackets.
		panic(fmt.Errorf("cannot unmarshal pending validator set changes: %w", err))
	}
	return packets.GetList()
}

// AppendPendingVSCPackets adds the given ValidatorSetChange packet to the list
// of pending ValidatorSetChange packets stored under chain ID
func (k Keeper) AppendPendingVSCPackets(ctx sdk.Context, chainID string, newPackets ...ccv.ValidatorSetChangePacketData) {
	pds := append(k.GetPendingVSCPackets(ctx, chainID), newPackets...)

	store := ctx.KVStore(k.storeKey)
	packets := types.ValidatorSetChangePackets{List: pds}
	buf, err := packets.Marshal()
	if err != nil {
		// An error here would indicate something is very wrong,
		// packets is instantiated in this method and should be able to be marshaled.
		panic(fmt.Errorf("cannot marshal pending validator set changes: %w", err))
	}
	store.Set(types.PendingVSCsKey(chainID), buf)
}

// DeletePendingVSCPackets deletes the list of pending ValidatorSetChange packets for chain ID
func (k Keeper) DeletePendingVSCPackets(ctx sdk.Context, chainID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.PendingVSCsKey(chainID))
}

// SetConsumerClientId sets the client ID for the given chain ID
func (k Keeper) SetConsumerClientId(ctx sdk.Context, chainID, clientID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ChainToClientKey(chainID), []byte(clientID))
}

// GetConsumerClientId returns the client ID for the given chain ID.
func (k Keeper) GetConsumerClientId(ctx sdk.Context, chainID string) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	clientIdBytes := store.Get(types.ChainToClientKey(chainID))
	if clientIdBytes == nil {
		return "", false
	}
	return string(clientIdBytes), true
}

// DeleteConsumerClientId removes from the store the clientID for the given chainID.
func (k Keeper) DeleteConsumerClientId(ctx sdk.Context, chainID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.ChainToClientKey(chainID))
}

// SetSlashLog updates validator's slash log for a consumer chain
// If an entry exists for a given validator address, at least one
// double signing slash packet was received by the provider from at least one consumer chain
func (k Keeper) SetSlashLog(
	ctx sdk.Context,
	providerAddr types.ProviderConsAddress,
) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.SlashLogKey(providerAddr), []byte{})
}

// GetSlashLog returns a validator's slash log status
// True will be returned if an entry exists for a given validator address
func (k Keeper) GetSlashLog(
	ctx sdk.Context,
	providerAddr types.ProviderConsAddress,
) (found bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.SlashLogKey(providerAddr))
	return bz != nil
}

func (k Keeper) BondDenom(ctx sdk.Context) (string, error) {
	return k.stakingKeeper.BondDenom(ctx)
}

func (k Keeper) GetAllRegisteredAndProposedChainIDs(ctx sdk.Context) []string {
	allConsumerChains := []string{}
	allConsumerChains = append(allConsumerChains, k.GetAllRegisteredConsumerChainIDs(ctx)...)
	proposedChains := k.GetAllProposedConsumerChainIDs(ctx)
	for _, proposedChain := range proposedChains {
		allConsumerChains = append(allConsumerChains, proposedChain.ChainID)
	}
	pendingChainIDs := k.GetAllPendingConsumerChainIDs(ctx)
	allConsumerChains = append(allConsumerChains, pendingChainIDs...)

	return allConsumerChains
}

func (k Keeper) UnbondingCanComplete(ctx sdk.Context, id uint64) error {
	return k.stakingKeeper.UnbondingCanComplete(ctx, id)
}

func (k Keeper) UnbondingTime(ctx sdk.Context) (time.Duration, error) {
	return k.stakingKeeper.UnbondingTime(ctx)
}
