package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/MaximoCoder/enveil-core/detector"
	"github.com/MaximoCoder/enveil-cli/internal/ui"
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Manage Enveil git hooks",
}

var hookInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the pre-commit hook in the current repository",
	RunE:  runHookInstall,
}

var hookRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the secrets scanner (used internally by the hook)",
	RunE:  runHookRun,
}

func init() {
	rootCmd.AddCommand(hookCmd)
	hookCmd.AddCommand(hookInstallCmd)
	hookCmd.AddCommand(hookRunCmd)
}

func runHookInstall(cmd *cobra.Command, args []string) error {
	// Verify we are in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("no git repository found in the current directory")
	}

	hookPath := filepath.Join(".git", "hooks", "pre-commit")

	// Hook content: a shell script that calls enveil hook run
	hookContent := `#!/bin/sh
		# Hook installed by Enveil
		# To skip in an emergency: git commit --no-verify

		enveil hook run
		if [ $? -ne 0 ]; then
		exit 1
		fi
		`

	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		return fmt.Errorf("error installing hook: %w", err)
	}

	ui.Success("Pre-commit hook installed")
	ui.Info("Enveil will scan your files before every commit")
	return nil
}

func runHookRun(cmd *cobra.Command, args []string) error {

	// Allow skipping via ENVEIL_SKIP=1 environment variable
	if os.Getenv("ENVEIL_SKIP") == "1" {
		fmt.Fprintln(os.Stderr, "  Enveil: scan skipped (ENVEIL_SKIP=1)")
		return nil
	}

	// Get only added or modified staged files, not deletions
	output, err := exec.Command("git", "diff", "--cached", "--name-only", "--diff-filter=ACM").Output()
	if err != nil {
		return fmt.Errorf("error getting staged files: %w", err)
	}

	stagedFiles := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(stagedFiles) == 0 || stagedFiles[0] == "" {
		return nil
	}

	totalFindings := 0

	for _, file := range stagedFiles {
		if file == "" {
			continue
		}

		// Check if the file is a blocked .env file
		if isBlockedEnvFile(file) {
			totalFindings++
			fmt.Fprintf(os.Stderr, "\n  Enveil: blocked file detected: %s\n", file)
			fmt.Fprintf(os.Stderr, "    This file should not be committed directly.\n")
			fmt.Fprintf(os.Stderr, "    Use 'enveil import %s' to import it into the vault instead.\n", file)
			continue
		}

		// Skip binary files and files that do not make sense to scan
		if isBinaryExtension(file) || isIgnoredFile(file) {
			continue
		}

		// Read staged content, not the file on disk
		content, err := exec.Command("git", "show", ":"+file).Output()
		if err != nil {
			continue
		}

		findings := detector.ScanContent(string(content))
		if len(findings) == 0 {
			continue
		}

		totalFindings += len(findings)
		fmt.Fprintf(os.Stderr, "\n  Enveil: possible secret detected in %s\n", file)
		for _, f := range findings {
			fmt.Fprintf(os.Stderr, "    Line %d: %s\n", f.Line, f.Reason)
		}
	}

	if totalFindings > 0 {
		fmt.Fprintf(os.Stderr, "\n  Commit blocked by Enveil.\n")
		fmt.Fprintf(os.Stderr, "  To fix: move secrets to vault with 'enveil set' or 'enveil import'\n")
		fmt.Fprintf(os.Stderr, "\n  To override (only if you know what you are doing):\n")
		fmt.Fprintf(os.Stderr, "    ENVEIL_SKIP=1 git commit\n")
		fmt.Fprintf(os.Stderr, "    git commit --no-verify\n\n")
		os.Exit(1)
	}

	return nil
}

// isBlockedEnvFile returns true for .env files that should never be committed
func isBlockedEnvFile(filename string) bool {
	base := filepath.Base(filename)

	// Always allow these patterns - they are templates with fake values
	allowed := []string{
		".env.example",
		".env.template",
		".env.sample",
		".env.test",
		".env.development",
		".env.staging",
	}
	for _, a := range allowed {
		if base == a {
			return false
		}
	}

	// Block these patterns - they contain real secrets
	blocked := []string{
		".env",
		".env.local",
		".env.production",
		".env.prod",
	}
	for _, b := range blocked {
		if base == b {
			return true
		}
	}

	// Block any .env.*.local pattern
	if strings.HasPrefix(base, ".env.") && strings.HasSuffix(base, ".local") {
		return true
	}

	return false
}

// isIgnoredFile returns true for files that are known to contain high entropy non-secret content
func isIgnoredFile(filename string) bool {
	ignored := map[string]bool{
		"go.sum":          true,
		"package-lock.json": true,
		"yarn.lock":       true,
		"Cargo.lock":      true,
		"composer.lock":   true,
		"Gemfile.lock":    true,
		"poetry.lock":     true,
	}
	
	// Ignore Enveil's own detector source to avoid self-detection
	if strings.Contains(filename, "internal/detector/") {
		return true
	}

	return ignored[filepath.Base(filename)]
}

// isBinaryExtension returns true for extensions that do not make sense to scan
func isBinaryExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	binary := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".pdf": true, ".zip": true, ".tar": true, ".gz": true,
		".exe": true, ".bin": true, ".so":  true, ".dylib": true,
		".mp4": true, ".mp3": true, ".mov": true, ".db":    true,
	}
	return binary[ext]
}