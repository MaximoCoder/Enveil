package main

import (
	"fmt"
	"strings"

	"github.com/MaximoCoder/enveil-core/config"
	"github.com/MaximoCoder/enveil-cli/internal/serverclient"
	"github.com/MaximoCoder/enveil-cli/internal/ui"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage server connection",
}

var serverConnectCmd = &cobra.Command{
	Use:   "connect <url>",
	Short: "Connect to an Enveil server",
	Args:  cobra.ExactArgs(1),
	RunE:  runServerConnect,
}

var serverDisconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Disconnect from the server",
	RunE:  runServerDisconnect,
}

var serverStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check server connection status",
	RunE:  runServerStatus,
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.AddCommand(serverConnectCmd)
	serverCmd.AddCommand(serverDisconnectCmd)
	serverCmd.AddCommand(serverStatusCmd)

	serverConnectCmd.Flags().String("key", "", "API key for authentication")
	serverConnectCmd.MarkFlagRequired("key")
}

func runServerConnect(cmd *cobra.Command, args []string) error {
	url := strings.TrimRight(args[0], "/")
	apiKey, _ := cmd.Flags().GetString("key")

	// Test connection before saving
	client := serverclient.New(url, apiKey)
	if err := client.Health(); err != nil {
		return fmt.Errorf("could not connect to server: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cfg.ServerURL = url
	cfg.ServerAPIKey = apiKey

	if err := cfg.Save(); err != nil {
		return err
	}

	ui.Success("Connected to server at %s", url)
	ui.Info("All commands will now use the server instead of the local vault")
	return nil
}

func runServerDisconnect(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cfg.ServerURL = ""
	cfg.ServerAPIKey = ""

	if err := cfg.Save(); err != nil {
		return err
	}

	ui.Success("Disconnected from server. Using local vault.")
	return nil
}

func runServerStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !cfg.HasServer() {
		ui.Muted("  No server configured. Using local vault.")
		fmt.Println()
		return nil
	}

	client := serverclient.New(cfg.ServerURL, cfg.ServerAPIKey)
	if err := client.Health(); err != nil {
		ui.Error("Server at %s is unreachable: %v", cfg.ServerURL, err)
		return nil
	}

	ui.Success("Connected to %s", cfg.ServerURL)
	return nil
}