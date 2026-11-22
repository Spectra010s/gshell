package cmd

import (
	"errors"

	"gshell/internal/auth"

	"github.com/spf13/cobra"
)

// authCmd manages user credentials
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Google authentication credentials",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to your Google account",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := getUI()
		if _, err := auth.Login(cmd.Context(), u); err != nil {
			return err
		}
		if email, _ := auth.Email(); email != "" {
			u.Successf("Authenticated as %s", email)
		} else {
			u.Successf("Authenticated")
		}
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove saved credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := getUI()

		token, err := auth.LoadToken()
		if errors.Is(err, auth.ErrNoCredentials) {
			return errors.New("not logged in")
		}
		if err != nil {
			return err
		}

		if err := auth.RevokeToken(cmd.Context(), token); err != nil {
			u.Warnf("Could not revoke credentials: %v", err)
		}

		if err := auth.DeleteToken(); err != nil {
			return err
		}
		u.Successf("Logged out. Credentials removed.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "View current authentication state",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		u := getUI()

		tok, err := auth.LoadToken()
		if err != nil {
			return err
		}

		// Refresh an expired token so the status reflects whether the session
		// can actually be used.
		if !tok.Valid() {
			tokenSource, err := auth.GetTokenSource(ctx)
			if err != nil {
				return err
			}
			if _, err := tokenSource.Token(); err != nil {
				return err
			}
		}

		u.Successf("Authenticated to Google Cloud Shell")
		if email, err := auth.Email(); err == nil && email != "" {
			u.Printf("  Account: %s\n", email)
		}
		if path, err := auth.CredentialsPath(); err == nil {
			u.Printf("  Credentials: %s\n", path)
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
