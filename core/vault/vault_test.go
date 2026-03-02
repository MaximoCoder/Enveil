package vault

import (
	"os"
	"testing"
)

// tempVault creates a temporary vault for testing and returns its path and a cleanup function
func tempVault(t *testing.T) (*Vault, func()) {
	t.Helper()

	f, err := os.CreateTemp("", "enveil-test-vault-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	f.Close()

	path := f.Name()
	masterKey := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 64 hex chars = 32 bytes

	v, err := Open(path, masterKey)
	if err != nil {
		os.Remove(path)
		t.Fatalf("failed to open vault: %v", err)
	}

	return v, func() {
		v.Close()
		os.Remove(path)
	}
}

// --- Open ---

func TestOpenCreatesVault(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	if v == nil {
		t.Fatal("expected vault to be non-nil")
	}
}

func TestOpenRejectsWrongKey(t *testing.T) {
	f, _ := os.CreateTemp("", "enveil-test-vault-*.db")
	f.Close()
	defer os.Remove(f.Name())

	correctKey := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	wrongKey := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	v, err := Open(f.Name(), correctKey)
	if err != nil {
		t.Fatalf("failed to open vault with correct key: %v", err)
	}
	// Create schema so the file is a valid encrypted vault
	v.Close()

	_, err = Open(f.Name(), wrongKey)
	if err == nil {
		t.Fatal("opening vault with wrong key should fail")
	}
}

// --- Projects ---

func TestCreateAndGetProjectByPath(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	id, err := v.CreateProject("myapp", "/home/user/myapp")
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero project ID")
	}

	gotID, gotName, err := v.GetProjectByPath("/home/user/myapp")
	if err != nil {
		t.Fatalf("GetProjectByPath failed: %v", err)
	}
	if gotID != id {
		t.Fatalf("expected project ID %d, got %d", id, gotID)
	}
	if gotName != "myapp" {
		t.Fatalf("expected project name 'myapp', got %q", gotName)
	}
}

func TestGetProjectByPathReturnsZeroWhenNotFound(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	id, name, err := v.GetProjectByPath("/nonexistent/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 0 || name != "" {
		t.Fatal("expected zero ID and empty name for nonexistent path")
	}
}

func TestCreateAndGetProjectByName(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	id, err := v.CreateProject("backend", "/home/user/backend")
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	gotID, gotPath, err := v.GetProjectByName("backend")
	if err != nil {
		t.Fatalf("GetProjectByName failed: %v", err)
	}
	if gotID != id {
		t.Fatalf("expected project ID %d, got %d", id, gotID)
	}
	if gotPath != "/home/user/backend" {
		t.Fatalf("expected path '/home/user/backend', got %q", gotPath)
	}
}

func TestListProjects(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	v.CreateProject("project-a", "/home/user/project-a")
	v.CreateProject("project-b", "/home/user/project-b")

	projects, err := v.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
}

func TestDeleteProject(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	id, _ := v.CreateProject("todelete", "/home/user/todelete")
	envID, _ := v.CreateEnvironment(id, "development")
	v.SetVariable(envID, "KEY", "value")

	if err := v.DeleteProject(id); err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}

	gotID, _, _ := v.GetProjectByPath("/home/user/todelete")
	if gotID != 0 {
		t.Fatal("project should not exist after deletion")
	}
}

// --- Environments ---

func TestCreateAndGetEnvironment(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	projectID, _ := v.CreateProject("myapp", "/home/user/myapp")

	envID, err := v.CreateEnvironment(projectID, "development")
	if err != nil {
		t.Fatalf("CreateEnvironment failed: %v", err)
	}
	if envID == 0 {
		t.Fatal("expected non-zero environment ID")
	}

	gotID, err := v.GetEnvironment(projectID, "development")
	if err != nil {
		t.Fatalf("GetEnvironment failed: %v", err)
	}
	if gotID != envID {
		t.Fatalf("expected environment ID %d, got %d", envID, gotID)
	}
}

