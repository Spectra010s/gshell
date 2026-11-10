package cmd

import (
	"fmt"
	"time"

	"gshell/internal/auth"

	"github.com/spf13/cobra"
)

// authCmd manages user credentials
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Google authentication credentials",
	Long:  `Authenticate gshell with your Google account to access Cloud Shell sessions and workspaces.`,
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to your Google account",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := auth.Login(cmd.Context())
		return err
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove saved credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.DeleteToken(); err != nil {
			return err
		}
		fmt.Println("Successfully logged out. Credentials removed.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "View current authentication state",
	RunE: func(cmd *cobra.Command, args []string) error {
		tok, err := auth.LoadToken()
		if err != nil {
			if err == auth.ErrNoCredentials {
				fmt.Println("To get started with gshell, please run: gshell auth login")
				return nil
			}
			return err
		}

		if tok.Valid() {
			timeLeft := time.Until(tok.Expiry).Round(time.Minute)
			fmt.Printf("Authenticated. Access token expires in %s (auto-refreshes automatically).\n", timeLeft)
		} else {
			fmt.Println("Access token expired (will auto-refresh on next request).")
		}
		return nil
	},
}

func init() {
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(authStatusCmd)

	rootCmd.AddCommand(authCmd)
}
