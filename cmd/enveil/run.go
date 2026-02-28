package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/maximodev/enveil/internal/config"
	"github.com/maximodev/enveil/internal/vault"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <comando>",
	Short: "Ejecuta un comando con las variables del entorno activo inyectadas",
	// DisableFlagParsing permite pasar flags al comando hijo sin que cobra las intercepte
	// por ejemplo: enveil run npm run dev --watch
	DisableFlagParsing: true,
	Args:               cobra.MinimumNArgs(1),
	RunE:               runRun,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !cfg.IsInitialized() {
		return fmt.Errorf("enveil no esta inicializado, corre 'enveil init' primero")
	}

	if cfg.ActiveProject == "" {
		return fmt.Errorf("no hay proyecto activo, corre 'enveil init' en este directorio")
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

	// Obtener proyecto y entorno activo
	projectID, _, err := v.GetProjectByPath(getCurrentDir())
	if err != nil {
		return err
	}
	if projectID == 0 {
		return fmt.Errorf("proyecto no encontrado en el vault, corre 'enveil init'")
	}

	envID, err := v.GetEnvironment(projectID, cfg.ActiveEnv)
	if err != nil {
		return err
	}
	if envID == 0 {
		return fmt.Errorf("entorno '%s' no encontrado", cfg.ActiveEnv)
	}

	// Obtener todas las variables del entorno
	vars, err := v.GetVariables(envID)
	if err != nil {
		return err
	}

	// Construir el environment del proceso hijo
	// Partimos del environment actual del sistema y agregamos las variables del vault
	env := os.Environ()
	for k, val := range vars {
		env = append(env, fmt.Sprintf("%s=%s", k, val))
	}

	// Buscar el binario del comando
	binary, err := exec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("comando no encontrado: %s", args[0])
	}

	// syscall.Exec reemplaza el proceso actual con el nuevo proceso
	// Esto es mas limpio que exec.Command porque no crea un proceso hijo
	// sino que el proceso actual SE CONVIERTE en el nuevo proceso
	// Las variables nunca tocan el disco, solo viven en la memoria del proceso
	return syscall.Exec(binary, args, env)
}