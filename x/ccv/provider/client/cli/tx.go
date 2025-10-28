package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	ibctmtypes "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/allinbits/interchain-security/x/ccv/provider/types"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(NewAssignConsumerKeyCmd())
	cmd.AddCommand(NewSubmitConsumerMisbehaviourCmd())
	cmd.AddCommand(NewSubmitConsumerDoubleVotingCmd())
	cmd.AddCommand(NewConsumerModificationCmd())

	return cmd
}

func NewAssignConsumerKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assign-consensus-key [consumer-chain-id] [consumer-pubkey]",
		Short: "assign a consensus public key to use for a consumer chain",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			signer := clientCtx.GetFromAddress().String()
			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			txf = txf.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			providerValAddr := clientCtx.GetFromAddress()

			msg, err := types.NewMsgAssignConsumerKey(args[0], sdk.ValAddress(providerValAddr), args[1], signer)
			if err != nil {
				return err
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	_ = cmd.MarkFlagRequired(flags.FlagFrom)

	return cmd
}

func NewSubmitConsumerMisbehaviourCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-consumer-misbehaviour [misbehaviour]",
		Short: "submit an IBC misbehaviour for a consumer chain",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit an IBC misbehaviour detected on a consumer chain.
An IBC misbehaviour contains two conflicting IBC client headers, which are used to form a light client attack evidence.
The misbehaviour type definition can be found in the IBC client messages, see ibc-go/proto/ibc/core/client/v1/tx.proto.

Example:
%s tx provider submit-consumer-misbehaviour [path/to/misbehaviour.json] --from node0 --home ../node0 --chain-id $CID
			`, version.AppName)),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			txf = txf.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			submitter := clientCtx.GetFromAddress()
			misbJson, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}

			cdc := codec.NewProtoCodec(clientCtx.InterfaceRegistry)

			misbehaviour := ibctmtypes.Misbehaviour{}
			if err := cdc.UnmarshalJSON(misbJson, &misbehaviour); err != nil {
				return fmt.Errorf("misbehaviour unmarshalling failed: %s", err)
			}

			msg, err := types.NewMsgSubmitConsumerMisbehaviour(submitter, &misbehaviour)
			if err != nil {
				return err
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	_ = cmd.MarkFlagRequired(flags.FlagFrom)

	return cmd
}

func NewSubmitConsumerDoubleVotingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-consumer-double-voting [evidence] [infraction_header]",
		Short: "submit a double voting evidence for a consumer chain",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a Tendermint duplicate vote evidence detected on a consumer chain with
 the IBC light client header for the infraction height.
 The DuplicateVoteEvidence type definition can be found in the Tendermint messages,
 see cometbft/proto/tendermint/types/evidence.proto and the IBC header
 definition can be found in the IBC messages, see ibc-go/proto/ibc/lightclients/tendermint/v1/tendermint.proto.

Example:
%s tx provider submit-consumer-double-voting [path/to/evidence.json] [path/to/infraction_header.json] --from node0 --home ../node0 --chain-id $CID
`, version.AppName)),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			txf = txf.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			submitter := clientCtx.GetFromAddress()

			ev := tmproto.DuplicateVoteEvidence{}
			evidenceJson, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}

			if err := json.Unmarshal(evidenceJson, &ev); err != nil {
				return fmt.Errorf("duplicate vote evidence unmarshalling failed: %s", err)
			}

			headerJson, err := os.ReadFile(args[1])
			if err != nil {
				return err
			}

			cdc := codec.NewProtoCodec(clientCtx.InterfaceRegistry)

			header := ibctmtypes.Header{}
			if err := cdc.UnmarshalJSON(headerJson, &header); err != nil {
				return fmt.Errorf("infraction IBC header unmarshalling failed: %s", err)
			}

			msg, err := types.NewMsgSubmitConsumerDoubleVoting(submitter, &ev, &header)
			if err != nil {
				return err
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	_ = cmd.MarkFlagRequired(flags.FlagFrom)

	return cmd
}

func NewConsumerModificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consumer-modification [path/to/modification.json]",
		Short: "submit a governance consumer-modification (supports pre-launch rename via new_chain_id)",
		Long: strings.TrimSpace(fmt.Sprintf(`
Submit a governance proposal-like message to modify a consumer chain (TopN, caps, allow/deny lists),
and optionally rename its chain-id *before launch* using "new_chain_id".

Example:
  %s tx provider consumer-modification ./mod.json --from <key> --chain-id <CID>

mod.json schema (all fields optional except authority, chain_id):
{
  "title": "Update consumer params",
  "description": "Adjust caps; optionally rename pre-launch",
  "chain_id": "consumer-1",
  "top_N": 53,
  "validators_power_cap": 32,
  "validator_set_cap": 0,
  "allowlist": ["cosmosvalcons1..."],
  "denylist":  ["cosmosvalcons1..."],
  "authority": "cosmos1govacct...",   // governance authority address (required)
  "new_chain_id": "consumer-mainnet"  // optional; ignored after launch
}
`, version.AppName)),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			txf = txf.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			// Read user JSON
			raw, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}

			var in struct {
				Title              string   `json:"title"`
				Description        string   `json:"description"`
				ChainID            string   `json:"chain_id"`
				Top_N              uint32   `json:"top_N"`
				ValidatorsPowerCap uint32   `json:"validators_power_cap"`
				ValidatorSetCap    uint32   `json:"validator_set_cap"`
				Allowlist          []string `json:"allowlist"`
				Denylist           []string `json:"denylist"`
				Authority          string   `json:"authority"`
				NewChainID         string   `json:"new_chain_id"`
			}
			if err := json.Unmarshal(raw, &in); err != nil {
				return fmt.Errorf("modification data unmarshalling failed: %w", err)
			}

			// Basic checks that mirror your proto expectations
			if strings.TrimSpace(in.ChainID) == "" {
				return fmt.Errorf("chain_id cannot be empty")
			}
			if strings.TrimSpace(in.Authority) == "" {
				return fmt.Errorf("authority cannot be empty")
			}

			msg := &types.MsgConsumerModification{
				Title:              in.Title,
				Description:        in.Description,
				ChainId:            in.ChainID,
				Top_N:              in.Top_N,
				ValidatorsPowerCap: in.ValidatorsPowerCap,
				ValidatorSetCap:    in.ValidatorSetCap,
				Allowlist:          in.Allowlist,
				Denylist:           in.Denylist,
				Authority:          in.Authority,
				NewChainId:         in.NewChainID, // <-- your new field
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	_ = cmd.MarkFlagRequired(flags.FlagFrom)
	return cmd
}
