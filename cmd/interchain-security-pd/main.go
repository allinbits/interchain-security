package main

import (
	"fmt"
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	appparams "github.com/allinbits/interchain-security/app/params"
	app "github.com/allinbits/interchain-security/app/provider"
	"github.com/allinbits/interchain-security/cmd/interchain-security-pd/cmd"
)

func main() {
	appparams.SetAddressPrefixes("cosmos")
	rootCmd := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
