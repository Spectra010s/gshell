package cmd

import (
	"fmt"

	"gshell/internal/auth"
	"gshell/internal/cloudshell"

	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current Cloud Shell account and project info",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		tokenSource, err := auth.GetTokenSource(ctx)
		if err != nil {
			return err
		}

		client := cloudshell.NewClient(ctx, tokenSource)

		fmt.Println("Querying Cloud Shell environment...")
		env, err := client.GetEnvironment(ctx)
		if err != nil {
			return err
		}

		fmt.Println()
		fmt.Printf("  Environment:  %s\n", env.Name)
		fmt.Printf("  State:        %s\n", env.State)
		if env.SSHHost != "" {
			fmt.Printf("  SSH Host:     %s\n", env.SSHHost)
			fmt.Printf("  SSH Port:     %d\n", env.SSHPort)
			fmt.Printf("  SSH User:     %s\n", env.SSHUsername)
		}
		fmt.Printf("  Public Keys:  %d registered\n", len(env.PublicKeys))
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
