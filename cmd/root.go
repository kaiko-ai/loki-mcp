package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/kaiko-ai/loki-mcp/internal/handlers"
	"github.com/kaiko-ai/loki-mcp/internal/logging"
)

// envFlag registers a string flag on fs, binds it to viper, and appends the
// resolved env var to the usage string so --help surfaces it. The env var is
// derived as <LOKI_PREFIX>_<NAME> with dashes replaced by underscores, matching
// viper's AutomaticEnv + key replacer configuration.
func envFlag(fs *pflag.FlagSet, name, shorthand, def, usage string) {
	env := strings.ReplaceAll(strings.ToUpper(name), "-", "_")
	if prefix := viper.GetEnvPrefix(); prefix != "" {
		env = prefix + "_" + env
	}
	fs.StringP(name, shorthand, def, fmt.Sprintf("%s [env: %s]", usage, env))
	_ = viper.BindPFlag(name, fs.Lookup(name))
}

// Version is set via ldflags at build time
var Version = "dev"

// Configure viper at package-init time (before any file's init() registers
// flags), so envFlag can resolve the env prefix regardless of init() order.
var _ = func() bool {
	viper.SetEnvPrefix("LOKI")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return true
}()

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "loki-mcp",
	Short: "MCP server for Grafana Loki",
	Long: `An MCP (Model Context Protocol) server that provides Grafana Loki query tools
to AI assistants like Claude Desktop, Cursor, and n8n.

Use subcommands to run the server in different modes:
  - stdio: For Claude Desktop and similar MCP clients
  - http:  For web-based clients and n8n`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := logging.Init(viper.GetString("log-level"), viper.GetString("log-format")); err != nil {
			return err
		}

		if queryFilter := viper.GetString("query-filter"); queryFilter != "" {
			if err := handlers.InitializeFilterConfig(queryFilter); err != nil {
				return err
			}
			logging.Infof("Query filter configured: %s", queryFilter)
		}

		// Ping Loki to verify connectivity before starting the server
		params := &handlers.LokiParams{
			URL:      viper.GetString("url"),
			Username: viper.GetString("username"),
			Password: viper.GetString("password"),
			Token:    viper.GetString("token"),
			Org:      viper.GetString("org-id"),
		}
		if err := handlers.PingLoki(params); err != nil {
			return fmt.Errorf("loki connectivity check failed: %w", err)
		}
		logging.Infof("Loki reachable at %s", params.URL)

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Persistent flags
	pf := rootCmd.PersistentFlags()
	envFlag(pf, "query-filter", "f", "",
		"LogQL stream selector to restrict all queries (e.g., {namespace=\"prod\"})")
	envFlag(pf, "log-level", "l", "info", "Log level: debug, info, warn, error")
	envFlag(pf, "log-format", "", "text", "Log format: text, json")

	// Loki connection flags
	envFlag(pf, "url", "u", "http://localhost:3100", "Loki server URL")
	envFlag(pf, "org-id", "", "", "Loki org ID for multi-tenant setups (X-Scope-OrgID)")
	envFlag(pf, "username", "", "", "Basic auth username")
	envFlag(pf, "password", "", "", "Basic auth password")
	envFlag(pf, "token", "", "", "Bearer token")
}
