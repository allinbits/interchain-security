package keeper

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v3/modules/core/23-commitment/types"
	ibctmtypes "github.com/cosmos/ibc-go/v3/modules/light-clients/07-tendermint/types"
	"github.com/cosmos/interchain-security/x/ccv/provider/types"
	abci "github.com/tendermint/tendermint/abci/types"

	consumertypes "github.com/cosmos/interchain-security/x/ccv/consumer/types"
)

// CreateConsumerChainProposal will receive the consumer chain's client state from the proposal.
// If the spawn time has already passed, then set the consumer chain. Otherwise store the client
// as a pending client, and set once spawn time has passed.
func (k Keeper) CreateConsumerChainProposal(ctx sdk.Context, p *types.CreateConsumerChainProposal) error {
	if ctx.BlockTime().After(p.SpawnTime) {
		return k.CreateConsumerClient(ctx, p.ChainId, p.InitialHeight, p.LockUnbondingOnTimeout)
	}

	k.SetPendingClientInfo(ctx, p)
	return nil
}

// StopConsumerChainProposal stops a consumer chain and released the outstanding unbonding operations.
// If the stop time hasn't already passed, it stores the proposal as a pending proposal.
func (k Keeper) StopConsumerChainProposal(ctx sdk.Context, p *types.StopConsumerChainProposal) error {
	if ctx.BlockTime().After(p.StopTime) {
		return k.StopConsumerChain(ctx, p.ChainId, false, true)
	}

	k.SetPendingStopProposal(ctx, p.ChainId, p.StopTime)
	return nil
}

// StopConsumerChain cleans up the states for the given consumer chain ID and, if the given lockUbd is false,
// it completes the outstanding unbonding operations lock by the consumer chain.
func (k Keeper) StopConsumerChain(ctx sdk.Context, chainID string, lockUbd, closeChan bool) (err error) {

	// clean up states
	k.DeleteConsumerClient(ctx, chainID)
	k.DeleteLockUnbondingOnTimeout(ctx, chainID)

	// close channel and delete the mappings between chain ID and channel ID
	if channelID, found := k.GetChainToChannel(ctx, chainID); found {
		if closeChan {
			k.CloseChannel(ctx, channelID)
		}
		k.chanCloseInit(ctx, channelID)
		k.DeleteChainToChannel(ctx, chainID)
		k.DeleteChannelToChain(ctx, channelID)
	}

	// TODO remove pending VSC packets once https://github.com/cosmos/interchain-security/issues/27 is fixed
	k.DeleteInitChainHeight(ctx, chainID)
	k.EmptySlashAcks(ctx, chainID)

	// release unbonding operations if they aren't locked
	if !lockUbd {
		// iterate over the consumer chain's unbonding operation VSC ids
		k.IterateOverUnbondingOpIndex(ctx, chainID, func(vscID uint64, ids []uint64) bool {
			// range over the unbonding operations for the current VSC ID
			for _, id := range ids {
				unbondingOp, found := k.GetUnbondingOp(ctx, id)
				if !found {
					err = fmt.Errorf("could not find UnbondingOp according to index - id: %d", id)
					return false
				}
				// remove consumer chain ID from unbonding op record
				unbondingOp.UnbondingConsumerChains, _ = removeStringFromSlice(unbondingOp.UnbondingConsumerChains, chainID)

				// If unbonding op is completely unbonded from all relevant consumer chains
				if len(unbondingOp.UnbondingConsumerChains) == 0 {
					// Attempt to complete unbonding in staking module
					err = k.stakingKeeper.UnbondingCanComplete(ctx, unbondingOp.Id)
					if err != nil {
						return false
					}
					// Delete unbonding op
					k.DeleteUnbondingOp(ctx, unbondingOp.Id)
				} else {
					k.SetUnbondingOp(ctx, unbondingOp)
				}
			}
			// clean up index
			k.DeleteUnbondingOpIndex(ctx, chainID, vscID)
			return true
		})
	}
	if err != nil {
		return err
	}

	return nil
}

