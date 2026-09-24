package resolver

import (
	"deed/internal/models"
	"deed/internal/styles"
	"fmt"
	"maps"
	"strings"

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

// parentRef describes a direct FK edge from a table to one of its parents.
type parentRef struct {
	table       string
	cardinality string // "1:1" when the FK column is unique, "M:1" otherwise
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

	t := tree.New().Root(styles.Node.Render(tableName))

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
	var parents []parentRef

	for _, col := range entity.Columns {
		parent, isFK := col.FK()
		if isFK && parent != tableName && !seenParents[parent] {
			seenParents[parent] = true
			cardinality := "M:1"
			if col.HasUniqueConstraint() {
				cardinality = "1:1"
			}
			parents = append(parents, parentRef{table: parent, cardinality: cardinality})
			// Record direct dependency
			r.Dependencies[tableName][parent] = struct{}{}
		}
	}

	// Recurse over parents, attach child trees, and draw each branch with its cardinality
	// baked into the connector itself, e.g. "├─(1:1)─▶ parent_table".
	for _, parent := range parents {
		// Copy visited map per branch to allow shared dependencies across distinct subtrees
		branchVisited := make(map[string]bool, len(visited))
		maps.Copy(branchVisited, visited)

		childTree := r.GetDependencyTreeUI(parent.table, tables, branchVisited)
		t.Child(childTree)

		// Merge parent's transitive dependencies into current table
		if parentDeps, ok := r.Dependencies[parent.table]; ok {
			for dep := range parentDeps {
				r.Dependencies[tableName][dep] = struct{}{}
			}
		}
	}

	t.Enumerator(cardinalityEnumerator(parents))
	t.Indenter(cardinalityIndenter(parents))

	return t
}

// cardinalityEnumerator draws each branch as "├─(cardinality)─▶" / "╰─(cardinality)─▶",
// baking the relationship type directly into the connector.
func cardinalityEnumerator(parents []parentRef) tree.Enumerator {
	return func(children tree.Children, i int) string {
		corner := "├─"
		if children.Length()-1 == i {
			corner = "╰─"
		}
		return fmt.Sprintf("%s(%s)─▶", corner, parents[i].cardinality)
	}
}

// cardinalityIndenter continues the vertical bar under a branch at the same width
// as cardinalityEnumerator produces, so nested subtrees line up under the first
// letter of the table name. Width is measured with lipgloss.Width (display columns,
// not bytes) since the connector is built from multi-byte box-drawing runes.
func cardinalityIndenter(parents []parentRef) tree.Indenter {
	enum := cardinalityEnumerator(parents)
	return func(children tree.Children, i int) string {
		width := lipgloss.Width(enum(children, i))
		if children.Length()-1 == i {
			return strings.Repeat(" ", width)
		}
		return "│" + strings.Repeat(" ", width-1)
	}
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
