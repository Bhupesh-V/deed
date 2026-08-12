package cmd

import (
	"context"
	"deed/database/postgres"
	"deed/internal/config"
	"deed/internal/deed"
	"deed/internal/models"
	"deed/internal/styles"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

var (
	dbURL     string
	tables    []string
	count     int64
	configUrl string
)

// TODO: fix for longer table list
func renderBanner(dbURL, configUrl string, tables []string, count int64) {
	// Format table list as clean badges
	var tableBadges []string
	for _, t := range tables {
		tableBadges = append(tableBadges, styles.Badge.Render(t))
	}
	tablesFormatted := strings.Join(tableBadges, " ")

	// Content assembly
	title := styles.Title.Render("🌱 deed seed")

	value, prefix := humanize.ComputeSI(float64(count))

	content := fmt.Sprintf(
		"%s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s",
		title,
		styles.Label.Render("DSN"), styles.Value.Render(dbURL),
		styles.Label.Render("Config"), styles.Value.Render(configUrl),
		styles.Label.Render("Count"), styles.Value.Render(fmt.Sprintf("%s (%.2f%s)", humanize.Comma(int64(count)), value, prefix)),
		styles.Label.Render("Tables"), tablesFormatted,
	)

	// Render framed card
	fmt.Println(styles.Box.Render(content))
}

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database with mock data",
	Run: func(cmd *cobra.Command, args []string) {
		renderBanner(dbURL, configUrl, tables, count)

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

	seedCmd.Flags().Int64Var(&count, "count", 100, "Number of records to insert")
	seedCmd.Flags().StringVar(&configUrl, "config", "", "Path to the configuration file")

	seedCmd.MarkFlagRequired("dsn")
}