// CreateConsumerClient will create the CCV client for the given consumer chain. The CCV channel must be built
// on top of the CCV client to ensure connection with the right consumer chain.
func (k Keeper) CreateConsumerClient(ctx sdk.Context, chainID string, initialHeight clienttypes.Height, lockUbdOnTimeout bool) error {
	unbondingTime := k.stakingKeeper.UnbondingTime(ctx)

	// create clientstate by getting template client from parameters and filling in zeroed fields from proposal.
	clientState := k.GetTemplateClient(ctx)
	clientState.ChainId = chainID
	clientState.LatestHeight = initialHeight
	clientState.TrustingPeriod = unbondingTime / 2
	clientState.UnbondingPeriod = unbondingTime

	// TODO: Allow for current validators to set different keys
	consensusState := ibctmtypes.NewConsensusState(ctx.BlockTime(), commitmenttypes.NewMerkleRoot([]byte(ibctmtypes.SentinelRoot)), ctx.BlockHeader().NextValidatorsHash)
	clientID, err := k.clientKeeper.CreateClient(ctx, clientState, consensusState)
	if err != nil {
		return err
	}

	k.SetConsumerClient(ctx, chainID, clientID)
	consumerGen, err := k.MakeConsumerGenesis(ctx)
	if err != nil {
		return err
	}

	k.SetConsumerGenesis(ctx, chainID, consumerGen)

	// store LockUnbondingOnTimeout flag
	if lockUbdOnTimeout {
		k.SetLockUnbondingOnTimeout(ctx, chainID)
	}
	return nil
}

func (k Keeper) MakeConsumerGenesis(ctx sdk.Context) (gen consumertypes.GenesisState, err error) {
	unbondingTime := k.stakingKeeper.UnbondingTime(ctx)
	height := clienttypes.GetSelfHeight(ctx)

	clientState := k.GetTemplateClient(ctx)
	clientState.ChainId = ctx.ChainID()
	clientState.LatestHeight = height //(+-1???)
	clientState.TrustingPeriod = unbondingTime / 2
	clientState.UnbondingPeriod = unbondingTime

	consState, err := k.clientKeeper.GetSelfConsensusState(ctx, height)
	if err != nil {
		return gen, sdkerrors.Wrapf(clienttypes.ErrConsensusStateNotFound, "error %s getting self consensus state for: %s", err, height)
	}

	gen = *consumertypes.DefaultGenesisState()

	gen.Params.Enabled = true
	gen.NewChain = true
	gen.ProviderClientState = clientState
	gen.ProviderConsensusState = consState.(*ibctmtypes.ConsensusState)

	var lastPowers []stakingtypes.LastValidatorPower

	k.stakingKeeper.IterateLastValidatorPowers(ctx, func(addr sdk.ValAddress, power int64) (stop bool) {
		lastPowers = append(lastPowers, stakingtypes.LastValidatorPower{Address: addr.String(), Power: power})
		return false
	})

	updates := []abci.ValidatorUpdate{}

	for _, p := range lastPowers {
		addr, err := sdk.ValAddressFromBech32(p.Address)
		if err != nil {
			panic(err)
		}

		val, found := k.stakingKeeper.GetValidator(ctx, addr)
		if !found {
			panic("Validator from LastValidatorPowers not found in staking keeper")
		}

		tmProtoPk, err := val.TmConsPublicKey()
		if err != nil {
			panic(err)
		}

		updates = append(updates, abci.ValidatorUpdate{
			PubKey: tmProtoPk,
			Power:  p.Power,
		})
	}

	gen.InitialValSet = updates

	return gen, nil
}

// SetPendingClientInfo sets the initial height for the given timestamp and chain ID
func (k Keeper) SetPendingClientInfo(ctx sdk.Context, clientInfo *types.CreateConsumerChainProposal) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(clientInfo)
	if err != nil {
		return err
	}
	store.Set(types.PendingClientKey(clientInfo.SpawnTime, clientInfo.ChainId), bz)
	return nil
}

