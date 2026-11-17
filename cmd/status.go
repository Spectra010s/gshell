package cmd

import (
	"gshell/internal/auth"
	"gshell/internal/cloudshell"

	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show your Cloud Shell environment status",
	Long: `Show the state of your Cloud Shell environment.

If the environment is suspended, run gshell to start it.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		u := getUI()

		project, err := resolveProject()
		if err != nil {
			return err
		}

		tokenSource, err := auth.GetTokenSource(ctx)
		if err != nil {
			return err
		}

		client := cloudshell.NewClient(ctx, tokenSource)

		env, err := client.GetEnvironment(ctx)
		if err != nil {
			return err
		}

		if account := env.Account(); account != "" {
			u.Printf("  Account      %s\n", account)
		}
		if project == "" {
			u.Printf("  Project      not set\n")
		} else {
			u.Printf("  Project      %s\n", project)
		}
		u.Printf("  State        %s\n", displayState(env.State))
		if env.SSHHost != "" {
			u.Printf("  SSH          %s@%s:%d\n", env.SSHUsername, env.SSHHost, env.SSHPort)
		} else {
			u.Printf("  SSH          not assigned\n")
		}
		u.Printf("  Public keys  %d registered\n", len(env.PublicKeys))

		if env.State != "RUNNING" {
			u.Printf("\nRun gshell to start the machine.\n")
		}

		return nil
	},
}

// displayState turns the API's state enum into something readable.
func displayState(state string) string {
	switch state {
	case "RUNNING":
		return "Running"
	case "SUSPENDED":
		return "Suspended"
	case "PENDING":
		return "Starting"
	case "STATE_UNSPECIFIED", "":
		return "Unknown"
	default:
		return state
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
