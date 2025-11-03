# CHANGELOG

## Unreleased

## v5.2.0-atomone-1

*In Development*

**ICS1 - A fresh implementation of Interchain Security for AtomOne**

Forked from Cosmos Interchain Security v5.2.0 and reimagined for the AtomOne ecosystem. This is ICS1 - a new baseline, not an incremental update.

### Base Configuration

- **AtomOne SDK v0.50.14-atomone.1** - Custom governance and minimal hub philosophy
- **IBC v10** - Modern IBC protocol for interchain communication
- **Go 1.24.5** - Latest Go toolchain with improved type safety

### Features

**Lightweight Partial Set Security (PSS)**

- TopN parameter for governance-controlled validator set management
- Validator opt-in/opt-out capabilities
- Allowlist/denylist support per consumer chain
- Power capping for consumer chains
- Removes complex power shaping and consumer commission rates

**Consumer Chain Management**

- Consumer chain-id updates ([#10](https://github.com/allinbits/interchain-security/pull/10))
- IBC connection reuse for standalone→consumer transitions ([#11](https://github.com/allinbits/interchain-security/pull/11))
- IBC transfer memos for reward attribution ([#13](https://github.com/allinbits/interchain-security/pull/13))

**Optimizations**

- VSCMatured packet removal - reduces IBC traffic without affecting security ([#6](https://github.com/allinbits/interchain-security/pull/6), [#7](https://github.com/allinbits/interchain-security/pull/7))
- Removed legacy migration code (v4→v5, v5→v6)
- Removed x/crisis module dependency

**AtomOne Integration**

- Custom governance adapter for AtomOne's governance model ([#4](https://github.com/allinbits/interchain-security/pull/4))
- Module namespace: `allinbits` ([#14](https://github.com/allinbits/interchain-security/pull/14))

### Design Philosophy

ICS1 aligns with the AtomOne Constitution's principles:

- Every validator is compensated for running consumer chains
- Hub remains minimal with clear separation between core shards and consumer chains
- All ICS zones must be profitable to validators
- Governance-controlled validator set management

### Implementation Details

This release implements lightweight PSS (TopN, opt-in/out, allowlist/denylist, power capping) for governance flexibility while removing advanced PSS features that add unnecessary complexity.

The VSCMatured packet removal is an optimization that reduces IBC traffic. The provider and consumer no longer need to acknowledge validator set changes, simplifying the state machine without compromising security guarantees.

#### Stacked Sub-PRs

This release consists of the following pull requests:

- [#3](https://github.com/allinbits/interchain-security/pull/3) - IBC v10 upgrade
- [#4](https://github.com/allinbits/interchain-security/pull/4) - AtomOne governance compatibility adapter
- [#5](https://github.com/allinbits/interchain-security/pull/5) - Dependency alignment with AtomOne SDK
- [#6](https://github.com/allinbits/interchain-security/pull/6) - VSCMatured removal (provider)
- [#7](https://github.com/allinbits/interchain-security/pull/7) - VSCMatured removal (consumer) & legacy cleanup
- [#8](https://github.com/allinbits/interchain-security/pull/8) - E2E/integration test fixes
- [#9](https://github.com/allinbits/interchain-security/pull/9) - TopN for governance parameters
- [#10](https://github.com/allinbits/interchain-security/pull/10) - Consumer chain-id updates
- [#11](https://github.com/allinbits/interchain-security/pull/11) - Connection reuse for transitions
- [#13](https://github.com/allinbits/interchain-security/pull/13) - IBC transfer memos
- [#14](https://github.com/allinbits/interchain-security/pull/14) - Module naming (cosmos→allinbits)

---

## Fork Point: Cosmos ICS v5.2.0

ICS1 is based on [Cosmos Interchain Security v5.2.0](https://github.com/cosmos/interchain-security/releases/tag/v5.2.0) (September 4, 2024).

For upstream changelog history, see the [Cosmos ICS CHANGELOG](https://github.com/cosmos/interchain-security/blob/v5.2.0/CHANGELOG.md).
