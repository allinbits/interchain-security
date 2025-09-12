package common

import (
	"context"

	"github.com/cosmos/cosmos-sdk/baseapp"
)

// SimpleVersionModifier implements the baseapp.VersionModifier interface
// required by AtomOne SDK v0.50.14 for ABCI queries to function properly.
//
// ICS1 E2E FIX: AtomOne SDK requires a VersionModifier for the baseapp to handle
// ABCI info queries correctly. Without this, Hermes relayer fails with
// "app.versionModifier is nil" error when creating IBC connections.
//
// This implementation returns protocol version 0, which is sufficient for testing
// and basic operations. Production deployments may want to implement proper
// version tracking if protocol upgrades are planned.
type SimpleVersionModifier struct{}

// Ensure SimpleVersionModifier implements the baseapp.VersionModifier interface
var _ baseapp.VersionModifier = (*SimpleVersionModifier)(nil)

// SetAppVersion sets the application protocol version.
// This implementation is a no-op as we don't need to track version changes for testing.
func (s SimpleVersionModifier) SetAppVersion(ctx context.Context, version uint64) error {
	// For testing purposes, we don't need to store the version
	return nil
}

// AppVersion returns the current application protocol version.
// Returns 0 as the default version, which is sufficient for testing.
func (s SimpleVersionModifier) AppVersion(ctx context.Context) (uint64, error) {
	return 0, nil
}
