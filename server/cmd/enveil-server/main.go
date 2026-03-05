package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MaximoCoder/Enveil/core/crypto"
	"github.com/MaximoCoder/Enveil/core/vault"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const version = "0.2.2"

type Server struct {
	vault        *vault.Vault
	apiKey       string
	transportKey []byte
	router       *chi.Mux
}

func main() {
	apiKey := os.Getenv("ENVEIL_API_KEY")
	if apiKey == "" {
		log.Fatal("ENVEIL_API_KEY environment variable is required")
	}

	vaultPath := os.Getenv("ENVEIL_VAULT_PATH")
	if vaultPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		vaultPath = home + "/.enveil-server/vault.db"
	}

	vaultPassword := os.Getenv("ENVEIL_VAULT_PASSWORD")
	if vaultPassword == "" {
		log.Fatal("ENVEIL_VAULT_PASSWORD environment variable is required")
	}

	port := os.Getenv("ENVEIL_PORT")
	if port == "" {
		port = "8080"
	}

	// Ensure vault directory exists
	vaultDir := vaultPath[:len(vaultPath)-len("/vault.db")]
	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		log.Fatalf("error creating vault directory: %v", err)
	}

	// Derive master key from password
	saltPath := vaultDir + "/salt"
	var salt []byte
	if _, err := os.Stat(saltPath); os.IsNotExist(err) {
		// First run: generate salt
		newSalt, err := crypto.GenerateSalt()
		if err != nil {
			log.Fatalf("error generating salt: %v", err)
		}
		if err := os.WriteFile(saltPath, newSalt, 0600); err != nil {
			log.Fatalf("error saving salt: %v", err)
		}
		salt = newSalt
		fmt.Println("  First run: vault initialized")
	} else {
		salt, err = os.ReadFile(saltPath)
		if err != nil {
			log.Fatalf("error reading salt: %v", err)
		}
	}

	masterKey := crypto.DeriveKey(vaultPassword, salt)
	masterKeyHex := crypto.KeyToHex(masterKey)

	// Open vault
	v, err := vault.Open(vaultPath, masterKeyHex)
	if err != nil {
		log.Fatalf("error opening vault: %v", err)
	}
	defer v.Close()

	// Setup server
	s := &Server{
		vault:        v,
		apiKey:       apiKey,
		transportKey: crypto.DeriveTransportKey(apiKey),
		router:       chi.NewRouter(),
	}

	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(s.authMiddleware)

	s.registerRoutes()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: s.router,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Printf("  Enveil Server %s running on port %s\n", version, port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("error starting server: %v", err)
		}
	}()

	<-stop
	fmt.Println("\n  Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	fmt.Println("  Server stopped")
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow health check without auth
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get("X-API-Key")
		if key != s.apiKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}