package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yiffyi/xfbbroker/cmd"
	"github.com/yiffyi/xfbbroker/misc"
)

func main() {
	err := misc.LoadConfig([]string{"."})
	if err != nil {
		panic(err)
	}

	misc.SetupLogger()

	var rootCmd = &cobra.Command{
		Use:   "xfbbroker",
		Short: "xfbbroker is a CLI application",
		Long:  "xfbbroker is a CLI application built with spf13/cobra to manage your tasks.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Welcome to xfbbroker!")
		},
	}

	rootCmd.AddCommand(cmd.SetupServeCommand())
	rootCmd.AddCommand(cmd.SetupDBCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
