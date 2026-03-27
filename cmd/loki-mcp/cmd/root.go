package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/grafana/loki-mcp/internal/handlers"
	"github.com/grafana/loki-mcp/internal/logging"
)

// Version is set via ldflags at build time
var Version = "dev"

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
	viper.SetEnvPrefix("LOKI")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Persistent flags
	rootCmd.PersistentFlags().StringP("query-filter", "f", "",
		"LogQL stream selector to restrict all queries (e.g., {namespace=\"prod\"})")
	rootCmd.PersistentFlags().StringP("log-level", "l", "info",
		"Log level: debug, info, warn, error")
	rootCmd.PersistentFlags().String("log-format", "text",
		"Log format: text, json")

	// Bind flags to viper
	_ = viper.BindPFlag("query-filter", rootCmd.PersistentFlags().Lookup("query-filter"))
	_ = viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	_ = viper.BindPFlag("log-format", rootCmd.PersistentFlags().Lookup("log-format"))

	// Bind Loki connection env vars
	_ = viper.BindEnv("url", "LOKI_URL")
	_ = viper.BindEnv("org-id", "LOKI_ORG_ID")
	_ = viper.BindEnv("username", "LOKI_USERNAME")
	_ = viper.BindEnv("password", "LOKI_PASSWORD")
	_ = viper.BindEnv("token", "LOKI_TOKEN")

	viper.SetDefault("url", "http://localhost:3100")
}
