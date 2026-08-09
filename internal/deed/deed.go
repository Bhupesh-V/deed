package deed

import (
	"context"
	"deed/database"
	"deed/internal/config"
	"deed/internal/feeder"
	"deed/internal/models"
	"deed/internal/resolver"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/charmbracelet/lipgloss/tree"
	"golang.org/x/sync/errgroup"
)

const numWorkersPerTable = 10

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

	ready := make(map[string]chan struct{}, len(tablesToIngest))
	for _, table := range tablesToIngest {
		ready[table] = make(chan struct{})
	}

	g, ctx := errgroup.WithContext(ctx)
	var bounds sync.Map

	f, err := feeder.New(d.db, d.config, d.input, allEntities)
	if err != nil {
		return err
	}

	fmt.Printf("\n--- Starting Ingestion (%d tables) ---\n\n", len(tablesToIngest))
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

			// Determine total row count for this table
			totalCount := d.input.Count
			if tableCfg, ok := d.config.Rules.Rules.Tables[entity.Name]; ok && tableCfg.Count > 0 {
				totalCount = tableCfg.Count
			}

			// Determine parallelism for this table
			workers := numWorkersPerTable
			if totalCount < 100000 {
				workers = 1 // Force single COPY stream for lookup/small tables
			}

			chunkSize := totalCount / int64(workers)
			var totalInserted atomic.Int64
			// var colNames []string

			tableGroup, tableCtx := errgroup.WithContext(ctx)

			for w := 0; w < workers; w++ {
				workerID := w
				startOffset := int64(workerID) * chunkSize
				rowsForWorker := chunkSize
				if workerID == workers-1 {
					// Handle remaining row count division remainders on last worker
					rowsForWorker = totalCount - startOffset
				}

				tableGroup.Go(func() error {
					// NOTE: Pass startOffset into feeder so row generator sequences remain deterministic
					cols, stream, err := f.Prepare(tableCtx, table, rowsForWorker, startOffset, allEntities, &bounds)
					if err != nil {
						return fmt.Errorf("prepare chunk failed for %s (worker %d): %w", table, workerID, err)
					}

					// if workerID == 0 {
					// 	colNames = cols
					// }

					inserted, err := d.db.Ingest(tableCtx, table, cols, stream)
					if err != nil {
						return fmt.Errorf("ingest chunk failed for %s (worker %d): %w", table, workerID, err)
					}

					totalInserted.Add(inserted)
					return nil
				})
			}

			if err := tableGroup.Wait(); err != nil {
				return err
			}

			// colNames, stream, err := f.Prepare(ctx, table, d.input.Count, allEntities, &bounds)
			// if err != nil {
			// 	return fmt.Errorf("prepare failed for %s: %w", table, err)
			// }

			// insertedRows, err := d.db.Ingest(ctx, table, colNames, stream)
			// if err != nil {
			// 	return fmt.Errorf("ingest failed for %s: %w", table, err)
			// }

			if entity.PK() != nil {
				lb, up, err := d.db.GetBounds(ctx, table, entity.PK().Name)
				if err != nil {
					return fmt.Errorf("get bounds failed for %s: %w", table, err)
				}

				bounds.Store(table, &models.Bound{Lower: lb, Upper: up})
			}

			fmt.Printf("✅ Inserted %d rows into %s\n", totalInserted.Load(), table)

			// Unblock downstream dependents immediately
			close(ready[table])
			return nil
		})
	}

	return g.Wait()
}
