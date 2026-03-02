package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/MaximoCoder/enveil-cli/internal/serverclient"
	"github.com/MaximoCoder/enveil-cli/internal/ui"
	"github.com/MaximoCoder/enveil-core/config"
	"github.com/MaximoCoder/enveil-core/vault"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage server connection",
}

var serverPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push the current project and its variables to the server",
	RunE:  runServerPush,
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

var serverUseProjectCmd = &cobra.Command{
	Use:   "use-project <name>",
	Short: "Associate the current directory with an existing project on the server",
	Args:  cobra.ExactArgs(1),
	RunE:  runServerUseProject,
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.AddCommand(serverPushCmd)
	serverCmd.AddCommand(serverConnectCmd)
	serverCmd.AddCommand(serverDisconnectCmd)
	serverCmd.AddCommand(serverStatusCmd)
	serverCmd.AddCommand(serverUseProjectCmd)

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

func runServerUseProject(cmd *cobra.Command, args []string) error {
	 name := args[0]

    cfg, err := config.Load()
    if err != nil {
        return err
    }

    if !cfg.HasServer() {
        return fmt.Errorf("no server configured, run 'enveil server connect' first")
    }

    // Verificar que el proyecto existe en el servidor
    client := serverclient.New(cfg.ServerURL, cfg.ServerAPIKey)
    projects, err := client.ListProjects()
    if err != nil {
        return fmt.Errorf("error fetching projects from server: %w", err)
    }

    found := false
    for _, p := range projects {
        if pname, ok := p["Name"].(string); ok && pname == name {
            found = true
            break
        }
    }

    if !found {
        var names []string
        for _, p := range projects {
            if pname, ok := p["Name"].(string); ok {
                names = append(names, pname)
            }
        }
        return fmt.Errorf("project '%s' not found on server. Available: %s", name, strings.Join(names, ", "))
    }

    // Abrir el vault local y registrar el directorio
    cwd, err := os.Getwd()
    if err != nil {
        return fmt.Errorf("error getting current directory: %w", err)
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

    // Verificar si el directorio ya está registrado
    projectID, existingName, err := v.GetProjectByPath(cwd)
    if err != nil {
        return err
    }

    if projectID != 0 {
        if existingName == name {
            ui.Warning("Directory already associated with project '%s'", name)
            return nil
        }
        return fmt.Errorf("directory already registered as project '%s', unregister it first", existingName)
    }

    // Registrar en el vault local
    projectID, err = v.CreateProject(name, cwd)
    if err != nil {
        return err
    }

    _, err = v.CreateEnvironment(projectID, "development")
    if err != nil {
        return err
    }

    // Actualizar config global
    cfg.ActiveProject = name
    if cfg.ActiveEnv == "" {
        cfg.ActiveEnv = "development"
    }
    if err := cfg.Save(); err != nil {
        return err
    }

    ui.Success("Directory associated with server project '%s'", name)
    ui.Info("Run 'enveil list' to see available variables")
    return nil
}

func runServerPush(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !cfg.HasServer() {
		return fmt.Errorf("no server configured, run 'enveil server connect' first")
	}

	if cfg.ActiveProject == "" {
		return fmt.Errorf("no active project, run 'enveil init' in your project directory first")
	}

	// Abrir vault local
	masterKeyHex, err := promptAndDeriveKey(cfg)
	if err != nil {
		return err
	}

	v, err := vault.Open(cfg.VaultPath, masterKeyHex)
	if err != nil {
		return err
	}
	defer v.Close()

	// Obtener el proyecto local
	projectID, _, err := v.GetProjectByName(cfg.ActiveProject)
	if err != nil {
		return fmt.Errorf("error looking up project: %w", err)
	}
	if projectID == 0 {
		return fmt.Errorf("project '%s' not found in local vault", cfg.ActiveProject)
	}

	client := serverclient.New(cfg.ServerURL, cfg.ServerAPIKey)

	// Crear el proyecto en el servidor si no existe
	if err := client.EnsureProject(cfg.ActiveProject); err != nil {
		return fmt.Errorf("error creating project on server: %w", err)
	}

	// Obtener todos los entornos locales
	envs, err := v.ListEnvironments(projectID)
	if err != nil {
		return fmt.Errorf("error listing environments: %w", err)
	}

	totalVars := 0

	for _, envName := range envs {
		// Crear el entorno en el servidor
		if err := client.EnsureEnvironment(cfg.ActiveProject, envName); err != nil {
			return fmt.Errorf("error creating environment '%s' on server: %w", envName, err)
		}

		// Obtener el ID del entorno local
		envID, err := v.GetEnvironment(projectID, envName)
		if err != nil {
			return fmt.Errorf("error getting environment '%s': %w", envName, err)
		}

		// Leer todas las variables del entorno
		vars, err := v.GetVariables(envID)
		if err != nil {
			return fmt.Errorf("error reading variables for '%s': %w", envName, err)
		}

		// Subir cada variable al servidor
		for key, value := range vars {
			if err := client.SetVariable(cfg.ActiveProject, envName, key, value); err != nil {
				return fmt.Errorf("error pushing variable '%s': %w", key, err)
			}
			totalVars++
		}

		ui.Info("Pushed environment '%s' (%d variables)", envName, len(vars))
	}

	ui.Success("Project '%s' pushed to server (%d variables total)", cfg.ActiveProject, totalVars)
	return nil
}