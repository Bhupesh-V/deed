package resolver

import (
	"deed/internal/models"
	"fmt"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
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
			parent, _, isFK := col.FK()
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
		parent, _, isFK := col.FK()
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

// GetDependencyTree builds and returns a lipgloss/tree Tree while populating r.Dependencies.
func (r *Resolver) GetDependencyTreeUI(tableName string, tables map[string]*models.Entity, visited map[string]bool) *tree.Tree {
	if visited == nil {
		visited = make(map[string]bool)
	}

	// Ensure map entry exists for this table
	if _, exists := r.Dependencies[tableName]; !exists {
		r.Dependencies[tableName] = make(map[string]struct{})
	}

	// Define node styling
	nodeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3dd3ab")).Bold(false)

	// Create the root node for this sub-tree
	t := tree.New().Root(nodeStyle.Render("🔗 " + tableName))

	// Stop recursion if already visited on this branch to avoid infinite cycles
	if visited[tableName] {
		return t.Child(lipgloss.NewStyle().Faint(true).Render("(circular reference)"))
	}
	visited[tableName] = true

	entity, exists := tables[tableName]
	if !exists {
		return t
	}

	// Collect unique direct parent tables
	seenParents := make(map[string]bool)
	var parents []string

	for _, col := range entity.Columns {
		parent, _, isFK := col.FK()
		if isFK && parent != tableName && !seenParents[parent] {
			seenParents[parent] = true
			parents = append(parents, parent)
			// Record direct dependency
			r.Dependencies[tableName][parent] = struct{}{}
		}
	}

	// Recurse over parents and attach child trees
	for _, parent := range parents {
		// Copy visited map per branch to allow shared dependencies across distinct subtrees
		branchVisited := make(map[string]bool, len(visited))
		for k, v := range visited {
			branchVisited[k] = v
		}

		// Build child subtree recursively
		childTree := r.GetDependencyTreeUI(parent, tables, branchVisited)
		t.Child(childTree)

		// Merge parent's transitive dependencies into current table
		if parentDeps, ok := r.Dependencies[parent]; ok {
			for dep := range parentDeps {
				r.Dependencies[tableName][dep] = struct{}{}
			}
		}
	}

	return t
}

// GetRequiredTables returns a deduplicated list of all target tables and their recursive prerequisites.
func (r *Resolver) GetRequiredTables(lookups []string, allTables map[string]*models.Entity) []string {
	lookupSet := make(map[string]struct{})

	if len(lookups) > 0 {
		// Pull all recursive ancestors using your existing helper
		deps := r.GetDependenciesForTables(lookups)
		for dep := range deps {
			lookupSet[dep] = struct{}{}
		}
		// Include the requested target tables themselves
		for _, l := range lookups {
			lookupSet[l] = struct{}{}
		}
	} else {
		// If no specific lookups provided, target every table in the schema
		for t := range allTables {
			lookupSet[t] = struct{}{}
		}
	}

	required := make([]string, 0, len(lookupSet))
	for table := range lookupSet {
		required = append(required, table)
	}

	return required
}
