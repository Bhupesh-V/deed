package resolver

import (
	"deed/internal/models"
	"fmt"
)

// FindInsertionOrder performs a Topological Sort (Kahn's Algorithm)
// to resolve foreign key table dependencies.
func FindInsertionOrder(tables map[string]*models.Entity) ([]string, error) {
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize maps for every known table
	for name := range tables {
		inDegree[name] = 0
		graph[name] = []string{}
	}

	// Populate the graph by iterating over columns and their constraints
	for tableName, table := range tables {
		for _, col := range table.Columns {
			for _, c := range col.Constraint {
				// Check for foreign key constraints pointing to a valid referenced table
				if c.Type == "FOREIGN KEY" && c.ReferencedTable != nil {
					parent := *c.ReferencedTable

					// Ignore self-referencing foreign keys (e.g. employee.manager_id -> employee.id)
					// to avoid deadlock in Kahn's algorithm queue
					if parent == tableName {
						continue
					}

					// Only track dependency if the parent table exists in our execution scope
					if _, exists := tables[parent]; exists {
						graph[parent] = append(graph[parent], tableName)
						inDegree[tableName]++
					}
				}
			}
		}
	}

	// Collect all root nodes (in-degree == 0)
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var order []string

	// Process queue
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		order = append(order, curr)

		for _, child := range graph[curr] {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	// If order length does not match total tables, a circular dependency exists
	if len(order) != len(tables) {
		return nil, fmt.Errorf("circular dependency detected in database schema")
	}

	return order, nil
}
