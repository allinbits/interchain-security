package main

import (
	"fmt"
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	app "github.com/allinbits/interchain-security/app/consumer"
	appparams "github.com/allinbits/interchain-security/app/params"
	"github.com/allinbits/interchain-security/cmd/interchain-security-cd/cmd"
)

func main() {
	appparams.SetAddressPrefixes(app.Bech32MainPrefix)

	rootCmd := cmd.NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
