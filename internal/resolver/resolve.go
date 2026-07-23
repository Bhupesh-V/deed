package main

import (
	"deed/internal/models"
	"fmt"
)

// FindInsertionOrder performs a Topological Sort (Kahn's Algorithm)
func FindInsertionOrder(tables map[string]*models.Entity) ([]string, error) {
	// Build an explicit adjacency list and calculate accurate in-degrees
	// graph[parent_table] = []child_tables
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize maps for every known table
	for name := range tables {
		inDegree[name] = 0
		graph[name] = []string{}
	}

	// Populate the graph and track structural dependencies
	for _, table := range tables {
		for _, c := range table.Constraints {
			// Match against your updated models.ForeignKey enum string
			if c.Type == models.ForeignKey && c.ForeignKey != nil {
				parent := c.ForeignKey.ParentTable

				// Ignore self-referencing foreign keys (e.g., employee.manager_id -> employee.id)
				// to prevent locking the table out of Kahn's queue.
				if parent == table.Name {
					continue
				}

				// Only map dependencies to tables present in our execution scope
				if _, exists := tables[parent]; exists {
					graph[parent] = append(graph[parent], table.Name)
					inDegree[table.Name]++
				}
			}
		}
	}

	// Collect all start nodes with an in-degree of 0
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var order []string

	// Process the queue using the pre-built lookup graph
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		order = append(order, curr)

		// Directly fetch and decrement the specific child nodes depending on this table
		for _, child := range graph[curr] {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	// Validate that all nodes were successfully sorted
	if len(order) != len(tables) {
		return nil, fmt.Errorf("circular dependency detected in database schema")
	}

	return order, nil
}
