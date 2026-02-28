package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/maximodev/enveil/internal/config"
	"github.com/maximodev/enveil/internal/ui"
	"github.com/maximodev/enveil/internal/vault"
	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set KEY=VALUE",
	Short: "Save or update a variable in the active environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runSet,
}

func init() {
	rootCmd.AddCommand(setCmd)
}

func runSet(cmd *cobra.Command, args []string) error {
	// Parsear KEY=VALUE
	parts := strings.SplitN(args[0], "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format, use KEY=VALUE")
	}

	key := strings.TrimSpace(parts[0])
	value := parts[1]

	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	// Cargar config
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

	// Abrir vault
	masterKeyHex, err := promptAndDeriveKey(cfg)
	if err != nil {
		return err
	}

	v, err := vault.Open(cfg.VaultPath, masterKeyHex)
	if err != nil {
		return err
	}
	defer v.Close()

	// Obtener el proyecto activo
	projectID, _, err := v.GetProjectByPath(getCurrentDir())
	if err != nil {
		return err
	}
	if projectID == 0 {
		return fmt.Errorf("project not found in vault, run 'enveil init'")
	}

	// Obtener el entorno activo
	envID, err := v.GetEnvironment(projectID, cfg.ActiveEnv)
	if err != nil {
		return err
	}
	if envID == 0 {
		return fmt.Errorf("environment '%s' not found", cfg.ActiveEnv)
	}

	// Guardar la variable
	if err := v.SetVariable(envID, key, value); err != nil {
		return err
	}

	ui.Success("'%s' saved in %s", key, ui.EnvBadge(cfg.ActiveProject, cfg.ActiveEnv))
	return nil
}

func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}