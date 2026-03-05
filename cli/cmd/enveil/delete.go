package main

import (
	"fmt"
	"strings"

	"github.com/MaximoCoder/Enveil/cli/internal/serverclient"
	"github.com/MaximoCoder/Enveil/cli/internal/ui"
	"github.com/MaximoCoder/Enveil/core/config"
	"github.com/MaximoCoder/Enveil/core/vault"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <KEY>",
	Short: "Delete a variable from the active environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

var deleteLocalFlag bool

func init() {
	deleteCmd.Flags().BoolVar(&deleteLocalFlag, "local", false, "Delete only the local override, leave the server variable intact")
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

	// --local: only remove the local override
	if deleteLocalFlag {
		return deleteLocalOverride(cfg, key)
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

		// Also remove local override if one exists, to avoid a ghost value
		if err := removeLocalOverrideIfExists(cfg, key); err != nil {
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

func deleteLocalOverride(cfg *config.Config, key string) error {
	masterKeyHex, err := promptAndDeriveKey(cfg)
	if err != nil {
		return err
	}

	v, err := vault.Open(cfg.VaultPath, masterKeyHex)
	if err != nil {
		return err
	}
	defer v.Close()

	if err := v.DeleteLocalOverride(cfg.ActiveProject, cfg.ActiveEnv, key); err != nil {
		return err
	}

	ui.Success("local override for '%s' removed — server value will be used", key)
	return nil
}

func removeLocalOverrideIfExists(cfg *config.Config, key string) error {
	masterKeyHex, err := promptAndDeriveKey(cfg)
	if err != nil {
		return err
	}

	v, err := vault.Open(cfg.VaultPath, masterKeyHex)
	if err != nil {
		return err
	}
	defer v.Close()

	return v.DeleteLocalOverride(cfg.ActiveProject, cfg.ActiveEnv, key)
}