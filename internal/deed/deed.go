package deed

import (
	"context"
	"deed/database/postgres"
	"deed/internal/config"
	"deed/internal/models"
	"deed/internal/resolver"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Deed struct {
	config *config.Config
}

func New(dburl string) *Deed {
	return &Deed{
		&config.Config{
			DbUrl: dburl,
		},
	}
}

func (d *Deed) Start(ctx context.Context) error {
	// gather metadata from postgres
	// resolve deps
	// seed

	connStr := d.config.DbUrl

	// 1. Parse the base connection string into a Config struct
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// 2. Apply native pool settings directly to the struct
	config.MaxConns = 20                       // Maximum number of connections in the pool
	config.MinConns = 5                        // Minimum number of idle connections to keep alive
	config.MaxConnLifetime = 30 * time.Minute  // Max time a connection can exist before being recreated
	config.MaxConnIdleTime = 5 * time.Minute   // Max time an idle connection can sit before being closed
	config.HealthCheckPeriod = 1 * time.Minute // Interval to check if idle connections are still healthy

	// 3. Configure underlying pgx connection parameters if needed
	config.ConnConfig.ConnectTimeout = 5 * time.Second

	// 4. Create the pool using your custom configuration
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	pg := postgres.New(pool)
	entities, err := pg.GetEntities(ctx)
	if err != nil {
		return err
	}
	tables := make(map[string]*models.Entity)
	for _, t := range entities {
		tables[t.Name] = &t
	}

	order, err := resolver.FindInsertionOrder(tables)
	if err != nil {
		return err
	}

	fmt.Printf("Safe Insertion Order: %v\n\n", order)

	return nil
}
