package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yiffyi/xfbbroker/cmd/db"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Edit database via command line",
}

func SetupDBCommand() *cobra.Command {
	dbCmd.AddCommand(db.SetupUserCommand())
	dbCmd.AddCommand(db.SetupNotifyCommand())
	return dbCmd
}
