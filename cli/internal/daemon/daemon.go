package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// Request is the message the CLI sends to the daemon
type Request struct {
	Command string `json:"command"`
}

// Response is the message the daemon sends back to the CLI
type Response struct {
	Success bool   `json:"success"`
	Key     string `json:"key,omitempty"`
	Error   string `json:"error,omitempty"`
}

// SocketPath returns the path to the Unix socket
func SocketPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".enveil", "daemon.sock"), nil
}

// PidPath returns the path to the daemon PID file
func PidPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".enveil", "daemon.pid"), nil
}

// IsRunning checks whether the daemon is running by attempting to connect to the socket
func IsRunning() bool {
	socketPath, err := SocketPath()
	if err != nil {
		return false
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// GetKey attempts to retrieve the master key from the daemon
// Returns the key and true if the daemon is running, empty string and false otherwise
func GetKey() (string, bool) {
	socketPath, err := SocketPath()
	if err != nil {
		return "", false
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return "", false
	}
	defer conn.Close()

	// Send request
	req := Request{Command: "get_key"}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return "", false
	}

	// Read response
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return "", false
	}

	if !resp.Success || resp.Key == "" {
		return "", false
	}

	return resp.Key, true
}

// Server is the daemon server
type Server struct {
	masterKey  string
	socketPath string
	listener   net.Listener
}

// NewServer creates a new daemon server with the master key held in memory
func NewServer(masterKey string) (*Server, error) {
	socketPath, err := SocketPath()
	if err != nil {
		return nil, err
	}

	// Remove previous socket if it exists
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("error creating socket: %w", err)
	}

	// Only the current user can access the socket
	if err := os.Chmod(socketPath, 0600); err != nil {
		listener.Close()
		return nil, fmt.Errorf("error setting socket permissions: %w", err)
	}

	return &Server{
		masterKey:  masterKey,
		socketPath: socketPath,
		listener:   listener,
	}, nil
}

// Serve starts the daemon main loop accepting connections
func (s *Server) Serve() error {
	defer os.Remove(s.socketPath)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// If the listener was closed this is a normal exit
			return nil
		}
		go s.handleConnection(conn)
	}
}

// Close shuts down the server
func (s *Server) Close() {
	s.listener.Close()
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	var req Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		return
	}

	var resp Response

	switch req.Command {
	case "get_key":
		resp = Response{Success: true, Key: s.masterKey}
	case "status":
		resp = Response{Success: true}
	default:
		resp = Response{Success: false, Error: "unknown command"}
	}

	json.NewEncoder(conn).Encode(resp)
}