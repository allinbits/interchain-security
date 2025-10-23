package main

import (
	"time"

	gov "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

// start relayer so that all messages are relayed
func stepsStartRelayer() []Step {
	return []Step{
		{
			Action: StartRelayerAction{},
			State:  State{},
		},
	}
}

// submits a consumer-removal proposal and removes the chain
func stepsStopChain(consumerName string, propNumber uint) []Step {
	s := []Step{
		{
			Action: SubmitConsumerRemovalProposalAction{
				Chain:          ChainID("provi"),
				From:           ValidatorID("bob"),
				Deposit:        10000001,
				ConsumerChain:  ChainID(consumerName),
				StopTimeOffset: 0 * time.Millisecond,
			},
			State: State{
				ChainID("provi"): ChainState{
					ValBalances: &map[ValidatorID]uint{
						ValidatorID("bob"): 9489999999,
					},
					Proposals: &map[uint]Proposal{
						propNumber: ConsumerRemovalProposal{
							Deposit:  10000001,
							Chain:    ChainID(consumerName),
							StopTime: 0,
							Status:   gov.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD.String(),
						},
					},
					ConsumerChains: &map[ChainID]bool{"consu": true}, // consumer chain not yet removed
				},
			},
		},
		{
			Action: VoteGovProposalAction{
				Chain:      ChainID("provi"),
				From:       []ValidatorID{ValidatorID("alice"), ValidatorID("bob"), ValidatorID("carol")},
				Vote:       []string{"yes", "yes", "yes"},
				PropNumber: propNumber,
			},
			State: State{
				ChainID("provi"): ChainState{
					Proposals: &map[uint]Proposal{
						propNumber: ConsumerRemovalProposal{
							Deposit:  10000001,
							Chain:    ChainID(consumerName),
							StopTime: 0,
							Status:   gov.ProposalStatus_PROPOSAL_STATUS_PASSED.String(),
						},
					},
					ValBalances: &map[ValidatorID]uint{
						ValidatorID("bob"): 9500000000,
					},
					ConsumerChains: &map[ChainID]bool{}, // Consumer chain is now removed
				},
			},
		},
	}

	return s
}

// submits a consumer-removal proposal and votes no on it
// the chain should not be removed
func stepsConsumerRemovalPropNotPassing(consumerName string, propNumber uint) []Step {
	s := []Step{
		{
			Action: SubmitConsumerRemovalProposalAction{
				Chain:          ChainID("provi"),
				From:           ValidatorID("bob"),
				Deposit:        10000001,
				ConsumerChain:  ChainID(consumerName),
				StopTimeOffset: 0 * time.Millisecond,
			},
			State: State{
				ChainID("provi"): ChainState{
					ValBalances: &map[ValidatorID]uint{
						ValidatorID("bob"): 9489999999,
					},
					Proposals: &map[uint]Proposal{
						propNumber: ConsumerRemovalProposal{
							Deposit:  10000001,
							Chain:    ChainID(consumerName),
							StopTime: 0,
							Status:   gov.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD.String(),
						},
					},
					ConsumerChains: &map[ChainID]bool{"consu": true}, // consumer chain not removed
				},
			},
		},
		{
			Action: VoteGovProposalAction{
				Chain:      ChainID("provi"),
				From:       []ValidatorID{ValidatorID("alice"), ValidatorID("bob"), ValidatorID("carol")},
				Vote:       []string{"no", "no", "no"},
				PropNumber: propNumber,
			},
			State: State{
				ChainID("provi"): ChainState{
					Proposals: &map[uint]Proposal{
						propNumber: ConsumerRemovalProposal{
							Deposit:  10000001,
							Chain:    ChainID(consumerName),
							StopTime: 0,
							Status:   gov.ProposalStatus_PROPOSAL_STATUS_REJECTED.String(),
						},
					},
					ValBalances: &map[ValidatorID]uint{
						ValidatorID("bob"): 9500000000,
					},
					ConsumerChains: &map[ChainID]bool{"consu": true}, // consumer chain not removed
				},
			},
		},
	}

	return s
}

// ICS1: Re-adds a consumer chain that was previously removed, reusing the existing IBC connection.
// This tests the connection reuse feature for standalone-to-consumer transitions.
// The consumerName should match a previously removed consumer chain that still has an active IBC connection.
func stepsReAddConsumerWithConnectionReuse(consumerName string, propNumber uint) []Step {
	s := []Step{
		{
			Action: SubmitConsumerAdditionProposalAction{
				Chain:         ChainID("provi"),
				From:          ValidatorID("alice"),
				Deposit:       10000001,
				ConsumerChain: ChainID(consumerName),
				SpawnTime:     0,
				InitialHeight: clienttypes.Height{RevisionNumber: 0, RevisionHeight: 1},
				TopN:          100, // All validators must validate (100% = Replicated Security)
				// ICS1: Reuse connection-0 that was created when the consumer was first added
				ConnectionId: "connection-0",
			},
			State: State{
				ChainID("provi"): ChainState{
					// Don't check balances - they vary due to gas costs from previous operations
					Proposals: &map[uint]Proposal{
						propNumber: ConsumerAdditionProposal{
							Deposit:       10000001,
							Chain:         ChainID(consumerName),
							SpawnTime:     0,
							InitialHeight: clienttypes.Height{RevisionNumber: 0, RevisionHeight: 1},
							Status:        gov.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD.String(),
						},
					},
					ProposedConsumerChains: &[]string{consumerName},
				},
			},
		},
		{
			Action: VoteGovProposalAction{
				Chain:      ChainID("provi"),
				From:       []ValidatorID{ValidatorID("alice"), ValidatorID("bob"), ValidatorID("carol")},
				Vote:       []string{"yes", "yes", "yes"},
				PropNumber: propNumber,
			},
			State: State{
				ChainID("provi"): ChainState{
					Proposals: &map[uint]Proposal{
						propNumber: ConsumerAdditionProposal{
							Deposit:       10000001,
							Chain:         ChainID(consumerName),
							SpawnTime:     0,
							InitialHeight: clienttypes.Height{RevisionNumber: 0, RevisionHeight: 1},
							Status:        gov.ProposalStatus_PROPOSAL_STATUS_PASSED.String(),
						},
					},
					// Don't check balances - they vary due to gas costs from previous operations
					// Consumer chain is re-added
					ConsumerChains: &map[ChainID]bool{ChainID(consumerName): true},
				},
			},
		},
	}

	return s
}
