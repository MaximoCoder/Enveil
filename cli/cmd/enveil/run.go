package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/MaximoCoder/enveil-cli/internal/serverclient"
	"github.com/MaximoCoder/enveil-core/config"
	"github.com/MaximoCoder/enveil-core/vault"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <comando>",
	Short: "Run a command with the active environment variables injected",
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
		return fmt.Errorf("enveil is not initialized, run 'enveil init' first")
	}

	var vars map[string]string

	// Use server if configured
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

	// Build environment and exec
	env := os.Environ()
	for k, v := range vars {
		env = append(env, k+"="+v)
	}

	binary, err := exec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("command not found: %s", args[0])
	}

	return syscall.Exec(binary, args, env)
}