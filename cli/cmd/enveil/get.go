package main

import (
	"fmt"
	"strings"

	"github.com/MaximoCoder/enveil-cli/internal/serverclient"
	"github.com/MaximoCoder/enveil-core/config"
	"github.com/MaximoCoder/enveil-core/vault"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <KEY>",
	Short: "Get the value of a variable in the active environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runGet,
}

func init() {
	rootCmd.AddCommand(getCmd)
}

func runGet(cmd *cobra.Command, args []string) error {
	key := strings.TrimSpace(args[0])

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !cfg.IsInitialized() {
		return fmt.Errorf("enveil is not initialized, run 'enveil init' first")
	}

	// Derive key once — used for local vault in both modes
	masterKeyHex, err := promptAndDeriveKey(cfg)
	if err != nil {
		return err
	}

	localVault, err := vault.Open(cfg.VaultPath, masterKeyHex)
	if err != nil {
		return err
	}
	defer localVault.Close()

	if cfg.HasServer() {
		// Check local override first
		if overrideVal, exists, err := localVault.GetLocalOverride(cfg.ActiveProject, cfg.ActiveEnv, key); err != nil {
			return fmt.Errorf("error reading local overrides: %w", err)
		} else if exists {
			fmt.Println(overrideVal)
			return nil
		}

		client := serverclient.New(cfg.ServerURL, cfg.ServerAPIKey)
		vars, err := client.GetVariables(cfg.ActiveProject, cfg.ActiveEnv)
		if err != nil {
			return fmt.Errorf("error fetching variables from server: %w", err)
		}

		value, ok := vars[key]
		if !ok {
			return fmt.Errorf("variable '%s' not found", key)
		}
		fmt.Println(value)
		return nil
	}

	projectID, _, err := localVault.GetProjectByPath(getCurrentDir())
	if err != nil {
		return err
	}
	if projectID == 0 {
		return fmt.Errorf("project not found in vault, run 'enveil init'")
	}

	envID, err := localVault.GetEnvironment(projectID, cfg.ActiveEnv)
	if err != nil {
		return err
	}
	if envID == 0 {
		return fmt.Errorf("environment '%s' not found", cfg.ActiveEnv)
	}

	value, err := localVault.GetVariable(envID, key)
	if err != nil {
		return err
	}
	if value == "" {
		return fmt.Errorf("variable '%s' not found", key)
	}

	fmt.Println(value)
	return nil
}