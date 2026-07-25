package resolver

import (
	"deed/internal/models"
	"fmt"
	"sort"
)

func FindInsertionOrder(tables map[string]*models.Entity) ([][]string, error) {
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	// Track processed edges to avoid duplicate inDegree increments on composite foreign keys
	// Key format: "parentTable->childTable"
	seenEdges := make(map[string]bool)

	for name := range tables {
		inDegree[name] = 0
		graph[name] = []string{}
	}

	for tableName, table := range tables {
		for _, col := range table.Columns {
			for _, c := range col.Constraint {
				if c.Type == models.ForeignKey.String() && c.ReferencedTable != nil && *c.ReferencedTable != "" {
					parent := *c.ReferencedTable

					// Skip self-referencing foreign keys for now
					if parent == tableName {
						continue
					}

					// Ensure parent table is within execution scope
					if _, exists := tables[parent]; exists {
						edgeKey := fmt.Sprintf("%s->%s", parent, tableName)

						// Only add edge and increment inDegree ONCE per table pair
						if !seenEdges[edgeKey] {
							seenEdges[edgeKey] = true
							graph[parent] = append(graph[parent], tableName)
							inDegree[tableName]++
						}
					}
				}
			}
		}
	}

	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	sort.Strings(queue)

	var clusters [][]string
	processedCount := 0

	for len(queue) > 0 {
		var currentCluster []string
		var nextQueue []string

		for _, curr := range queue {
			currentCluster = append(currentCluster, curr)
			processedCount++

			for _, child := range graph[curr] {
				inDegree[child]--
				if inDegree[child] == 0 {
					nextQueue = append(nextQueue, child)
				}
			}
		}

		sort.Strings(nextQueue)
		clusters = append(clusters, currentCluster)
		queue = nextQueue
	}

	if processedCount != len(tables) {
		return nil, fmt.Errorf("circular dependency detected in database schema")
	}

	return clusters, nil
}
