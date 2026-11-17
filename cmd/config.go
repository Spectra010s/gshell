package cmd

import (
	"fmt"

	appconfig "gshell/internal/config"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage gshell configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set project <project-id>",
	Short: "Set the project to use in Cloud Shell",
	Long:  `Set the Google Cloud project that gshell activates when it connects.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] != "project" {
			return fmt.Errorf("unknown setting %q; supported settings: project", args[0])
		}
		if err := appconfig.ValidateProject(args[1]); err != nil {
			return err
		}

		cfg, err := appconfig.Load()
		if err != nil {
			return err
		}
		cfg.Project = args[1]

		if err := appconfig.Save(cfg); err != nil {
			return err
		}

		getUI().Successf("Project set to %s", args[1])
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show gshell configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := getUI()

		cfg, err := appconfig.Load()
		if err != nil {
			return err
		}

		if cfg.Project == "" {
			u.Printf("  Project      not set\n")
			u.Printf("\nSet one with: gshell config set project <project-id>\n")
			return nil
		}

		u.Printf("  Project      %s\n", cfg.Project)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}
