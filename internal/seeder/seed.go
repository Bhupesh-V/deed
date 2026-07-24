package seeder

import (
	"deed/internal/models"
	"fmt"
	"math/rand"
)

type Seeder struct {
	// Registry to keep track of generated primary keys per table
	// e.g., GeneratedIDs["countries"] = [1, 2, 3]
	GeneratedIDs map[string][]int
}

// NewSeeder initializes a seeder instance with a ready map allocation.
func NewSeeder() *Seeder {
	return &Seeder{
		GeneratedIDs: make(map[string][]int),
	}
}

// GenerateMockData processes the tables in topological order
func (sd *Seeder) GenerateMockData(tables map[string]*models.Entity, order []string, rowsPerTable int) {
	fmt.Println("=== Starting Safe Mock Data Generation ===")

	for _, tableName := range order {
		table := tables[tableName]
		fmt.Printf("\nGenerating data for table: [%s]\n", tableName)

		// Create a lookup map of local_column -> parent_table for this table's foreign keys.
		fkLookup := make(map[string]string)
		for _, col := range table.Columns {
			for _, c := range col.Constraint {
				if c.Type == "FOREIGN KEY" && c.ReferencedTable != nil {
					fkLookup[col.Name] = *c.ReferencedTable
				}
			}
		}

		for i := 1; i <= rowsPerTable; i++ {
			// Simulate generating a unique serial Primary Key ID
			generatedID := len(sd.GeneratedIDs[tableName]) + 1
			sd.GeneratedIDs[tableName] = append(sd.GeneratedIDs[tableName], generatedID)

			fmt.Printf("  Row #%d -> Columns: [id: %d", i, generatedID)

			// Process each column exactly once
			for _, col := range table.Columns {
				if col.Name == "id" {
					continue
				}

				// Check if this specific column is part of a foreign key dependency
				if parentTable, isFK := fkLookup[col.Name]; isFK {
					parentIDs := sd.GeneratedIDs[parentTable]

					// Handle edge case where parent table contains no generated rows
					if len(parentIDs) == 0 {
						fmt.Printf(", %s: NULL (No parent data)", col.Name)
						continue
					}

					// Fetch a random valid ID already generated for the parent table
					randomParentID := parentIDs[rand.Intn(len(parentIDs))]
					fmt.Printf(", %s: %d (FK -> %s)", col.Name, randomParentID, parentTable)
				} else {
					// Regular column (Generate generic mock data)
					fmt.Printf(", %s: mock_value_%d", col.Name, generatedID)
				}
			}
			fmt.Println("]")
		}
	}
}
