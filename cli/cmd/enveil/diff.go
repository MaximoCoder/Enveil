package main

import (
	"fmt"

	"github.com/MaximoCoder/Enveil/cli/internal/serverclient"
	"github.com/MaximoCoder/Enveil/cli/internal/ui"
	"github.com/MaximoCoder/Enveil/core/config"
	"github.com/MaximoCoder/Enveil/core/vault"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff <entorno1> <entorno2>",
	Short: "Compare variables between two environments without revealing values",
	Args:  cobra.ExactArgs(2),
	RunE:  runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	env1Name := args[0]
	env2Name := args[1]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !cfg.IsInitialized() {
		return fmt.Errorf("enveil is not initialized, run 'enveil init' first")
	}

	var vars1, vars2 map[string]string

	if cfg.HasServer() {
		client := serverclient.New(cfg.ServerURL, cfg.ServerAPIKey)
		vars1, err = client.GetVariables(cfg.ActiveProject, env1Name)
		if err != nil {
			return fmt.Errorf("error fetching '%s' from server: %w", env1Name, err)
		}
		vars2, err = client.GetVariables(cfg.ActiveProject, env2Name)
		if err != nil {
			return fmt.Errorf("error fetching '%s' from server: %w", env2Name, err)
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
			return fmt.Errorf("this directory is not registered, run 'enveil init'")
		}

		env1ID, err := v.GetEnvironment(projectID, env1Name)
		if err != nil {
			return err
		}
		if env1ID == 0 {
			return fmt.Errorf("environment '%s' does not exist", env1Name)
		}

		env2ID, err := v.GetEnvironment(projectID, env2Name)
		if err != nil {
			return err
		}
		if env2ID == 0 {
			return fmt.Errorf("environment '%s' does not exist", env2Name)
		}

		vars1, err = v.GetVariables(env1ID)
		if err != nil {
			return err
		}
		vars2, err = v.GetVariables(env2ID)
		if err != nil {
			return err
		}
	}

	// Compare
	onlyIn1 := []string{}
	onlyIn2 := []string{}
	different := []string{}
	same := []string{}

	for k, v1 := range vars1 {
		v2, exists := vars2[k]
		if !exists {
			onlyIn1 = append(onlyIn1, k)
		} else if v1 != v2 {
			different = append(different, k)
		} else {
			same = append(same, k)
		}
	}

	for k := range vars2 {
		if _, exists := vars1[k]; !exists {
			onlyIn2 = append(onlyIn2, k)
		}
	}

	ui.Header(fmt.Sprintf("Diff  %s vs %s", env1Name, env2Name))

	if len(onlyIn1) == 0 && len(onlyIn2) == 0 && len(different) == 0 {
		ui.Success("Environments are identical")
		return nil
	}

	if len(onlyIn1) > 0 {
		ui.Bold(" Only in '%s':", env1Name)
		for _, k := range onlyIn1 {
			color.New(color.FgRed).Printf("  - %s\n", k)
		}
		fmt.Println()
	}

	if len(onlyIn2) > 0 {
		ui.Bold(" Only in '%s':", env2Name)
		for _, k := range onlyIn2 {
			color.New(color.FgGreen).Printf("  + %s\n", k)
		}
		fmt.Println()
	}

	if len(different) > 0 {
		ui.Bold(" Different values:")
		for _, k := range different {
			fmt.Printf("  ~ %s\n", k)
		}
		fmt.Println()
	}

	if len(same) > 0 {
		ui.Muted(" %d variable(s) identical in both environments", len(same))
	}

	return nil
}