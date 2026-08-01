package resolver

import (
	"deed/internal/models"
	"fmt"
	"sort"
)

type Resolver struct {
	// DependencyGraph maps ParentTable -> []ChildTables
	dependencyGraph map[string][]string
	// Dependencies maps TableName -> Set of AllPrerequisiteTables
	Dependencies map[string]map[string]struct{}
}

func New() *Resolver {
	return &Resolver{
		dependencyGraph: make(map[string][]string),
		Dependencies:    make(map[string]map[string]struct{}),
	}
}

// GetDependenciesForTables returns a map containing all unique dependencies for the given list of tables.
func (r *Resolver) GetDependenciesForTables(tables []string) map[string]struct{} {
	result := make(map[string]struct{})

	for _, table := range tables {
		if deps, exists := r.Dependencies[table]; exists {
			for dep := range deps {
				result[dep] = struct{}{}
			}
		}
	}

	return result
}

func (r *Resolver) FindIngestionOrder(tables map[string]*models.Entity, lookups []string) ([][]string, error) {
	// Build lookupSet with requested tables AND all their recursive dependencies
	lookupSet := make(map[string]struct{})

	if len(lookups) > 0 {
		// Leverage GetDependenciesForTables to pull all recursive ancestor tables
		deps := r.GetDependenciesForTables(lookups)
		for dep := range deps {
			lookupSet[dep] = struct{}{}
		}
		// Also include the requested tables themselves
		for _, l := range lookups {
			lookupSet[l] = struct{}{}
		}
	} else {
		// If no lookups provided, target all tables
		for t := range tables {
			lookupSet[t] = struct{}{}
		}
	}

	// Standard topological sort across the full schema
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
			parent, _, isFK := col.GetFK()
			if !isFK || parent == tableName {
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
			// Filter using the expanded lookupSet (target + recursive prerequisites)
			if _, exists := lookupSet[curr]; exists {
				currentCluster = append(currentCluster, curr)
			}
			processedCount++

			for _, child := range graph[curr] {
				inDegree[child]--
				if inDegree[child] == 0 {
					nextQueue = append(nextQueue, child)
				}
			}
		}

		sort.Strings(nextQueue)

		if len(currentCluster) > 0 {
			clusters = append(clusters, currentCluster)
		}

		queue = nextQueue
	}
	// fmt.Println(processedCount)
	// fmt.Println(len(tables))

	if processedCount != len(tables) {
		return nil, fmt.Errorf("circular dependency detected in database schema")
	}

	return clusters, nil
}

// GetDependencyTree recursively prints all parent tables and populates r.Dependencies.
func (r *Resolver) GetDependencyTree(tableName string, tables map[string]*models.Entity, prefix string, isLast bool, depth int, visited map[string]bool) {
	if visited == nil {
		visited = make(map[string]bool)
	}

	// Ensure map entry exists for this table
	if _, exists := r.Dependencies[tableName]; !exists {
		r.Dependencies[tableName] = make(map[string]struct{})
	}

	// If already visited, stop printing to avoid infinite loops,
	// but the dependencies for this table are already stored in r.Dependencies[tableName].
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
		parent, _, isFK := col.GetFK()
		if isFK && parent != tableName && !seenParents[parent] {
			seenParents[parent] = true
			parents = append(parents, Edge{Parent: parent})
			// Record direct dependency
			r.Dependencies[tableName][parent] = struct{}{}
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

		// Always merge parent's transitive dependencies into the current table
		if parentDeps, ok := r.Dependencies[p.Parent]; ok {
			for dep := range parentDeps {
				r.Dependencies[tableName][dep] = struct{}{}
			}
		}
	}
}
