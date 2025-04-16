package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yiffyi/xfbbroker/cmd/db"
)

var dbCmd = &cobra.Command{
	Use: "db",

	Short:     "Edit database via command line",
	Args:      cobra.OnlyValidArgs, // only accepts args in ValidArgs
	ValidArgs: []string{"new-user", "blue", "green"},
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func SetupDBCommand() *cobra.Command {
	dbCmd.AddCommand(db.SetupUserCommand())
	return dbCmd
}
