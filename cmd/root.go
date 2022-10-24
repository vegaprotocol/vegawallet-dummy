package cmd

import (
	"os"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"github.com/spf13/cobra"
)

var rootExamples = cli.Examples(`
	# Specify a custom Vega home directory
	vegawallet-dummy --home PATH_TO_DIR COMMAND
`)

func NewCmdRoot() *cobra.Command {
	return BuildCmdRoot()
}

func BuildCmdRoot() *cobra.Command {
	rf := &RootFlags{}

	cmd := &cobra.Command{
		Use:          os.Args[0],
		Short:        "The dummy Vega wallet for development and testing",
		Long:         "FOR DEVELOPMENT AND TESTING ONLY!",
		Example:      rootExamples,
		SilenceUsage: true,
	}

	cmd.PersistentFlags().StringVar(
		&rf.Home,
		"home",
		"",
		"Specify the location of a custom Vega home",
	)

	_ = cmd.MarkPersistentFlagDirname("home")

	cmd.AddCommand(NewCmdService(rf))

	return cmd
}

type RootFlags struct {
	Home string
}
