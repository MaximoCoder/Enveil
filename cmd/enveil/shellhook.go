package main

import (
	"fmt"
	"os"

	"github.com/maximodev/enveil/internal/config"
	"github.com/maximodev/enveil/internal/vault"
	"github.com/spf13/cobra"
)

var shellHookCmd = &cobra.Command{
	Use:    "shell-hook",
	Short:  "Detect project on directory change (used by shell plugin)",
	Hidden: true,
	RunE:   runShellHook,
}

var shellInitCmd = &cobra.Command{
	Use:   "shell-init",
	Short: "Print shell integration script",
	RunE:  runShellInit,
}

func init() {
	rootCmd.AddCommand(shellHookCmd)
	rootCmd.AddCommand(shellInitCmd)
}

func runShellHook(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	cfg, err := config.Load()
	if err != nil || !cfg.IsInitialized() {
		return nil
	}

	// Intentar obtener la key del daemon
	// Si el daemon no esta corriendo no hacemos nada, no pedimos password
	masterKeyHex, ok := tryGetKeyFromDaemon()
	if !ok {
		// Sin daemon no podemos verificar el vault, pero podemos
		// al menos actualizar el proyecto activo si el directorio esta en config
		if cfg.ActiveProject != "" {
			fmt.Printf("export ENVEIL_PROJECT=%s\n", cfg.ActiveProject)
			fmt.Printf("export ENVEIL_ENV=%s\n", cfg.ActiveEnv)
		}
		return nil
	}

	v, err := vault.Open(cfg.VaultPath, masterKeyHex)
	if err != nil {
		return nil
	}
	defer v.Close()

	projectID, name, err := v.GetProjectByPath(cwd)
	if err != nil || projectID == 0 {
		// Directorio no registrado, limpiar variables
		fmt.Println("export ENVEIL_PROJECT=")
		fmt.Println("export ENVEIL_ENV=")
		return nil
	}

	// Actualizar config si el proyecto cambio
	if cfg.ActiveProject != name {
		cfg.ActiveProject = name
		cfg.ActiveEnv = "development"
		cfg.Save()
	}

	fmt.Printf("export ENVEIL_PROJECT=%s\n", name)
	fmt.Printf("export ENVEIL_ENV=%s\n", cfg.ActiveEnv)
	return nil
}

func runShellInit(cmd *cobra.Command, args []string) error {
	script := `
	# Enveil shell integration
	# Add this to your ~/.zshrc:
	#   eval "$(enveil shell-init)"

	_enveil_hook() {
	local output
	output=$(enveil shell-hook 2>/dev/null)
	if [ -n "$output" ]; then
		eval "$output"
	fi
	}

	# Run hook on directory change
	autoload -U add-zsh-hook
	add-zsh-hook chpwd _enveil_hook

	# Show enveil status on the right side of the prompt
	_enveil_prompt() {
	if [ -n "$ENVEIL_PROJECT" ]; then
		echo "%F{cyan}[${ENVEIL_PROJECT}/${ENVEIL_ENV}]%f"
	fi
	}

	# Use RPROMPT so it doesn't interfere with existing themes
	RPROMPT='$(_enveil_prompt)'

	# Run once on shell startup for current directory
	_enveil_hook
	`
	fmt.Print(script)
	return nil
}

func tryGetKeyFromDaemon() (string, bool) {
	// Importamos la funcion GetKey del paquete daemon
	// Si el daemon no esta corriendo retorna false silenciosamente
	return daemonGetKey()
}