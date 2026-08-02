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
}

func New(db database.Database, cfg *config.Config) *Deed {
	return &Deed{
		db:     db,
		config: cfg,
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
	lookUps := []string{"delivery_proofs"}

	// Populate r.Dependencies map
	for _, target := range lookUps {
		fmt.Printf("\n--- Dependencies for '%s' ---\n", target)
		r.GetDependencyTree(target, allEntities, "", true, 0, nil)
	}

	fmt.Printf("\n--- Getting Ingestion Order ---\n\n")

	// Find grouped ingestion order for all tables in lookUps AND all its prerequisites
	erGroups, err := r.FindIngestionOrder(allEntities, lookUps)
	if err != nil {
		log.Fatal(err)
	}

	s := seeder.New(d.db)

	rules := map[string]models.GenerationRule{
		// Level 1: users
		"email": {
			Type:         "regex",
			RegexPattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
		},
		"username": {
			Type:         "regex",
			RegexPattern: `^[a-zA-Z0-9_-]{3,30}$`,
		},
		"password_hash": {
			Type:         "regex",
			RegexPattern: `^\$2[ayb]\$[0-9]{2}\$[A-Za-z0-9./]{53}$`,
		},

		// Level 2: orders
		"order_number": {
			Type:         "regex",
			RegexPattern: `^ORD-[0-9]{8}-[0-9]{4}$`,
		},
		"status": {
			Type:         "regex",
			RegexPattern: `^(PENDING|PAID|SHIPPED|COMPLETED|CANCELLED)$`,
		},
		"total_amount": {
			Type:         "regex",
			RegexPattern: `^[0-9]{1,10}\.[0-9]{2}$`,
		},

		// Level 3: shipments
		"tracking_number": {
			Type:         "regex",
			RegexPattern: `^[A-Z0-9]{8,100}$`,
		},

		// Level 4: shipment_tracking_events
		"location": {
			Type:         "regex",
			RegexPattern: `^[A-Za-z\s.-]+,\s*[A-Z]{2}(\s*[0-9]{5})?$`,
		},
		"status_description": {
			Type:         "regex",
			RegexPattern: `^[A-Za-z0-9\s.,-]{1,255}$`,
		},

		// Level 5: delivery_proofs
		"recipient_signature_url": {
			Type:         "regex",
			RegexPattern: `^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/.*)?\.(png|jpg|jpeg|svg)$`,
		},
		"photo_url": {
			Type:         "regex",
			RegexPattern: `^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/.*)?\.(png|jpg|jpeg|webp)$`,
		},

		// Level 6: proof_verifications
		"confidence_score": {
			Type:         "regex",
			RegexPattern: `^(100\.00|[0-9]{1,2}\.[0-9]{2})$`,
		},
	}

	for i, g := range erGroups {
		fmt.Printf("Group %d\n", i)
		for _, table := range g {

			entity := allEntities[table]

			colNames, stream := s.CreateStream(entity, 1, rules)

			// Database layer consumes stream (Postgres uses CopyFrom, MySQL uses batch INSERT)
			insertedRows, err := d.db.BulkInsert(ctx, entity, colNames, stream)
			if err != nil {
				return err
			}

			fmt.Printf("✅ Successfully inserted %d rows into %s\n", insertedRows, entity.Name)
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
