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
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

var (
	dbURL     string
	tables    []string
	count     int
	configUrl string
)

// TODO: fix for longer table list
func renderBanner(dbURL, configUrl string, tables []string, count int) {
	// Colors
	primary := lipgloss.Color("42") // Spring green
	// muted := lipgloss.Color("241")    // Cool gray
	accent := lipgloss.Color("212")   // Soft pink
	highlight := lipgloss.Color("86") // Cyan

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primary).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(accent).
		Width(10)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	badgeStyle := lipgloss.NewStyle().
		Foreground(highlight).
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Bold(true)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primary).
		Padding(1, 2).
		Margin(1, 0)

	// Format table list as clean badges
	var tableBadges []string
	for _, t := range tables {
		tableBadges = append(tableBadges, badgeStyle.Render(t))
	}
	tablesFormatted := strings.Join(tableBadges, " ")

	// Content assembly
	title := titleStyle.Render("🌱 deed seed")

	value, prefix := humanize.ComputeSI(float64(count))

	content := fmt.Sprintf(
		"%s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s",
		title,
		labelStyle.Render("DSN"), valueStyle.Render(dbURL),
		labelStyle.Render("Config"), valueStyle.Render(configUrl),
		labelStyle.Render("Count"), valueStyle.Render(fmt.Sprintf("%s (%.2f%s)", humanize.Comma(int64(count)), value, prefix)),
		labelStyle.Render("Tables"), tablesFormatted,
	)

	// Render framed card
	fmt.Println(boxStyle.Render(content))
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

	seedCmd.Flags().IntVar(&count, "count", 100, "Number of records to insert")
	seedCmd.Flags().StringVar(&configUrl, "config", "", "Path to the configuration file")

	seedCmd.MarkFlagRequired("dsn")
}
