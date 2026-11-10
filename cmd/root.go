package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "gshell",
	Short:   "Terminal client for Google Cloud Shell",
	Version: "2.0.0",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Connecting to Cloud Shell...")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
