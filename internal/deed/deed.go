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
		fmt.Printf("\n--- Dependencies for '%s' ---\n", target)
		r.GetDependencyTree(target, allEntities, "", true, 0, nil)
	}

	fmt.Printf("\n--- Starting Ingestion ---\n\n")

	// Find grouped ingestion order for all tables in lookUps AND all its prerequisites
	erGroups, err := r.FindIngestionOrder(allEntities, lookUps)
	if err != nil {
		log.Fatal(err)
	}

	s := seeder.New(d.db)

	for _, g := range erGroups {
		// fmt.Printf("Group %d\n", i)
		for _, table := range g {
			entity := allEntities[table]

			fmt.Println("Ingesting data in", entity.Name)

			tableRules := make(map[string]models.GenerationRule)
			// main table takes input from cli
			var rowCount = d.input.Count

			if tableCfg, ok := d.config.Rules.Rules.Tables[entity.Name]; ok {
				if tableCfg.Count > 0 {
					// override cli from config
					rowCount = tableCfg.Count
				}
				for colName, colRule := range tableCfg.Columns {
					tableRules[colName] = models.GenerationRule{
						Type:         colRule.Type,
						RegexPattern: colRule.Pattern,
					}
				}
			}

			colNames, stream := s.CreateStream(entity, rowCount, tableRules)

			// Database layer consumes stream (Postgres uses CopyFrom, MySQL uses batch INSERT)
			insertedRows, err := d.db.Ingest(ctx, entity, colNames, stream)
			if err != nil {
				return err
			}

			fmt.Printf("\t✅ Successfully inserted %d rows into %s\n", insertedRows, entity.Name)
			// noTables := len(allEntities[table].Columns)

			// var fk int
			// for _, col := range allEntities[table].Columns {
			// 	for _, ctr := range col.Constraint {
			// 		if ctr.Type == models.ForeignKey.String() {
			// 			fk++
			// 		}
			// 	}
			// }
			// // for Group 0: total FK count should be 0, proving our sorting worked
			// fmt.Printf("\ttable: %v, no.of columns:%d, total FKs: %d\n\n", table, noTables, fk)
		}
	}

	return nil
}
