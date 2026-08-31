package cmd

import (
	"context"
	"deed/database/postgres"
	"deed/internal/config"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var configOutput string

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage deed config",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a starter deed config file from a database schema",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		db, err := postgres.New(ctx, dbURL)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		entities, err := db.GetEntities(ctx)
		if err != nil {
			log.Fatalf("Failed to read database schema: %v", err)
		}

		fileCfg := config.GenerateFromEntities(entities)

		out, err := json.MarshalIndent(fileCfg, "", "    ")
		if err != nil {
			log.Fatalf("Failed to marshal config: %v", err)
		}

		if err := os.WriteFile(configOutput, out, 0o644); err != nil {
			log.Fatalf("Failed to write config file %q: %v", configOutput, err)
		}

		fmt.Printf("deed config written to %s\n", configOutput)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)

	configInitCmd.Flags().StringVar(&dbURL, "dsn", "", "Database connection string (e.g., postgres://...)")
	configInitCmd.Flags().StringVar(&configOutput, "output", "", "Path to write the generated configuration file")

	configInitCmd.MarkFlagRequired("dsn")
	configInitCmd.MarkFlagRequired("output")
}
