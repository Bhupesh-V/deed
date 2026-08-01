package main

import (
	"context"
	"deed/database/postgres"
	"deed/internal/config"
	"deed/internal/deed"
	"log"
)

func main() {
	ctx := context.Background()
	dsn := "postgres://postgres:my_secure_password@127.0.0.1:5433/deed-commerce"

	db, err := postgres.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	cfg := &config.Config{DSN: dsn}

	app := deed.New(db, cfg)

	if err := app.Start(ctx); err != nil {
		log.Fatalf("Deed execution failed: %v", err)
	}
}