func TestGetEnvironmentReturnsZeroWhenNotFound(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	projectID, _ := v.CreateProject("myapp", "/home/user/myapp")

	id, err := v.GetEnvironment(projectID, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 0 {
		t.Fatal("expected zero ID for nonexistent environment")
	}
}

func TestListEnvironments(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	projectID, _ := v.CreateProject("myapp", "/home/user/myapp")
	v.CreateEnvironment(projectID, "development")
	v.CreateEnvironment(projectID, "staging")
	v.CreateEnvironment(projectID, "production")

	envs, err := v.ListEnvironments(projectID)
	if err != nil {
		t.Fatalf("ListEnvironments failed: %v", err)
	}
	if len(envs) != 3 {
		t.Fatalf("expected 3 environments, got %d", len(envs))
	}
}

// --- Variables ---

func TestSetAndGetVariable(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	projectID, _ := v.CreateProject("myapp", "/home/user/myapp")
	envID, _ := v.CreateEnvironment(projectID, "development")

	if err := v.SetVariable(envID, "DATABASE_URL", "postgres://localhost/mydb"); err != nil {
		t.Fatalf("SetVariable failed: %v", err)
	}

	value, err := v.GetVariable(envID, "DATABASE_URL")
	if err != nil {
		t.Fatalf("GetVariable failed: %v", err)
	}
	if value != "postgres://localhost/mydb" {
		t.Fatalf("expected 'postgres://localhost/mydb', got %q", value)
	}
}

func TestSetVariableOverwritesExistingValue(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	projectID, _ := v.CreateProject("myapp", "/home/user/myapp")
	envID, _ := v.CreateEnvironment(projectID, "development")

	v.SetVariable(envID, "API_KEY", "old-value")
	v.SetVariable(envID, "API_KEY", "new-value")

	value, _ := v.GetVariable(envID, "API_KEY")
	if value != "new-value" {
		t.Fatalf("expected 'new-value', got %q", value)
	}
}

func TestGetVariableReturnsEmptyWhenNotFound(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	projectID, _ := v.CreateProject("myapp", "/home/user/myapp")
	envID, _ := v.CreateEnvironment(projectID, "development")

	value, err := v.GetVariable(envID, "NONEXISTENT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "" {
		t.Fatalf("expected empty string, got %q", value)
	}
}

func TestGetVariables(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	projectID, _ := v.CreateProject("myapp", "/home/user/myapp")
	envID, _ := v.CreateEnvironment(projectID, "development")

	v.SetVariable(envID, "KEY_A", "value-a")
	v.SetVariable(envID, "KEY_B", "value-b")

	vars, err := v.GetVariables(envID)
	if err != nil {
		t.Fatalf("GetVariables failed: %v", err)
	}
	if len(vars) != 2 {
		t.Fatalf("expected 2 variables, got %d", len(vars))
	}
	if vars["KEY_A"] != "value-a" {
		t.Fatalf("expected 'value-a', got %q", vars["KEY_A"])
	}
	if vars["KEY_B"] != "value-b" {
		t.Fatalf("expected 'value-b', got %q", vars["KEY_B"])
	}
}

func TestDeleteVariable(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	projectID, _ := v.CreateProject("myapp", "/home/user/myapp")
	envID, _ := v.CreateEnvironment(projectID, "development")

	v.SetVariable(envID, "TO_DELETE", "value")

	if err := v.DeleteVariable(envID, "TO_DELETE"); err != nil {
		t.Fatalf("DeleteVariable failed: %v", err)
	}

	value, _ := v.GetVariable(envID, "TO_DELETE")
	if value != "" {
		t.Fatal("variable should not exist after deletion")
	}
}

func TestVariableExists(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	projectID, _ := v.CreateProject("myapp", "/home/user/myapp")
	envID, _ := v.CreateEnvironment(projectID, "development")

	exists, _ := v.VariableExists(envID, "KEY")
	if exists {
		t.Fatal("variable should not exist before being set")
	}

	v.SetVariable(envID, "KEY", "value")

	exists, _ = v.VariableExists(envID, "KEY")
	if !exists {
		t.Fatal("variable should exist after being set")
	}
}

// --- Isolation between environments ---

func TestVariablesAreIsolatedBetweenEnvironments(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	projectID, _ := v.CreateProject("myapp", "/home/user/myapp")
	devID, _ := v.CreateEnvironment(projectID, "development")
	prodID, _ := v.CreateEnvironment(projectID, "production")

	v.SetVariable(devID, "DB_HOST", "localhost")
	v.SetVariable(prodID, "DB_HOST", "prod.db.internal")

	devVal, _ := v.GetVariable(devID, "DB_HOST")
	prodVal, _ := v.GetVariable(prodID, "DB_HOST")

	if devVal != "localhost" {
		t.Fatalf("expected 'localhost' in development, got %q", devVal)
	}
	if prodVal != "prod.db.internal" {
		t.Fatalf("expected 'prod.db.internal' in production, got %q", prodVal)
	}
}

func TestVariablesAreIsolatedBetweenProjects(t *testing.T) {
	v, cleanup := tempVault(t)
	defer cleanup()

	idA, _ := v.CreateProject("project-a", "/home/user/project-a")
	idB, _ := v.CreateProject("project-b", "/home/user/project-b")

	envA, _ := v.CreateEnvironment(idA, "development")
	envB, _ := v.CreateEnvironment(idB, "development")

	v.SetVariable(envA, "SECRET", "secret-a")
	v.SetVariable(envB, "SECRET", "secret-b")

	valA, _ := v.GetVariable(envA, "SECRET")
	valB, _ := v.GetVariable(envB, "SECRET")

	if valA != "secret-a" {
		t.Fatalf("expected 'secret-a', got %q", valA)
	}
	if valB != "secret-b" {
		t.Fatalf("expected 'secret-b', got %q", valB)
	}
}