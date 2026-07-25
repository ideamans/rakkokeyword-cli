package main

import (
	"github.com/ideamans/go-llm-cli-kit/llmcmd"
	"github.com/spf13/cobra"

	"github.com/ideamans/rakkokeyword-cli/internal/llmdocs"
)

// llmConfig describes the `rakko llm` subcommand.
func llmConfig() llmcmd.Config {
	return llmcmd.Config{Docs: llmdocs.Docs()}
}

func init() {
	llmcmd.AddTo(rootCmd, llmConfig())

	// Bare `rakko` (no subcommand) prints standard help instead of erroring.
	rootCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	}
}
