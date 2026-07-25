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
	connStr := d.config.DbUrl

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// Maximum number of connections in the pool
	config.MaxConns = 20
	// Minimum number of idle connections to keep alive
	config.MinConns = 5
	// Max time a connection can exist before being recreated
	config.MaxConnLifetime = 30 * time.Minute
	// Max time an idle connection can sit before being closed
	config.MaxConnIdleTime = 5 * time.Minute
	// Interval to check if idle connections are still healthy
	config.HealthCheckPeriod = 1 * time.Minute
	//Configure underlying pgx connection parameters if needed
	config.ConnConfig.ConnectTimeout = 5 * time.Second

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

	r := resolver.New()

	erGroups, err := r.FindInsertionOrder(tables)
	if err != nil {
		return err
	}

	// test
	for i, g := range erGroups {
		fmt.Printf("Group %d\n", i)
		for _, table := range g {
			noTables := len(tables[table].Columns)

			var fk int
			for _, col := range tables[table].Columns {
				for _, ctr := range col.Constraint {
					if ctr.Type == models.ForeignKey.String() {
						fk++
					}
				}
			}
			// for Group 0: total FK count should be 0, proving our sorting worked
			fmt.Printf("\ttable: %v, no.of columns:%d, total FKs: %d\n\n", table, noTables, fk)
		}
	}

	lookUp := "app"
	fmt.Printf("\n--- Dependencies for '%s' ---\n", lookUp)
	r.GetDependencyTree(lookUp, tables, "", true, 0, nil)

	return nil
}
