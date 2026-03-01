package main

import (
	"fmt"
	"strings"

	"github.com/MaximoCoder/enveil-cli/internal/serverclient"
	"github.com/MaximoCoder/enveil-cli/internal/ui"
	"github.com/MaximoCoder/enveil-core/config"
	"github.com/MaximoCoder/enveil-core/vault"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <KEY>",
	Short: "Delete a variable from the active environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	key := strings.TrimSpace(args[0])

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !cfg.IsInitialized() {
		return fmt.Errorf("enveil is not initialized, run 'enveil init' first")
	}

	if cfg.HasServer() {
		fmt.Printf("\n  Delete '%s' from %s? [y/N]: ", key, ui.EnvBadge(cfg.ActiveProject, cfg.ActiveEnv))
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			ui.Muted("  Cancelled")
			return nil
		}

		client := serverclient.New(cfg.ServerURL, cfg.ServerAPIKey)
		if err := client.DeleteVariable(cfg.ActiveProject, cfg.ActiveEnv, key); err != nil {
			return err
		}
		ui.Success("'%s' deleted from %s", key, ui.EnvBadge(cfg.ActiveProject, cfg.ActiveEnv))
		return nil
	}

	masterKeyHex, err := promptAndDeriveKey(cfg)
	if err != nil {
		return err
	}

	v, err := vault.Open(cfg.VaultPath, masterKeyHex)
	if err != nil {
		return err
	}
	defer v.Close()

	projectID, _, err := v.GetProjectByPath(getCurrentDir())
	if err != nil {
		return err
	}
	if projectID == 0 {
		return fmt.Errorf("project not found in vault, run 'enveil init'")
	}

	envID, err := v.GetEnvironment(projectID, cfg.ActiveEnv)
	if err != nil {
		return err
	}
	if envID == 0 {
		return fmt.Errorf("environment '%s' not found", cfg.ActiveEnv)
	}

	exists, err := v.VariableExists(envID, key)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("variable '%s' not found", key)
	}

	fmt.Printf("\n  Delete '%s' from %s? [y/N]: ", key, ui.EnvBadge(cfg.ActiveProject, cfg.ActiveEnv))
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		ui.Muted("  Cancelled")
		return nil
	}

	if err := v.DeleteVariable(envID, key); err != nil {
		return err
	}

	ui.Success("'%s' deleted from %s", key, ui.EnvBadge(cfg.ActiveProject, cfg.ActiveEnv))
	return nil
}