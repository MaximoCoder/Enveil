package main

import (
	"fmt"
	"sort"

	"github.com/MaximoCoder/Enveil/cli/internal/serverclient"
	"github.com/MaximoCoder/Enveil/cli/internal/ui"
	"github.com/MaximoCoder/Enveil/core/config"
	"github.com/MaximoCoder/Enveil/core/vault"
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
	var overrides map[string]string

	// Derive key once — used for local vault regardless of mode
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
		client := serverclient.New(cfg.ServerURL, cfg.ServerAPIKey)
		vars, err = client.GetVariables(cfg.ActiveProject, cfg.ActiveEnv)
		if err != nil {
			return fmt.Errorf("error fetching variables from server: %w", err)
		}

		overrides, err = localVault.GetLocalOverrides(cfg.ActiveProject, cfg.ActiveEnv)
		if err != nil {
			return fmt.Errorf("error reading local overrides: %w", err)
		}
	} else {
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

		vars, err = localVault.GetVariables(envID)
		if err != nil {
			return err
		}
	}

	ui.Header(fmt.Sprintf("%s  %s", cfg.ActiveProject, ui.EnvBadge(cfg.ActiveProject, cfg.ActiveEnv)))

	if len(vars) == 0 && len(overrides) == 0 {
		ui.Muted("  No variables in this environment.")
		fmt.Println()
		return nil
	}

	allKeys := make(map[string]struct{})
	for k := range vars {
		allKeys[k] = struct{}{}
	}
	for k := range overrides {
		allKeys[k] = struct{}{}
	}

	maxLen := 0
	for k := range allKeys {
		if len(k) > maxLen {
			maxLen = len(k)
		}
	}

	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	localCount := 0
	for _, k := range keys {
		if _, isOverride := overrides[k]; isOverride {
			fmt.Printf("  %-*s  =  *** [local override]\n", maxLen, k)
			localCount++
		} else {
			fmt.Printf("  %-*s  =  ***\n", maxLen, k)
		}
	}

	fmt.Println()
	ui.Muted("  %d variable(s) total", len(allKeys))
	if localCount > 0 {
		ui.Muted("  %d local override(s) — visible only on this machine", localCount)
	}
	fmt.Println()
	return nil
}