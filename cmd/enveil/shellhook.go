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

	// Always verify the current directory against the vault
	// Never show a project badge based on config alone
	masterKeyHex, ok := tryGetKeyFromDaemon()
	if !ok {
		// Without daemon we cannot verify, clear the badge
		fmt.Println("export ENVEIL_PROJECT=")
		fmt.Println("export ENVEIL_ENV=")
		return nil
	}

	v, err := vault.Open(cfg.VaultPath, masterKeyHex)
	if err != nil {
		return nil
	}
	defer v.Close()

	projectID, name, err := v.GetProjectByPath(cwd)
	if err != nil || projectID == 0 {
		// Directory not registered, clear the badge
		fmt.Println("export ENVEIL_PROJECT=")
		fmt.Println("export ENVEIL_ENV=")
		return nil
	}

	// Update config if project changed
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
	# Add this to your ~/.zshrc or ~/.bashrc:
	#   eval "$(enveil shell-init)"

	_enveil_hook() {
	local output
	output=$(enveil shell-hook 2>/dev/null)
	if [ -n "$output" ]; then
		eval "$output"
	fi
	}

	# Detect shell and register hook accordingly
	if [ -n "$ZSH_VERSION" ]; then
	# zsh
	autoload -U add-zsh-hook
	add-zsh-hook chpwd _enveil_hook
	elif [ -n "$BASH_VERSION" ]; then
	# bash
	if [[ "$PROMPT_COMMAND" != *"_enveil_hook"* ]]; then
		PROMPT_COMMAND="_enveil_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
	fi
	fi

	# Show enveil status in prompt
	_enveil_prompt() {
	if [ -n "$ENVEIL_PROJECT" ]; then
		if [ -n "$ZSH_VERSION" ]; then
		echo "%F{cyan}[${ENVEIL_PROJECT}/${ENVEIL_ENV}]%f"
		else
		echo "\[\033[0;36m\][${ENVEIL_PROJECT}/${ENVEIL_ENV}]\[\033[0m\]"
		fi
	fi
	}

	if [ -n "$ZSH_VERSION" ]; then
	# zsh - use RPROMPT
	RPROMPT='$(_enveil_prompt)'
	elif [ -n "$BASH_VERSION" ]; then
	# bash - prepend to PS1
	if [[ "$PS1" != *'$(_enveil_prompt)'* ]]; then
		PS1='$(_enveil_prompt)'"$PS1"
	fi
	fi

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