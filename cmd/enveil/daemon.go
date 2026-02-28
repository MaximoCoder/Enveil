package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/maximodev/enveil/internal/config"
	"github.com/maximodev/enveil/internal/daemon"
	"github.com/maximodev/enveil/internal/ui"
	"github.com/maximodev/enveil/internal/vault"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the Enveil daemon",
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the daemon in the background",
	RunE:  runDaemonStart,
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the daemon",
	RunE:  runDaemonStop,
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the daemon is running",
	RunE:  runDaemonStatus,
}

// daemonServeCmd is the internal command that actually runs the server
// Hidden from help, only invoked by daemonStartCmd
var daemonServeCmd = &cobra.Command{
	Use:    "serve",
	Hidden: true,
	RunE:   runDaemonServe,
}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonServeCmd)
}

func runDaemonStart(cmd *cobra.Command, args []string) error {
	if daemon.IsRunning() {
		ui.Warning("Daemon is already running")
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !cfg.IsInitialized() {
		return fmt.Errorf("enveil is not initialized, run 'enveil init' first")
	}

	// Ask for password once
	masterKeyHex, err := promptAndDeriveKey(cfg)
	if err != nil {
		return err
	}

	// Validate the key against the vault before starting the daemon
	v, err := vault.Open(cfg.VaultPath, masterKeyHex)
	if err != nil {
		return fmt.Errorf("invalid password")
	}
	v.Close()

	// Launch child process passing the key as an environment variable
	// The key is already derived, it is not the user's password
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error getting executable path: %w", err)
	}

	child := exec.Command(exe, "daemon", "serve")
	child.Env = append(os.Environ(), "ENVEIL_MASTER_KEY="+masterKeyHex)
	child.Stdin = nil
	child.Stdout = nil
	child.Stderr = nil
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := child.Start(); err != nil {
		return fmt.Errorf("error starting daemon: %w", err)
	}

	// Save PID
	pidPath, err := daemon.PidPath()
	if err != nil {
		return err
	}

	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(child.Process.Pid)), 0600); err != nil {
		return fmt.Errorf("error saving PID: %w", err)
	}

	ui.Success("Daemon started (PID %d)", child.Process.Pid)
	ui.Info("Master key is in memory. No password needed until you stop the daemon.")
	return nil
}

func runDaemonServe(cmd *cobra.Command, args []string) error {
	// Get master key from environment
	masterKeyHex := os.Getenv("ENVEIL_MASTER_KEY")
	if masterKeyHex == "" {
		return fmt.Errorf("ENVEIL_MASTER_KEY no encontrada")
	}

	// Clear the variable from environment immediately
	os.Unsetenv("ENVEIL_MASTER_KEY")

	server, err := daemon.NewServer(masterKeyHex)
	if err != nil {
		return err
	}

	// Handle signals for clean shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		server.Close()
		os.Exit(0)
	}()

	return server.Serve()
}

func runDaemonStop(cmd *cobra.Command, args []string) error {
	pidPath, err := daemon.PidPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(pidPath)
	if os.IsNotExist(err) {
		// No PID found but there may be an orphan process, clean up the socket
		socketPath, _ := daemon.SocketPath()
		os.Remove(socketPath)
		ui.Info("Daemon is not running")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error reading PID: %w", err)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return fmt.Errorf("invalid PID: %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidPath)
		ui.Info("Daemon was not running")
		return nil
	}

	// Check if the process actually exists before sending signal
	if err := process.Signal(syscall.Signal(0)); err != nil {
		// Process does not exist, clean up orphan files
		os.Remove(pidPath)
		socketPath, _ := daemon.SocketPath()
		os.Remove(socketPath)
		ui.Warning("Daemon was not running, files cleaned up")
		return nil
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("error stopping daemon: %w", err)
	}

	os.Remove(pidPath)
	socketPath, _ := daemon.SocketPath()
	os.Remove(socketPath)
	ui.Success("Daemon stopped. Master key removed from memory.")
	return nil
}

func runDaemonStatus(cmd *cobra.Command, args []string) error {
	if daemon.IsRunning() {
		pidPath, _ := daemon.PidPath()
		data, err := os.ReadFile(pidPath)
		if err == nil {
			ui.Success("Daemon running (PID %s)", string(data))
		} else {
			ui.Success("Daemon running")
		}
	} else {
		ui.Muted("  Daemon is not running")
	}
	return nil
}