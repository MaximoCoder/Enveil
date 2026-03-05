package main

import (
	"encoding/json"
	"net/http"

	"github.com/MaximoCoder/Enveil/core/crypto"
	"github.com/go-chi/chi/v5"
)

func (s *Server) registerRoutes() {
	s.router.Get("/health", s.handleHealth)
	s.router.Get("/projects", s.handleListProjects)
	s.router.Post("/projects", s.handleCreateProject)
	s.router.Get("/projects/{project}/envs", s.handleListEnvs)
	s.router.Post("/projects/{project}/envs", s.handleCreateEnv)
	s.router.Get("/projects/{project}/envs/{env}/vars", s.handleGetVars)
	s.router.Post("/projects/{project}/envs/{env}/vars", s.handleSetVar)
	s.router.Delete("/projects/{project}/envs/{env}/vars/{key}", s.handleDeleteVar)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": version})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.vault.ListProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(projects)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Path == "" {
		req.Path = req.Name
	}

	id, err := s.vault.CreateProject(req.Name, req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id, "name": req.Name})
}

func (s *Server) handleListEnvs(w http.ResponseWriter, r *http.Request) {
	projectName := chi.URLParam(r, "project")

	projectID, _, err := s.vault.GetProjectByName(projectName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if projectID == 0 {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	envs, err := s.vault.ListEnvironments(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(envs)
}

func (s *Server) handleCreateEnv(w http.ResponseWriter, r *http.Request) {
	projectName := chi.URLParam(r, "project")

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	projectID, _, err := s.vault.GetProjectByName(projectName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if projectID == 0 {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	id, err := s.vault.CreateEnvironment(projectID, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id, "name": req.Name})
}

func (s *Server) handleGetVars(w http.ResponseWriter, r *http.Request) {
	projectName := chi.URLParam(r, "project")
	envName := chi.URLParam(r, "env")

	projectID, _, err := s.vault.GetProjectByName(projectName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if projectID == 0 {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	envID, err := s.vault.GetEnvironment(projectID, envName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if envID == 0 {
		http.Error(w, "environment not found", http.StatusNotFound)
		return
	}

	vars, err := s.vault.GetVariables(envID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Encrypt each value before sending to the CLI
	encrypted := make(map[string]string)
	for k, v := range vars {
		enc, err := crypto.Encrypt(v, s.transportKey)
		if err != nil {
			http.Error(w, "error encrypting values", http.StatusInternalServerError)
			return
		}
		encrypted[k] = enc
	}

	json.NewEncoder(w).Encode(encrypted)
}

func (s *Server) handleSetVar(w http.ResponseWriter, r *http.Request) {
	projectName := chi.URLParam(r, "project")
	envName := chi.URLParam(r, "env")

	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	projectID, _, err := s.vault.GetProjectByName(projectName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if projectID == 0 {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	envID, err := s.vault.GetEnvironment(projectID, envName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if envID == 0 {
		http.Error(w, "environment not found", http.StatusNotFound)
		return
	}

	// Decrypt the value that arrived encrypted from the CLI
	decrypted, err := crypto.Decrypt(req.Value, s.transportKey)
	if err != nil {
		http.Error(w, "error decrypting value", http.StatusBadRequest)
		return
	}

	if err := s.vault.SetVariable(envID, req.Key, decrypted); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"key": req.Key})
}

func (s *Server) handleDeleteVar(w http.ResponseWriter, r *http.Request) {
	projectName := chi.URLParam(r, "project")
	envName := chi.URLParam(r, "env")
	key := chi.URLParam(r, "key")

	projectID, _, err := s.vault.GetProjectByName(projectName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if projectID == 0 {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	envID, err := s.vault.GetEnvironment(projectID, envName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if envID == 0 {
		http.Error(w, "environment not found", http.StatusNotFound)
		return
	}

	if err := s.vault.DeleteVariable(envID, key); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}