package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "xfbbroker",
		Short: "xfbbroker is a CLI application",
		Long:  "xfbbroker is a CLI application built with spf13/cobra to manage your tasks.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Welcome to xfbbroker!")
		},
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}