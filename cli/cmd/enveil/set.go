package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/MaximoCoder/enveil-cli/internal/serverclient"
	"github.com/MaximoCoder/enveil-cli/internal/ui"
	"github.com/MaximoCoder/enveil-core/config"
	"github.com/MaximoCoder/enveil-core/vault"
	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set KEY=VALUE",
	Short: "Save or update a variable in the active environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runSet,
}

var setLocal bool

func init() {
	setCmd.Flags().BoolVar(&setLocal, "local", false, "Save as a local override (never pushed to server)")
	rootCmd.AddCommand(setCmd)
}

func runSet(cmd *cobra.Command, args []string) error {
	parts := strings.SplitN(args[0], "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format, use KEY=VALUE")
	}
	key := strings.TrimSpace(parts[0])
	value := parts[1]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !cfg.IsInitialized() {
		return fmt.Errorf("enveil is not initialized, run 'enveil init' first")
	}

	if cfg.ActiveProject == "" {
		return fmt.Errorf("no active project, run 'enveil init' in a project directory")
	}

	// --local: always goes to the local vault regardless of server mode
	if setLocal {
		return saveLocalOverride(cfg, key, value)
	}

	// Normal set: use server if configured
	if cfg.HasServer() {
		client := serverclient.New(cfg.ServerURL, cfg.ServerAPIKey)
		if err := client.EnsureProject(cfg.ActiveProject); err != nil {
			return err
		}
		if err := client.EnsureEnvironment(cfg.ActiveProject, cfg.ActiveEnv); err != nil {
			return err
		}
		if err := client.SetVariable(cfg.ActiveProject, cfg.ActiveEnv, key, value); err != nil {
			return err
		}
		ui.Success("'%s' saved in %s", key, ui.EnvBadge(cfg.ActiveProject, cfg.ActiveEnv))
		return nil
	}

	// Fall back to local vault
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

	if err := v.SetVariable(envID, key, value); err != nil {
		return err
	}

	ui.Success("'%s' saved in %s", key, ui.EnvBadge(cfg.ActiveProject, cfg.ActiveEnv))
	return nil
}

func saveLocalOverride(cfg *config.Config, key, value string) error {
	masterKeyHex, err := promptAndDeriveKey(cfg)
	if err != nil {
		return err
	}

	v, err := vault.Open(cfg.VaultPath, masterKeyHex)
	if err != nil {
		return err
	}
	defer v.Close()

	if err := v.SetLocalOverride(cfg.ActiveProject, cfg.ActiveEnv, key, value); err != nil {
		return err
	}

	ui.Success("'%s' saved as local override in %s", key, ui.EnvBadge(cfg.ActiveProject, cfg.ActiveEnv))
	fmt.Fprintf(os.Stderr, "  this value is local to this machine and will not be pushed to the server\n")
	return nil
}

func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}