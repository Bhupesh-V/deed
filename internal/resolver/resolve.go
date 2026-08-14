package resolver

import (
	"deed/internal/models"
	"deed/internal/styles"

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
// Make sure to call GetDependencyTreeUI before calling this
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

// GetDependencyTree builds and returns a lipgloss/tree Tree while populating r.Dependencies.
func (r *Resolver) GetDependencyTreeUI(tableName string, tables map[string]*models.Entity, visited map[string]bool) *tree.Tree {
	if visited == nil {
		visited = make(map[string]bool)
	}

	// Ensure map entry exists for this table
	if _, exists := r.Dependencies[tableName]; !exists {
		r.Dependencies[tableName] = make(map[string]struct{})
	}

	// Create the root node for this sub-tree
	t := tree.New().Root(styles.Node.Render("🔗 " + tableName))

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
		parent, isFK := col.FK()
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
		// Pull all recursive ancestors
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
