package deed

import (
	"context"
	"deed/database"
	"deed/internal/config"
	"deed/internal/models"
	"deed/internal/resolver"
	"deed/internal/seeder"
	"fmt"
	"log"

	"github.com/charmbracelet/lipgloss/tree"
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

	// Populate r.Dependencies map
	for _, target := range lookUps {
		fmt.Printf("\n--- Dependencies for '%s' ---\n\n", target)
		fmt.Println(r.GetDependencyTreeUI(target, allEntities, nil).Enumerator(tree.RoundedEnumerator))
	}

	fmt.Printf("\n--- Starting Ingestion ---\n\n")

	// Find grouped ingestion order for all tables in lookUps AND all its prerequisites
	erGroups, err := r.FindIngestionOrder(allEntities, lookUps)
	if err != nil {
		log.Fatal(err)
	}

	s := seeder.New(d.db, d.config)

	bounds := make(map[string]*models.Bound)

	for _, g := range erGroups {
		for _, table := range g {
			entity := allEntities[table]

			colNames, stream, err := s.Prepare(table, d.input.Count, allEntities, bounds)
			if err != nil {
				return err
			}
			// Database layer consumes stream (Postgres uses CopyFrom, MySQL uses batch INSERT)
			insertedRows, err := d.db.Ingest(ctx, table, colNames, stream)
			if err != nil {
				return err
			}

			if entity.GetPK().IsOrdered() {
				lb, up, err := d.db.GetBounds(ctx, table, entity.GetPK().Name)
				if err != nil {
					return err
				}
				bounds[table] = &models.Bound{Lower: lb, Upper: up}
			}

			fmt.Printf("✅ Inserted %d rows into %s\n", insertedRows, table)
		}
	}

	return nil
}
