package main

import (
	"fmt"

	"github.com/MaximoCoder/Enveil/cli/internal/serverclient"
	"github.com/MaximoCoder/Enveil/cli/internal/ui"
	"github.com/MaximoCoder/Enveil/core/config"
	"github.com/MaximoCoder/Enveil/core/vault"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environments of the active project",
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List environments of the active project",
	RunE:  runEnvList,
}

var envAddCmd = &cobra.Command{
	Use:   "add <nombre>",
	Short: "Create a new environment in the active project",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvAdd,
}

var envUseCmd = &cobra.Command{
	Use:   "use <nombre>",
	Short: "Switch the active environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvUse,
}

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envAddCmd)
	envCmd.AddCommand(envUseCmd)
}

func runEnvList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !cfg.IsInitialized() {
		return fmt.Errorf("enveil is not initialized, run 'enveil init' first")
	}

	ui.Header(fmt.Sprintf("Environments  %s", ui.EnvBadge(cfg.ActiveProject, cfg.ActiveEnv)))

	if cfg.HasServer() {
		client := serverclient.New(cfg.ServerURL, cfg.ServerAPIKey)
		envs, err := client.ListEnvironments(cfg.ActiveProject)
		if err != nil {
			return fmt.Errorf("error fetching environments from server: %w", err)
		}
		for _, e := range envs {
			if e == cfg.ActiveEnv {
				fmt.Printf("  %s %s\n", ui.ActiveMarker(), e)
			} else {
				fmt.Printf("  %s %s\n", ui.InactiveMarker(), e)
			}
		}
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
		return fmt.Errorf("this directory is not registered, run 'enveil init'")
	}

	envs, err := v.ListEnvironments(projectID)
	if err != nil {
		return err
	}

	for _, e := range envs {
		if e == cfg.ActiveEnv {
			fmt.Printf("  %s %s\n", ui.ActiveMarker(), e)
		} else {
			fmt.Printf("  %s %s\n", ui.InactiveMarker(), e)
		}
	}
	return nil
}

func runEnvAdd(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !cfg.IsInitialized() {
		return fmt.Errorf("enveil is not initialized, run 'enveil init' first")
	}

	if cfg.HasServer() {
		client := serverclient.New(cfg.ServerURL, cfg.ServerAPIKey)
		if err := client.EnsureEnvironment(cfg.ActiveProject, name); err != nil {
			return err
		}
		ui.Success("Environment '%s' created in project '%s'", name, cfg.ActiveProject)
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
		return fmt.Errorf("this directory is not registered, run 'enveil init'")
	}

	envID, err := v.GetEnvironment(projectID, name)
	if err != nil {
		return err
	}
	if envID != 0 {
		return fmt.Errorf("environment '%s' already exists", name)
	}

	_, err = v.CreateEnvironment(projectID, name)
	if err != nil {
		return err
	}

	ui.Success("Environment '%s' created in project '%s'", name, cfg.ActiveProject)
	return nil
}

func runEnvUse(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !cfg.IsInitialized() {
		return fmt.Errorf("enveil is not initialized, run 'enveil init' first")
	}

	if cfg.HasServer() {
		// Verify environment exists on server
		client := serverclient.New(cfg.ServerURL, cfg.ServerAPIKey)
		envs, err := client.ListEnvironments(cfg.ActiveProject)
		if err != nil {
			return err
		}
		found := false
		for _, e := range envs {
			if e == name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("environment '%s' does not exist, create it with 'enveil env add %s'", name, name)
		}

		cfg.ActiveEnv = name
		if err := cfg.Save(); err != nil {
			return err
		}
		ui.Success("Active environment switched to '%s'", name)
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
		return fmt.Errorf("this directory is not registered, run 'enveil init'")
	}

	envID, err := v.GetEnvironment(projectID, name)
	if err != nil {
		return err
	}
	if envID == 0 {
		return fmt.Errorf("environment '%s' does not exist, create it with 'enveil env add %s'", name, name)
	}

	cfg.ActiveEnv = name
	if err := cfg.Save(); err != nil {
		return err
	}

	ui.Success("Active environment switched to '%s'", name)
	return nil
}