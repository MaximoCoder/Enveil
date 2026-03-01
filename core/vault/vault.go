package vault

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

// Vault represents the encrypted database
type Vault struct {
	db *sql.DB
}

// Project represents a registered project
type Project struct {
	ID   int64
	Name string
	Path string
}

// Open opens or creates the vault at the given path with the given master key
func Open(path string, masterKey string) (*Vault, error) {
	// The DSN tells SQLite where the file is and which key to use
	dsn := fmt.Sprintf("%s?_pragma_key=%s&_pragma_cipher_page_size=4096", path, masterKey)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening vault: %w", err)
	}

	// Ensure correct permissions: only the current user can read the vault
	if err := os.Chmod(path, 0600); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error setting vault permissions: %w", err)
	}

	// Verify the connection actually works
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to vault: %w", err)
	}

	v := &Vault{db: db}

	if err := v.createSchema(); err != nil {
		return nil, err
	}

	return v, nil
}

// Close closes the database connection
func (v *Vault) Close() error {
	return v.db.Close()
}

// createSchema creates the tables if they do not exist
func (v *Vault) createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		name      TEXT NOT NULL UNIQUE,
		path      TEXT NOT NULL UNIQUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS environments (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		name       TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id),
		UNIQUE(project_id, name)
	);

	CREATE TABLE IF NOT EXISTS variables (
		id             INTEGER PRIMARY KEY AUTOINCREMENT,
		environment_id INTEGER NOT NULL,
		key            TEXT NOT NULL,
		value          TEXT NOT NULL,
		created_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (environment_id) REFERENCES environments(id),
		UNIQUE(environment_id, key)
	);
	`

	_, err := v.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("error creating schema: %w", err)
	}

	return nil
}

// CreateProject registers a new project in the vault
func (v *Vault) CreateProject(name, path string) (int64, error) {
	result, err := v.db.Exec(
		"INSERT INTO projects (name, path) VALUES (?, ?)",
		name, path,
	)
	if err != nil {
		return 0, fmt.Errorf("error creating project: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error getting project id: %w", err)
	}

	return id, nil
}

// GetProjectByPath looks up a project by its path on disk
func (v *Vault) GetProjectByPath(path string) (int64, string, error) {
	var id int64
	var name string

	err := v.db.QueryRow(
		"SELECT id, name FROM projects WHERE path = ?", path,
	).Scan(&id, &name)

	if err == sql.ErrNoRows {
		return 0, "", nil
	}
	if err != nil {
		return 0, "", fmt.Errorf("error looking up project: %w", err)
	}

	return id, name, nil
}

// CreateEnvironment creates an environment within a project
func (v *Vault) CreateEnvironment(projectID int64, name string) (int64, error) {
	result, err := v.db.Exec(
		"INSERT INTO environments (project_id, name) VALUES (?, ?)",
		projectID, name,
	)
	if err != nil {
		return 0, fmt.Errorf("error creating environment: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error getting environment id: %w", err)
	}

	return id, nil
}

// GetEnvironment looks up an environment by project and name
func (v *Vault) GetEnvironment(projectID int64, name string) (int64, error) {
	var id int64

	err := v.db.QueryRow(
		"SELECT id FROM environments WHERE project_id = ? AND name = ?",
		projectID, name,
	).Scan(&id)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("error looking up environment: %w", err)
	}

	return id, nil
}

// SetVariable saves or updates a variable in an environment
func (v *Vault) SetVariable(environmentID int64, key, value string) error {
	_, err := v.db.Exec(`
		INSERT INTO variables (environment_id, key, value)
		VALUES (?, ?, ?)
		ON CONFLICT(environment_id, key)
		DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`, environmentID, key, value)

	if err != nil {
		return fmt.Errorf("error saving variable: %w", err)
	}

	return nil
}

// GetVariables returns all variables in an environment as a map
func (v *Vault) GetVariables(environmentID int64) (map[string]string, error) {
	rows, err := v.db.Query(
		"SELECT key, value FROM variables WHERE environment_id = ?",
		environmentID,
	)
	if err != nil {
		return nil, fmt.Errorf("error reading variables: %w", err)
	}
	defer rows.Close()

	vars := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("error reading row: %w", err)
		}
		vars[key] = value
	}

	return vars, nil
}

// ListEnvironments returns the names of all environments in a project
func (v *Vault) ListEnvironments(projectID int64) ([]string, error) {
	rows, err := v.db.Query(
		"SELECT name FROM environments WHERE project_id = ? ORDER BY created_at ASC",
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("error listing environments: %w", err)
	}
	defer rows.Close()

	var envs []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("error reading environment: %w", err)
		}
		envs = append(envs, name)
	}

	return envs, nil
}

// VariableExists checks whether a variable exists in an environment
func (v *Vault) VariableExists(environmentID int64, key string) (bool, error) {
	var count int
	err := v.db.QueryRow(
		"SELECT COUNT(*) FROM variables WHERE environment_id = ? AND key = ?",
		environmentID, key,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking variable: %w", err)
	}
	return count > 0, nil
}

// DeleteVariable removes a variable from an environment
func (v *Vault) DeleteVariable(environmentID int64, key string) error {
	_, err := v.db.Exec(
		"DELETE FROM variables WHERE environment_id = ? AND key = ?",
		environmentID, key,
	)
	if err != nil {
		return fmt.Errorf("error deleting variable: %w", err)
	}
	return nil
}

// GetVariable returns the value of a single variable
func (v *Vault) GetVariable(environmentID int64, key string) (string, error) {
	var value string
	err := v.db.QueryRow(
		"SELECT value FROM variables WHERE environment_id = ? AND key = ?",
		environmentID, key,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("error getting variable: %w", err)
	}
	return value, nil
}

// ListProjects returns all registered projects
func (v *Vault) ListProjects() ([]Project, error) {
	rows, err := v.db.Query(
		"SELECT id, name, path FROM projects ORDER BY created_at ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("error listing projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Path); err != nil {
			return nil, fmt.Errorf("error reading project: %w", err)
		}
		projects = append(projects, p)
	}

	return projects, nil
}

// DeleteProject removes a project and all its environments and variables
func (v *Vault) DeleteProject(projectID int64) error {
	// Get all environment IDs for this project
	rows, err := v.db.Query(
		"SELECT id FROM environments WHERE project_id = ?", projectID,
	)
	if err != nil {
		return fmt.Errorf("error getting environments: %w", err)
	}
	defer rows.Close()

	var envIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("error reading environment: %w", err)
		}
		envIDs = append(envIDs, id)
	}
	rows.Close()

	// Delete all variables for each environment
	for _, envID := range envIDs {
		_, err := v.db.Exec(
			"DELETE FROM variables WHERE environment_id = ?", envID,
		)
		if err != nil {
			return fmt.Errorf("error deleting variables: %w", err)
		}
	}

	// Delete all environments
	_, err = v.db.Exec(
		"DELETE FROM environments WHERE project_id = ?", projectID,
	)
	if err != nil {
		return fmt.Errorf("error deleting environments: %w", err)
	}

	// Delete the project
	_, err = v.db.Exec(
		"DELETE FROM projects WHERE id = ?", projectID,
	)
	if err != nil {
		return fmt.Errorf("error deleting project: %w", err)
	}

	return nil
}

// GetProjectByName looks up a project by its name
func (v *Vault) GetProjectByName(name string) (int64, string, error) {
	var id int64
	var path string

	err := v.db.QueryRow(
		"SELECT id, path FROM projects WHERE name = ?", name,
	).Scan(&id, &path)

	if err == sql.ErrNoRows {
		return 0, "", nil
	}
	if err != nil {
		return 0, "", fmt.Errorf("error looking up project: %w", err)
	}

	return id, path, nil
}