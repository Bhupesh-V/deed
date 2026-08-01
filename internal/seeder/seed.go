package seeder

import (
	"deed/internal/models"
	"deed/internal/stream"
	"fmt"
	"math/rand"
	"strings"

	"github.com/brianvoe/gofakeit/v6"
)

type Seeder struct {
	// In-memory foreign key registry mapping table -> generated primary keys
	idRegistry map[string][]int
}

func New() *Seeder {
	return &Seeder{
		idRegistry: make(map[string][]int),
	}
}

// TableStream implements databulk.RowStream
type TableStream struct {
	seeder       *Seeder
	targetCols   []models.Column
	rules        map[string]models.GenerationRule
	totalCount   int
	currentIndex int
	currentRow   []any
	err          error
}

// CreateStream generates column names and returns a databulk.RowStream
func (s *Seeder) CreateStream(
	entity *models.Entity,
	count int,
	rules map[string]models.GenerationRule,
) ([]string, stream.RowStream) {
	var targetCols []models.Column
	var colNames []string

	for _, col := range entity.Columns {
		if col.IsAutoIncrement() {
			continue
		}
		targetCols = append(targetCols, col)
		colNames = append(colNames, col.Name)
	}

	stream := &TableStream{
		seeder:       s,
		targetCols:   targetCols,
		rules:        rules,
		totalCount:   count,
		currentIndex: 0,
	}

	return colNames, stream
}

func (ts *TableStream) Next() bool {
	if ts.currentIndex >= ts.totalCount {
		return false
	}
	ts.currentIndex++

	row := make([]any, len(ts.targetCols))
	for i, col := range ts.targetCols {
		row[i] = ts.generateValue(col, ts.currentIndex)
	}

	ts.currentRow = row
	return true
}

func (ts *TableStream) Values() ([]any, error) { return ts.currentRow, nil }
func (ts *TableStream) Err() error             { return ts.err }

func (ts *TableStream) generateValue(col models.Column, rowIndex int) any {
	// Custom User Rule
	if rule, exists := ts.rules[col.Name]; exists {
		if rule.Type == "regex" {
			pattern := rule.RegexPattern
			return gofakeit.Regex(pattern)
		}
	}

	// Foreign Key Lookup from Seeder Memory Cache
	if parentTable, _, ok := col.GetFK(); ok {
		if parentIDs, ok := ts.seeder.idRegistry[parentTable]; ok && len(parentIDs) > 0 {
			return parentIDs[rand.Intn(len(parentIDs))]
		}
	}

	// Fallback to base type defaults
	baseType := strings.ToLower(col.Type.BaseType)
	switch {
	case strings.Contains(baseType, "int"):
		return rowIndex + 1

	case strings.Contains(baseType, "uuid"):
		return gofakeit.UUID()

	case strings.Contains(baseType, "bool"):
		return rowIndex%2 == 0

	case strings.Contains(baseType, "timestamp"), strings.Contains(baseType, "date"):
		return gofakeit.Date()

	case strings.Contains(baseType, "numeric"), strings.Contains(baseType, "decimal"), strings.Contains(baseType, "float"), strings.Contains(baseType, "real"):
		return gofakeit.Float64Range(1.00, 500.00)

	// case strings.Contains(baseType, "json"):
	// 	return `{"generated": true}`

	case strings.Contains(baseType, "inet"):
		return gofakeit.IPv4Address()

	case strings.Contains(baseType, "bpchar"), strings.Contains(baseType, "char"):
		// Handle CHAR(n) / bpchar fixed lengths using Precision
		if col.Type.Precision != nil && *col.Type.Precision > 0 && *col.Type.Precision <= 3 {
			return gofakeit.LetterN(uint(*col.Type.Precision))
		}
		return fmt.Sprintf("%s_%d", col.Name, rowIndex)

	default:
		// Default string generator with precision clipping if VARCHAR(n) limit exists
		val := fmt.Sprintf("%s_%d", col.Name, rowIndex)
		if col.Type.Precision != nil && *col.Type.Precision > 0 && len(val) > *col.Type.Precision {
			return val[:*col.Type.Precision]
		}
		return val
	}
}
