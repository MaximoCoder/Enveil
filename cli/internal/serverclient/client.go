package serverclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/MaximoCoder/Enveil/core/crypto"
)

// Client is an HTTP client for the Enveil server
type Client struct {
	baseURL      string
	apiKey       string
	transportKey []byte
	http         *http.Client
}

// New creates a new server client
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:      baseURL,
		apiKey:       apiKey,
		transportKey: crypto.DeriveTransportKey(apiKey),
		http:         &http.Client{Timeout: 10 * time.Second},
	}
}

// Health checks if the server is reachable
func (c *Client) Health() error {
	resp, err := c.http.Get(c.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("server unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	return nil
}

// EnsureProject creates a project on the server if it does not exist
func (c *Client) EnsureProject(name string) error {
	body, _ := json.Marshal(map[string]string{"name": name, "path": name})
	req, _ := http.NewRequest("POST", c.baseURL+"/projects", bytes.NewBuffer(body))
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("error creating project: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// ListProjects returns all projects on the server
func (c *Client) ListProjects() ([]map[string]any, error) {
	req, _ := http.NewRequest("GET", c.baseURL+"/projects", nil)
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error listing projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var projects []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return projects, nil
}

// EnsureEnvironment creates an environment on the server if it does not exist
func (c *Client) EnsureEnvironment(project, env string) error {
	body, _ := json.Marshal(map[string]string{"name": env})
	req, _ := http.NewRequest("POST", c.baseURL+"/projects/"+project+"/envs", bytes.NewBuffer(body))
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("error creating environment: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// SetVariable encrypts and sends a variable to the server
func (c *Client) SetVariable(project, env, key, value string) error {
	encrypted, err := crypto.Encrypt(value, c.transportKey)
	if err != nil {
		return fmt.Errorf("error encrypting value: %w", err)
	}

	body, _ := json.Marshal(map[string]string{"key": key, "value": encrypted})
	req, _ := http.NewRequest("POST", c.baseURL+"/projects/"+project+"/envs/"+env+"/vars", bytes.NewBuffer(body))
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("error setting variable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	return nil
}

// GetVariables fetches and decrypts all variables for an environment
func (c *Client) GetVariables(project, env string) (map[string]string, error) {
	req, _ := http.NewRequest("GET", c.baseURL+"/projects/"+project+"/envs/"+env+"/vars", nil)
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting variables: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var encrypted map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&encrypted); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	vars := make(map[string]string)
	for k, v := range encrypted {
		decrypted, err := crypto.Decrypt(v, c.transportKey)
		if err != nil {
			return nil, fmt.Errorf("error decrypting value for %s: %w", k, err)
		}
		vars[k] = decrypted
	}

	return vars, nil
}

// DeleteVariable removes a variable from the server
func (c *Client) DeleteVariable(project, env, key string) error {
	req, _ := http.NewRequest("DELETE", c.baseURL+"/projects/"+project+"/envs/"+env+"/vars/"+key, nil)
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("error deleting variable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	return nil
}

// ListEnvironments returns all environments for a project on the server
func (c *Client) ListEnvironments(project string) ([]string, error) {
	req, _ := http.NewRequest("GET", c.baseURL+"/projects/"+project+"/envs", nil)
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error listing environments: %w", err)
	}
	defer resp.Body.Close()

	var envs []string
	if err := json.NewDecoder(resp.Body).Decode(&envs); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return envs, nil
}