package cmd

import "github.com/spf13/cobra"

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Use this command to debug",
	Run: func(cmd *cobra.Command, args []string) {
		return
	},
}
