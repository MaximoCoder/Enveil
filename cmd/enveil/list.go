package main

import (
	"fmt"

	"github.com/maximodev/enveil/internal/config"
	"github.com/maximodev/enveil/internal/ui"
	"github.com/maximodev/enveil/internal/vault"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista las variables del entorno activo",
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

	envID, err := v.GetEnvironment(projectID, cfg.ActiveEnv)
	if err != nil {
		return err
	}
	if envID == 0 {
		return fmt.Errorf("environment '%s' not found", cfg.ActiveEnv)
	}

	vars, err := v.GetVariables(envID)
	if err != nil {
		return err
	}

	ui.Header(fmt.Sprintf("%s  %s", cfg.ActiveProject, ui.EnvBadge(cfg.ActiveProject, cfg.ActiveEnv)))

	if len(vars) == 0 {
		ui.Muted("  No variables in this environment.")
		return nil
	}

	// Calcular el ancho maximo de los keys para alinear los valores
	maxLen := 0
	for k := range vars {
		if len(k) > maxLen {
			maxLen = len(k)
		}
	}

	for k := range vars {
		fmt.Printf("  %-*s  =  ***\n", maxLen, k)
	}

	ui.Muted("  %d variable(s) total", len(vars))
	return nil
}