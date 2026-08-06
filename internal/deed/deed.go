package deed

import (
	"context"
	"deed/database"
	"deed/internal/config"
	"deed/internal/models"
	"deed/internal/resolver"
	"deed/internal/seeder"
	"fmt"
	"sync"

	"github.com/charmbracelet/lipgloss/tree"
	"golang.org/x/sync/errgroup"
)

type Deed struct {
	db     database.Database
	config *config.Config
	input  *models.Input
}

func New(db database.Database, cfg *config.Config, input *models.Input) *Deed {
	return &Deed{
		db:     db,
		config: cfg,
		input:  input,
	}
}

func (d *Deed) Start(ctx context.Context) error {
	entities, err := d.db.GetEntities(ctx)
	if err != nil {
		return err
	}
	allEntities := make(map[string]*models.Entity)
	for _, t := range entities {
		allEntities[t.Name] = &t
	}

	r := resolver.New()
	lookUps := d.input.Tables

	// Build tree UI & populate dependencies
	for _, target := range lookUps {
		fmt.Printf("\n--- Dependencies for '%s' ---\n\n", target)
		fmt.Println(r.GetDependencyTreeUI(target, allEntities, nil).Enumerator(tree.RoundedEnumerator))
	}

	if _, err := r.FindIngestionOrder(allEntities, lookUps); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	tablesToIngest := r.GetRequiredTables(lookUps, allEntities)

	fmt.Printf("\n--- Starting Ingestion (%d tables) ---\n\n", len(tablesToIngest))

	ready := make(map[string]chan struct{}, len(tablesToIngest))
	for _, table := range tablesToIngest {
		ready[table] = make(chan struct{})
	}

	g, ctx := errgroup.WithContext(ctx)
	var bounds sync.Map

	s := seeder.New(d.db, d.config)

	for _, table := range tablesToIngest {
		g.Go(func() error {
			entity := allEntities[table]

			// Wait ONLY for direct dependencies being processed in this run
			for _, dep := range entity.DirectDependencies() {
				if ch, ok := ready[dep]; ok {
					select {
					case <-ch:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}

			colNames, stream, err := s.Prepare(table, d.input.Count, allEntities, &bounds)
			if err != nil {
				return fmt.Errorf("prepare failed for %s: %w", table, err)
			}

			insertedRows, err := d.db.Ingest(ctx, table, colNames, stream)
			if err != nil {
				return fmt.Errorf("ingest failed for %s: %w", table, err)
			}

			if entity.PK().IsOrdered() {
				lb, up, err := d.db.GetBounds(ctx, table, entity.PK().Name)
				if err != nil {
					return fmt.Errorf("get bounds failed for %s: %w", table, err)
				}

				bounds.Store(table, &models.Bound{Lower: lb, Upper: up})
			}

			fmt.Printf("✅ Inserted %d rows into %s\n", insertedRows, table)

			// Unblock downstream dependents immediately
			close(ready[table])
			return nil
		})
	}

	return g.Wait()
}
