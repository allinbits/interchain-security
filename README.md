# Interchain Security - AtomOne Fork

**ICS1 - A fresh implementation of Interchain Security for AtomOne**

Forked from Cosmos Interchain Security v5.2.0 and reimagined for the AtomOne ecosystem. This is ICS1 - a new baseline, not an incremental update.

## Modifications from Upstream v5.2.0

This fork includes the following changes from the upstream Cosmos Interchain Security v5.2.0:

1. **AtomOne SDK Integration** - Uses AtomOne's fork of Cosmos SDK v0.50.14-atomone.1 with custom governance
2. **IBC v10 Upgrade** - Migrated from IBC v8 to v10 for improved interchain communication
3. **VSCMatured Packet Removal** - Optimization that removes VSCMatured packets from provider and consumer chains
4. **Module Namespace Change** - Updated from `cosmos` to `allinbits` namespace
5. **Lightweight PSS** - Retains TopN, opt-in/out, allowlist/denylist, and power capping for governance control
6. **Removed Advanced PSS** - Power shaping and consumer-specific commission rates removed
7. **IBC Connection Reuse** - Support for reusing IBC connections during standalone→consumer chain transitions
8. **IBC Transfer Memos** - Added memo support to IBC transfers for reward attribution
9. **Consumer Chain-ID Updates** - Support for updating consumer chain identifiers
10. **Removed Legacy Code** - Removed v4→v5 and v5→v6 migration code and x/crisis module

See [CHANGELOG.md](./CHANGELOG.md) for detailed technical changes.

## AtomOne Constitution

This implementation aligns with the AtomOne Constitution's goals for Interchain Security:

- Every validator is compensated for running ICS consumer chains
- Hub remains minimal with clear separation between core shards and consumer chains
- All ICS zones must be profitable to validators
- Governance-controlled validator set management

For more details, see the [AtomOne Constitution](https://github.com/atomone-hub/genesis).

## Prerequisites

- Go 1.24 or later
- jq (optional, for testnet)
- Docker (optional, for integration tests)

## Installation

```bash
# Install interchain-security-pd and interchain-security-cd binaries
make install

# Run provider
interchain-security-pd

# Run consumer
interchain-security-cd

# If the above fail, ensure ~/go/bin is on $PATH
export PATH=$PATH:$(go env GOPATH)/bin
```

## Testing

See [TESTING.md](./TESTING.md) for detailed testing instructions.

```bash
# Run unit tests
make test-unit

# Run integration tests
make test-integration

# Build everything
go build ./...
```

## Attribution

This software is based on [Cosmos Interchain Security](https://github.com/cosmos/interchain-security) v5.2.0.

## Learn More

- [IBC Specifications](https://github.com/cosmos/ibc)
- [AtomOne Genesis](https://github.com/atomone-hub/genesis)
- [AtomOne SDK](https://github.com/atomone-hub/cosmos-sdk)
- [Upstream ICS Documentation](https://cosmos.github.io/interchain-security/)
