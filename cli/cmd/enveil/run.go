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

	if cfg.HasServer() {
		client := serverclient.New(cfg.ServerURL, cfg.ServerAPIKey)
		vars, err = client.GetVariables(cfg.ActiveProject, cfg.ActiveEnv)
		if err != nil {
			return fmt.Errorf("error fetching variables from server: %w", err)
		}

		// Apply local overrides on top of server variables
		vars, err = applyLocalOverrides(cfg, vars)
		if err != nil {
			return err
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

// applyLocalOverrides opens the local vault and merges any overrides
// on top of the provided vars map. Local overrides always win.
// Returns a new map — does not mutate the original.
func applyLocalOverrides(cfg *config.Config, vars map[string]string) (map[string]string, error) {
	masterKeyHex, err := promptAndDeriveKey(cfg)
	if err != nil {
		return nil, err
	}

	v, err := vault.Open(cfg.VaultPath, masterKeyHex)
	if err != nil {
		return nil, err
	}
	defer v.Close()

	overrides, err := v.GetLocalOverrides(cfg.ActiveProject, cfg.ActiveEnv)
	if err != nil {
		return nil, fmt.Errorf("error reading local overrides: %w", err)
	}

	if len(overrides) == 0 {
		return vars, nil
	}

	merged := make(map[string]string, len(vars))
	for k, v := range vars {
		merged[k] = v
	}
	for k, v := range overrides {
		merged[k] = v
	}

	return merged, nil
}