package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ideamans/rakkokeyword-cli/internal/config"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage the rakkokeyword API key",
	Long: "The API key is resolved in this order: --api-key, then the\n" +
		"RAKKOKEYWORD_API_KEY environment variable, then RAKKO_API_KEY, then the\n" +
		"config file. Keys are issued on the STANDARD plan (up to 5) in the\n" +
		"rakkokeyword account settings.",
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show where the API key comes from and which endpoint it talks to",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Config file : %s\n", config.Path())
		fmt.Fprintf(out, "API base    : %s\n", cfg.BaseURLResolved())

		key := cfg.APIKeyResolved(flagAPIKey)
		if key == "" {
			fmt.Fprintln(out, "API key     : (not set)")
			fmt.Fprintf(out, "              set %s, pass --api-key, or run `rakko auth set-api-key <key>`\n", config.EnvAPIKey)
			return nil
		}
		fmt.Fprintf(out, "API key     : %s (from %s)\n", maskKey(key), cfg.APIKeySource(flagAPIKey))
		return nil
	},
}

// maskKey hides all but the last 4 characters of a key.
func maskKey(key string) string {
	if len(key) <= 4 {
		return strings.Repeat("*", len(key))
	}
	return strings.Repeat("*", len(key)-4) + key[len(key)-4:]
}

var authSetAPIKeyCmd = &cobra.Command{
	Use:   "set-api-key <api-key>",
	Short: "Store the API key in the config file",
	Long: "Writes the key to the config file with owner-only permissions.\n\n" +
		"Prefer the RAKKOKEYWORD_API_KEY environment variable on shared machines\n" +
		"and in CI — it leaves nothing on disk.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		cfg.APIKey = args[0]
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "API key set to %s\nSaved: %s\n", maskKey(args[0]), config.Path())
		return nil
	},
}

var authSetFormatCmd = &cobra.Command{
	Use:   "set-format <format>",
	Short: "Store the default output format (table / json / jsonl / csv)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		cfg.DefaultFormat = args[0]
		if _, err := resolveFormatValue(args[0]); err != nil {
			return err
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Default format set to %q\nSaved: %s\n", args[0], config.Path())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authSetAPIKeyCmd)
	authCmd.AddCommand(authSetFormatCmd)
}
