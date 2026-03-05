package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/MaximoCoder/Enveil/cli/internal/daemon"
	"github.com/MaximoCoder/Enveil/cli/internal/ui"
	"github.com/MaximoCoder/Enveil/core/config"
	"github.com/MaximoCoder/Enveil/core/crypto"
	"github.com/MaximoCoder/Enveil/core/vault"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Enveil or register the current project",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Si no esta inicializado, hacer el setup inicial
	if !cfg.IsInitialized() {
		if err := setupEnveil(cfg); err != nil {
			return err
		}
	}

	// Registrar el directorio actual como proyecto
	return registerProject(cfg)
}

func setupEnveil(cfg *config.Config) error {
	ui.Header("Welcome to Enveil")
	ui.Info("Setting up for the first time...")
	fmt.Println()

	// Pedir contraseña sin mostrarla en pantalla
	fmt.Print("  Create a master password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("error reading password: %w", err)
	}

	fmt.Print("  Confirm master password: ")
	confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("error reading password: %w", err)
	}

	if string(passwordBytes) != string(confirmBytes) {
		return fmt.Errorf("passwords do not match")
	}

	// Generar salt y derivar master key
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return err
	}

	masterKey := crypto.DeriveKey(string(passwordBytes), salt)
	masterKeyHex := crypto.KeyToHex(masterKey)

	// Obtener ruta del vault
	vaultPath, err := config.DefaultVaultPath()
	if err != nil {
		return err
	}

	// Crear ~/.enveil antes de intentar abrir el vault
	enveilDir := filepath.Dir(vaultPath)
	if err := os.MkdirAll(enveilDir, 0700); err != nil {
		return fmt.Errorf("error creating enveil directory: %w", err)
	}

	// Crear y verificar el vault
	v, err := vault.Open(vaultPath, masterKeyHex)
	if err != nil {
		return err
	}
	defer v.Close()

	// Guardar config
	cfg.VaultPath = vaultPath
	cfg.Salt = hex.EncodeToString(salt)

	if err := cfg.Save(); err != nil {
		return err
	}

	ui.Success("Vault created at %s", vaultPath)
	return nil
}

func registerProject(cfg *config.Config) error {
	// Obtener directorio actual
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting current directory: %w", err)
	}

	// Pedir contraseña para abrir el vault
	masterKeyHex, err := promptAndDeriveKey(cfg)
	if err != nil {
		return err
	}

	v, err := vault.Open(cfg.VaultPath, masterKeyHex)
	if err != nil {
		return err
	}
	defer v.Close()

	// Verificar si el proyecto ya existe
	projectID, name, err := v.GetProjectByPath(cwd)
	if err != nil {
		return err
	}

	if projectID != 0 {
		ui.Warning("Directory already registered as project '%s'", name)
		return nil
	}

	// Usar el nombre del directorio como nombre del proyecto
	projectName := filepath.Base(cwd)

	projectID, err = v.CreateProject(projectName, cwd)
	if err != nil {
		return err
	}

	// Crear entorno development por defecto
	_, err = v.CreateEnvironment(projectID, "development")
	if err != nil {
		return err
	}

	// Actualizar config con el proyecto activo
	cfg.ActiveProject = projectName
	cfg.ActiveEnv = "development"
	if err := cfg.Save(); err != nil {
		return err
	}

	ui.Success("Project '%s' registered with environment 'development'", projectName)
	return nil
}

func promptAndDeriveKey(cfg *config.Config) (string, error) {
	// Intentar obtener la key del daemon primero
	if key, ok := daemon.GetKey(); ok {
		return key, nil
	}

	// Si el daemon no esta corriendo, pedir la contraseña
	fmt.Print("  Master password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("error reading password: %w", err)
	}

	salt, err := hex.DecodeString(cfg.Salt)
	if err != nil {
		return "", fmt.Errorf("error decoding salt: %w", err)
	}

	masterKey := crypto.DeriveKey(string(passwordBytes), salt)
	return crypto.KeyToHex(masterKey), nil
}

func daemonGetKey() (string, bool) {
	return daemon.GetKey()
}