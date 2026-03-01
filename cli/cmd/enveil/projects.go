package main

import (
	"fmt"
	"strings"

	"github.com/MaximoCoder/enveil-core/config"
	"github.com/MaximoCoder/enveil-cli/internal/ui"
	"github.com/MaximoCoder/enveil-core/vault"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List all registered projects",
	RunE:  runProjects,
}

var unregisterCmd = &cobra.Command{
	Use:   "unregister",
	Short: "Remove the current project from the vault",
	RunE:  runUnregister,
}

func init() {
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(unregisterCmd)
}

func runProjects(cmd *cobra.Command, args []string) error {
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

	projects, err := v.ListProjects()
	if err != nil {
		return err
	}

	ui.Header("Registered Projects")

	if len(projects) == 0 {
		ui.Muted("  No projects registered.")
		fmt.Println()
		return nil
	}

	for _, p := range projects {
		if p.Name == cfg.ActiveProject {
			fmt.Printf("  %s %s\n", ui.ActiveMarker(), p.Name)
		} else {
			fmt.Printf("  %s %s\n", ui.InactiveMarker(), p.Name)
		}
		ui.Muted("    %s", p.Path)
	}

	fmt.Println()
	ui.Muted("  %d project(s) total", len(projects))
	fmt.Println()
	return nil
}

func runUnregister(cmd *cobra.Command, args []string) error {
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

	projectID, name, err := v.GetProjectByPath(getCurrentDir())
	if err != nil {
		return err
	}
	if projectID == 0 {
		return fmt.Errorf("this directory is not registered as a project")
	}

	// Ask for confirmation
	fmt.Printf("\n  Remove project '%s' and all its variables? [y/N]: ", name)
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		ui.Muted("  Cancelled")
		return nil
	}

	if err := v.DeleteProject(projectID); err != nil {
		return err
	}

	// Clear active project from config if it was this one
	if cfg.ActiveProject == name {
		cfg.ActiveProject = ""
		cfg.ActiveEnv = ""
		cfg.Save()
	}

	ui.Success("Project '%s' removed from vault", name)
	return nil
}