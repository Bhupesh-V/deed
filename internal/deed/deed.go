package deed

import (
	"context"
	"deed/database"
	"deed/internal/config"
	"deed/internal/feeder"
	"deed/internal/models"
	"deed/internal/resolver"
	"deed/internal/styles"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/lipgloss/tree"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
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

// newSpeedDecorator accurately tracks throughput for fast/small table runs
func newSpeedDecorator() decor.Decorator {
	var startTime time.Time
	var endTime time.Time
	var startOnce sync.Once
	var endOnce sync.Once

	return decor.Any(func(s decor.Statistics) string {
		// Mark start time when first row arrives
		if s.Current > 0 {
			startOnce.Do(func() {
				startTime = time.Now()
			})
		}

		// Lock end time as soon as ingestion hits 100%
		if s.Completed || (s.Total > 0 && s.Current >= s.Total) {
			endOnce.Do(func() {
				endTime = time.Now()
			})
		}

		// Calculate elapsed time (fixed if completed, dynamic if running)
		var elapsed float64
		if !endTime.IsZero() {
			elapsed = endTime.Sub(startTime).Seconds()
		} else {
			elapsed = time.Since(startTime).Seconds()
		}

		if elapsed <= 0.0001 {
			elapsed = 0.0001
		}

		spd := float64(s.Current) / elapsed
		if spd >= 1000000 {
			return fmt.Sprintf("%11.2fM rows/s", spd/1000000)
		}
		return fmt.Sprintf("%12.0f rows/s", spd)
	}, decor.WCSyncSpace)
}

func (d *Deed) Start(ctx context.Context) error {
	entities, err := d.db.GetEntities(ctx)
	if err != nil || entities == nil {
		return fmt.Errorf("unable to fetch db schema: %w", err)
	}

	allEntities := make(map[string]*models.Entity)
	for _, t := range entities {
		allEntities[t.Name] = &t
	}

	r := resolver.New()
	lookUps := d.input.Tables

	// Build tree UI & populate dependencies
	for _, target := range lookUps {
		fmt.Println(styles.Title.Render(fmt.Sprintf("\nDependencies for '%s'\n", target)))
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

	f, err := feeder.New(d.db, d.config, d.input, allEntities)
	if err != nil {
		return err
	}

	progressContainer := mpb.NewWithContext(ctx)
	bars := make(map[string]*mpb.Bar, len(tablesToIngest))

	fmt.Println(styles.Title.Render(fmt.Sprintf("\nSeeding Data (%d tables)\n", len(tablesToIngest))))

	for i, table := range tablesToIngest {
		totalRows := f.GetTableCount(table)

		bar, _ := progressContainer.Add(
			totalRows,
			mpb.BarStyle().
				Filler("━").
				Tip("🌱").
				Padding("─").
				Build(),
			mpb.BarPriority(1000+i), // Initial placement priority
			mpb.PrependDecorators(
				decor.Name(fmt.Sprintf("%-24s", table), decor.WCSyncSpace),
				decor.CountersNoUnit("%7d / %-7d", decor.WCSyncSpace),
			),
			mpb.AppendDecorators(
				decor.Percentage(decor.WC{W: 5}),
				newSpeedDecorator(),
				decor.OnComplete(
					decor.Name(" "+styles.Dim.Render("⏳"), decor.WCSyncSpace),
					" "+styles.Success.Render("✔ Done"),
				),
			),
		)
		bars[table] = bar
	}

	g, ctx := errgroup.WithContext(ctx)
	var bounds sync.Map
	var completionOrder atomic.Int32

	start := time.Now()

	for _, table := range tablesToIngest {
		g.Go(func() error {
			entity := allEntities[table]
			bar := bars[table]

			// Wait for direct dependencies
			for _, dep := range entity.DirectDependencies() {
				if ch, ok := ready[dep]; ok {
					select {
					case <-ch:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}

			colNames, stream, err := f.Prepare(ctx, table, d.input.Count, allEntities, &bounds)
			if err != nil {
				return fmt.Errorf("prepare failed for %s: %w", table, err)
			}

			onProgress := func(n int) {
				bar.IncrBy(n)
			}

			_, err = d.db.Ingest(ctx, table, colNames, stream, onProgress)
			if err != nil {
				return fmt.Errorf("ingest failed for %s: %w", table, err)
			}

			if entity.PK() != nil {
				lb, up, err := d.db.GetBounds(ctx, table, entity.PK().Name)
				if err != nil {
					return fmt.Errorf("get bounds failed for %s: %w", table, err)
				}

				bounds.Store(table, &models.Bound{Lower: lb, Upper: up})
			}

			// Shift completed bar to top based on completion sequence
			order := completionOrder.Add(1)
			bar.SetPriority(int(order))

			close(ready[table])
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	progressContainer.Wait()

	elapsed := time.Since(start)
	finalMessage := fmt.Sprintf("\n✨ Ingestion complete across all tables. Took %.2f seconds\n", elapsed.Seconds())
	fmt.Println(styles.Success.Render(finalMessage))

	return nil
}
