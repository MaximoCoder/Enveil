package main

import (
	"fmt"
	"sort"

	"github.com/MaximoCoder/enveil-cli/internal/serverclient"
	"github.com/MaximoCoder/enveil-cli/internal/ui"
	"github.com/MaximoCoder/enveil-core/config"
	"github.com/MaximoCoder/enveil-core/vault"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List variables in the active environment",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
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

	var vars map[string]string

	if cfg.HasServer() {
		client := serverclient.New(cfg.ServerURL, cfg.ServerAPIKey)
		vars, err = client.GetVariables(cfg.ActiveProject, cfg.ActiveEnv)
		if err != nil {
			return fmt.Errorf("error fetching variables from server: %w", err)
		}
	} else {
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

		vars, err = v.GetVariables(envID)
		if err != nil {
			return err
		}
	}

	ui.Header(fmt.Sprintf("%s  %s", cfg.ActiveProject, ui.EnvBadge(cfg.ActiveProject, cfg.ActiveEnv)))

	if len(vars) == 0 {
		ui.Muted("  No variables in this environment.")
		fmt.Println()
		return nil
	}

	// Find longest key for alignment
	maxLen := 0
	for k := range vars {
		if len(k) > maxLen {
			maxLen = len(k)
		}
	}

	// Sort keys
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf("  %-*s  =  ***\n", maxLen, k)
	}

	fmt.Println()
	ui.Muted("  %d variable(s) total", len(vars))
	fmt.Println()
	return nil
}