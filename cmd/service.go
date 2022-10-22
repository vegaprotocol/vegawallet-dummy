package cmd

import "github.com/spf13/cobra"

func NewCmdService(rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage the service",
		Long:  "Manage the service",
	}

	cmd.AddCommand(NewCmdRunService(rf))
	return cmd
}
