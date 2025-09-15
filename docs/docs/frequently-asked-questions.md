---
sidebar_position: 5
title: "Frequently Asked Questions"
slug: /faq
---

## What is a consumer chain?

Consumer chain is a blockchain operated by (a subset of) the validators of the provider chain. The ICS protocol ensures that the consumer chain gets information about which validators should run it (informs consumer chain about the current state of the validator set and the opted in validators for this consumer chain on the provider).

Consumer chains are run on infrastructure (virtual or physical machines) distinct from the provider, have their own configurations and operating requirements.

## What happens to consumer if provider is down?

In case the provider chain halts or experiences difficulties the consumer chain will keep operating - the provider chain and consumer chains represent different networks, which only share the validator set.

The consumer chain will not halt if the provider halts because they represent distinct networks and distinct infrastructures. Provider chain liveness does not impact consumer chain liveness.

However, if the `trusting_period` (currently 5 days for protocol safety reasons) elapses without receiving any updates from the provider, the consumer chain will essentially transition to a Proof of Authority chain.
This means that the validator set on the consumer will be the last validator set of the provider that the consumer knows about.

Steps to recover from this scenario and steps to "release" the validators from their duties will be specified at a later point.
At the very least, the consumer chain could replace the validator set, remove the ICS module and perform a genesis restart. The impact of this on the IBC clients and connections is currently under careful consideration.

## What happens to provider if consumer is down?

Consumer chains do not impact the provider chain.
The ICS protocol is concerned only with validator set management, and the only communication that the provider requires from the consumer is information about validator activity (essentially keeping the provider informed about slash events).

## Can I run the provider and consumer chains on the same machine?

Yes, but you should favor running them in separate environments so failure of one machine does not impact your whole operation.

## Can the consumer chain have its own token?

As any other cosmos-sdk chain the consumer chain can issue its own token, manage inflation parameters and use them to pay gas fees.

## How are Tx fees paid on consumer?

The consumer chain operates as any other cosmos-sdk chain. The ICS protocol does not impact the normal chain operations.

## Are there any restrictions the consumer chains need to abide by?

No. Consumer chains are free to choose how they wish to operate, which modules to include, use CosmWASM in a permissioned or a permissionless way.
The only thing that separates consumer chains from standalone chains is that they share their validator set with the provider chain.

## What's in it for the validators and stakers?

The consumer chains sends a portion of its fees and inflation as reward to the provider chain as defined by `ConsumerRedistributionFraction`. The rewards are distributed (sent to the provider) every `BlocksPerDistributionTransmission`.

:::note
  `ConsumerRedistributionFraction` and `BlocksPerDistributionTransmission` are parameters defined in the `ConsumerAdditionProposal` used to create the consumer chain. These parameters can be changed via consumer chain governance.
:::

## Can the consumer chain have its own governance?

**Yes.**

In that case the validators are not necessarily part of the governance structure. Instead, their place in governance is replaced by "representatives" (governors). The representatives do not need to run validators, they simply represent the interests of a particular interest group on the consumer chain.

Validators can also be representatives but representatives are not required to run validator nodes.

This feature discerns between validator operators (infrastructure) and governance representatives which further democratizes the ecosystem. This also reduces the pressure on validators to be involved in on-chain governance.

## Can validators opt out of validating a consumer chain?

No. In the AtomOne ICS1 Replicated Security model, all bonded validators must validate all consumer chains. There is no opt-in or opt-out mechanism.

## How does Slashing work?

Validators that perform an equivocation or a light-client attack on a consumer chain are slashed on the provider chain.
We achieve this by submitting the proof of the equivocation or the light-client attack to the provider chain (see [slashing](features/slashing.md)).

## Can Consumer Chains perform Software Upgrades?

Consumer chains are standalone chains, in the sense that they can run arbitrary logic and use any modules they want (ie CosmWASM).

Consumer chain upgrades are unlikely to impact the provider chain, as long as there are no changes to the ICS module.

## How can I connect to the testnets?

Check out the [Joining Interchain Security testnet](./validators/joining-testnet.md) section.

## How do I start using ICS?

To become a consumer chain use this [checklist](./consumer-development/onboarding.md) and check the [App integration section](./consumer-development/app-integration.md)

## Which relayers are supported?

Currently supported versions:

- Hermes 1.8.0

## How does key delegation work in ICS?

You can check the [Key Assignment Guide](./features/key-assignment.md) for specific instructions.

## What is Replicated Security in AtomOne ICS1?

AtomOne ICS1 implements pure Replicated Security, where ALL bonded validators from the provider chain must validate ALL consumer chains. This ensures maximum security for consumer chains as they receive the full security of the provider chain's validator set. Unlike Partial Set Security variants, there is no opt-in/opt-out mechanism - participation is mandatory for all validators.

## How does a validator know which consumers chains it has to validate?

In AtomOne ICS1's Replicated Security model, validators must validate ALL consumer chains. There is no need for complex queries as participation is mandatory for all bonded validators on the provider chain.

## How many chains must a validator validate?

Validators must validate ALL consumer chains in the Replicated Security model. There is no opt-in mechanism - participation is mandatory.

## Can validators assign a consensus keys while a consumer-addition proposal is in voting period?
Yes, see the [Key Assignment Guide](./features/key-assignment.md) for more information.

## Can validators assign a consensus key during the voting period for a consumer-addition proposal?
Yes, all validators can assign consensus keys during the voting period.

## Do validators need to take any action before a consumer chain launches?
Validators should ensure they are prepared to validate the consumer chain when it launches. All bonded validators will automatically be included in the consumer chain's validator set.

## How are validator commissions handled in Replicated Security?
All validators participate in all consumer chains and receive rewards according to the consumer chain's reward distribution parameters.

## Can a consumer chain modify its parameters?
Yes, by issuing a [`ConsumerModificationProposal`](./features/proposals.md#consumermodificationproposal), though power shaping and opt-in features are not available in the Replicated Security model.