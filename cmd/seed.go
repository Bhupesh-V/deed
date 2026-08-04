package cmd

import (
	"context"
	"deed/database/postgres"
	"deed/internal/config"
	"deed/internal/deed"
	"deed/internal/models"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	dbURL     string
	tables    []string
	count     int
	configUrl string
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database with mock data",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🌱 deed seed")
		fmt.Println("-----------------------------")
		fmt.Printf("DSN:     %s\n", dbURL)
		fmt.Printf("Tables:  %v\n", tables)
		fmt.Printf("Count:   %d\n", count)
		fmt.Printf("Config:  %s\n", configUrl)

		ctx := context.Background()
		cfg := config.New()

		if err := cfg.LoadFromFile(configUrl); err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		db, err := postgres.New(ctx, dbURL)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		app := deed.New(db, cfg, &models.Input{
			DSN:    dbURL,
			Tables: tables,
			Count:  count,
			Config: configUrl,
		})

		if err := app.Start(ctx); err != nil {
			fmt.Printf("\ndeed execution failed: \n\n%v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(seedCmd)

	seedCmd.Flags().StringVar(&dbURL, "dsn", "", "Database connection string (e.g., postgres://...)")

	// StringSliceVar automatically handles comma-separated values like "app,users"
	seedCmd.Flags().StringSliceVar(&tables, "tables", []string{}, "Comma-separated list of tables to seed")

	seedCmd.Flags().IntVar(&count, "count", 100, "Number of records to insert")
	seedCmd.Flags().StringVar(&configUrl, "config", "", "Path to the configuration file")

	seedCmd.MarkFlagRequired("dsn")
}
