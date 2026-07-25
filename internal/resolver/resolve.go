package resolver

import (
	"deed/internal/models"
	"fmt"
	"sort"
)

type Resolver struct {
	// DependencyGraph maps ParentTable -> []ChildTables
	dependencyGraph map[string][]string
}

func New() *Resolver {
	return &Resolver{
		dependencyGraph: make(map[string][]string),
	}
}

func (r *Resolver) FindInsertionOrder(tables map[string]*models.Entity) ([][]string, error) {
	graph := r.dependencyGraph
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
	// fmt.Println(processedCount)
	// fmt.Println(len(tables))

	if processedCount != len(tables) {
		return nil, fmt.Errorf("circular dependency detected in database schema")
	}

	return clusters, nil
}

// GetDependencyTree recursively prints all parent tables with clean tree alignment.
func (r *Resolver) GetDependencyTree(tableName string, tables map[string]*models.Entity, prefix string, isLast bool, depth int, visited map[string]bool) {
	if visited == nil {
		visited = make(map[string]bool)
	}

	if visited[tableName] {
		return
	}
	visited[tableName] = true

	// Root node handling
	if depth == 0 {
		fmt.Printf("🎯 %s\n", tableName)
	} else {
		marker := "├── "
		if isLast {
			marker = "└── "
		}
		fmt.Printf("%s%s🔗 %s\n", prefix, marker, tableName)
	}

	entity, exists := tables[tableName]
	if !exists {
		return
	}

	// Collect all valid direct parent tables first
	type Edge struct {
		Parent string
	}
	var parents []Edge
	seenParents := make(map[string]bool)

	for _, col := range entity.Columns {
		for _, c := range col.Constraint {
			if c.Type == models.ForeignKey.String() && c.ReferencedTable != nil && *c.ReferencedTable != "" {
				parent := *c.ReferencedTable
				if parent != tableName && !seenParents[parent] {
					seenParents[parent] = true
					parents = append(parents, Edge{Parent: parent})
				}
			}
		}
	}

	// Prepare prefix for children
	var childPrefix string
	if depth > 0 {
		if isLast {
			childPrefix = prefix + "    "
		} else {
			childPrefix = prefix + "│   "
		}
	}

	// Recurse over parents with correct last-sibling flags
	for i, p := range parents {
		isLastParent := (i == len(parents)-1)

		// Copy visited map per branch to allow shared dependencies across distinct subtrees
		branchVisited := make(map[string]bool)
		for k, v := range visited {
			branchVisited[k] = v
		}

		r.GetDependencyTree(p.Parent, tables, childPrefix, isLastParent, depth+1, branchVisited)
	}
}