// GetPendingClient gets the client pending info for the given timestamp and chain ID
func (k Keeper) GetPendingClientInfo(ctx sdk.Context, timestamp time.Time, chainID string) types.CreateConsumerChainProposal {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PendingClientKey(timestamp, chainID))
	if len(bz) == 0 {
		return types.CreateConsumerChainProposal{}
	}
	var clientInfo types.CreateConsumerChainProposal

	k.cdc.MustUnmarshal(bz, &clientInfo)
	return clientInfo
}

// IteratePendingClientInfo iterates over the pending client info in order and creates the consumer client if the spawn time has passed,
// otherwise it will break out of loop and return.
func (k Keeper) IteratePendingClientInfo(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.PendingClientKeyPrefix+"/"))
	defer iterator.Close()

	if !iterator.Valid() {
		return
	}

	for ; iterator.Valid(); iterator.Next() {
		suffixKey := iterator.Key()
		// splitKey contains the bigendian time in the first element and the chainID in the second element/
		splitKey := strings.Split(string(suffixKey), "/")

		timeNano := binary.BigEndian.Uint64([]byte(splitKey[0]))
		spawnTime := time.Unix(0, int64(timeNano))

		var clientInfo types.CreateConsumerChainProposal
		k.cdc.MustUnmarshal(iterator.Value(), &clientInfo)

		if ctx.BlockTime().After(spawnTime) {
			k.CreateConsumerClient(ctx, splitKey[1], clientInfo.InitialHeight, clientInfo.LockUnbondingOnTimeout)
		} else {
			break
		}
	}
}

// SetPendingStopProposal sets the consumer chain ID for the given timestamp
func (k Keeper) SetPendingStopProposal(ctx sdk.Context, chainID string, timestamp time.Time) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.PendingStopProposalKey(timestamp, chainID), []byte{})
}

// GetPendingStopProposal returns a boolean if a pending stop proposal exists for the given consumer chain ID and the timestamp
func (k Keeper) GetPendingStopProposal(ctx sdk.Context, chainID string, timestamp time.Time) bool {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PendingStopProposalKey(timestamp, chainID))

	return bz != nil
}

// DeletePendingStopProposal deletes the stop proposal for the given consumer chain ID and given timestamp
func (k Keeper) DeletePendingStopProposal(ctx sdk.Context, chainID string, timestamp time.Time) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.PendingStopProposalKey(timestamp, chainID))
}

// IteratePendingStopProposal iterates over the pending stop proposals in order and stop the chain if the stop time has passed,
// otherwise it will break out of loop and return.
func (k Keeper) IteratePendingStopProposal(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.PendingStopProposalKeyPrefix+"/"))
	defer iterator.Close()

	if !iterator.Valid() {
		return
	}

	for ; iterator.Valid(); iterator.Next() {
		suffixKey := iterator.Key()
		// splitKey contains the bigendian time in the first element and the chainID in the second element
		splitKey := strings.Split(string(suffixKey), "/")

		timeNano := binary.BigEndian.Uint64([]byte(splitKey[0]))
		stopTime := time.Unix(0, int64(timeNano))
		chainID := string([]byte(splitKey[1]))

		if ctx.BlockTime().After(stopTime) {
			k.StopConsumerChain(ctx, chainID, false, true)
			k.DeletePendingStopProposal(ctx, chainID, stopTime)
		} else {
			break
		}
	}
}

// CloseChannel closes the channel for the given channel ID on the condition
// that the channel exists and isn't already in the CLOSED state
func (k Keeper) CloseChannel(ctx sdk.Context, channelID string) {
	channel, found := k.channelKeeper.GetChannel(ctx, types.PortID, channelID)
	if found && channel.State != channeltypes.CLOSED {
		k.chanCloseInit(ctx, channelID)
	}
}
