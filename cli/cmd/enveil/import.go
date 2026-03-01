package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/MaximoCoder/enveil-core/config"
	"github.com/MaximoCoder/enveil-cli/internal/ui"
	"github.com/MaximoCoder/enveil-core/vault"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import variables from a .env file into the active environment",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	// Default to .env in current directory
	filePath := ".env"
	if len(args) > 0 {
		filePath = args[0]
	}

	// Check file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file '%s' not found", filePath)
	}

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

	// Parse the .env file
	vars, err := parseDotEnv(filePath)
	if err != nil {
		return err
	}

	if len(vars) == 0 {
		ui.Warning("No variables found in '%s'", filePath)
		return nil
	}

	// Open vault
	masterKeyHex, err := promptAndDeriveKey(cfg)
	if err != nil {
		return err
	}

	v, err := vault.Open(cfg.VaultPath, masterKeyHex)
	if err != nil {
		return err
	}
	defer v.Close()

	// Get active project and environment
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

	// Import variables
	imported := 0
	for key, value := range vars {
		if err := v.SetVariable(envID, key, value); err != nil {
			ui.Warning("Could not import '%s': %v", key, err)
			continue
		}
		imported++
	}

	ui.Success("Imported %d variable(s) into %s", imported, ui.EnvBadge(cfg.ActiveProject, cfg.ActiveEnv))

	// Ask if user wants to delete the original file
	fmt.Print("\n  Delete the original file? [y/N]: ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) == "y" {
		if err := os.Remove(filePath); err != nil {
			ui.Warning("Could not delete '%s': %v", filePath, err)
		} else {
			ui.Success("'%s' deleted", filePath)
		}
	}

	return nil
}

// parseDotEnv parses a .env file and returns a map of key-value pairs
func parseDotEnv(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines and comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Handle export prefix
		trimmed = strings.TrimPrefix(trimmed, "export ")
		trimmed = strings.TrimSpace(trimmed)

		// Split on first = only
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := parts[1]

		// Remove inline comments (only if not inside quotes)
		value = removeInlineComment(value)

		// Strip surrounding quotes
		value = stripQuotes(value)

		if key == "" {
			continue
		}

		vars[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return vars, nil
}

// removeInlineComment removes inline comments from a value
// Example: "postgres://localhost/db # this is prod" -> "postgres://localhost/db"
// But respects quoted values: '"value # not a comment"' stays intact
func removeInlineComment(value string) string {
	if len(value) == 0 {
		return value
	}

	// If value starts with a quote, don't remove inline comments
	if value[0] == '"' || value[0] == '\'' {
		return value
	}

	// Find # that is preceded by a space
	if idx := strings.Index(value, " #"); idx != -1 {
		value = value[:idx]
	}

	return strings.TrimSpace(value)
}

// stripQuotes removes surrounding quotes from a value
func stripQuotes(value string) string {
	if len(value) < 2 {
		return value
	}

	if (value[0] == '"' && value[len(value)-1] == '"') ||
		(value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}

	return value
}