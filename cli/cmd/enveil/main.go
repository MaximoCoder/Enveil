package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.2.1"

var rootCmd = &cobra.Command{
	Use:     "enveil",
	Short:   "Secure environment variable manager",
	Long:    "Enveil protects your secrets by keeping them encrypted and out of your filesystem.",
	Version: version,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}