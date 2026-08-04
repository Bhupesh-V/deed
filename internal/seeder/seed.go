package seeder

import (
	"deed/database"
	"deed/internal/models"
	"deed/internal/stream"
	"errors"
	"fmt"
	"strings"

	"deed/pkg/calc"
	"deed/pkg/fake"

	"github.com/brianvoe/gofakeit/v6"
)

type Seeder struct {
	db        database.Database
	batchSize int
}

func New(db database.Database) *Seeder {
	batchSize := 500

	return &Seeder{
		db:        db,
		batchSize: batchSize,
	}
}

// TableStream implements stream.RowStream
type TableStream struct {
	seeder       *Seeder
	targetCols   []models.Column
	rules        map[string]models.GenerationRule
	totalCount   int
	currentIndex int
	currentRow   []any
	fetcher      *fetcher
	faker        *fake.Fake
	err          error
}

// Prepare generates column names and returns a stream.RowStream
func (s *Seeder) Prepare(
	entity *models.Entity,
	count int,
	rules map[string]models.GenerationRule,
) ([]string, stream.RowStream, error) {
	var targetCols []models.Column
	var colNames []string

	err := validateCounts(entity, count)
	if err != nil {
		return nil, nil, err
	}

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
		fetcher:      newFetcher(s.db, s.batchSize),
		faker:        fake.New(),
	}

	return colNames, stream, nil
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
	// User Rule takes precedence
	if rule, exists := ts.rules[col.Name]; exists {
		if rule.Type == "regex" {
			pattern := rule.RegexPattern
			return gofakeit.Regex(pattern)
		}
	}

	if parentTable, _, ok := col.GetFK(); ok {
		val, err := ts.fetcher.GetParentId(parentTable, col.Name)
		if err != nil {
			ts.err = fmt.Errorf("failed to generate foreign key for %s: %w", col.Name, err)
		}
		return val
		// find type of relationship: 1-1, 1-M or M-N
		// 1-1: invoke calc.Permute
		// 1-M or M-N invoke fetcher.GetParentId
	}

	// Fallback to base type defaults
	baseType := strings.ToLower(col.Type.BaseType)
	switch {
	case strings.Contains(baseType, "int"):
		return rowIndex + 1

	case strings.Contains(baseType, "uuid"):
		return gofakeit.UUID()

	case strings.Contains(baseType, "bool"):
		// TODO: figure out bool Percentage
		return rowIndex%2 == 0

	case strings.Contains(baseType, "timestamp"), strings.Contains(baseType, "timestampz"), strings.Contains(baseType, "date"):
		// TODO: incremental dates
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
		} else if col.Type.Length != nil {
			val, err := ts.faker.LetterN(col.Name, uint(*col.Type.Length))
			if err != nil {
				ts.err = err
				return ""
			}
			return val
		}

	case strings.Contains(baseType, "varchar"):
		if col.Type.Length != nil {
			return gofakeit.LetterN(uint(*col.Type.Length))
		}

	default:
		// Default string generator with precision clipping if VARCHAR(n) limit exists
		val := fmt.Sprintf("%s_%d", col.Name, rowIndex)
		if col.Type.Precision != nil && *col.Type.Precision > 0 && len(val) > *col.Type.Precision {
			return val[:*col.Type.Precision]
		}
		return val
	}

	return ""
}

func validateCounts(entity *models.Entity, count int) error {
	var errs []error

	for _, col := range entity.Columns {
		if col.HasUniqueConstraint() && !col.IsAutoIncrement() {
			var datasetsize int64

			baseType := strings.ToLower(col.Type.BaseType)

			switch {
			case strings.Contains(baseType, "numeric"), strings.Contains(baseType, "decimal"), strings.Contains(baseType, "int"), strings.Contains(baseType, "int4"):
				if col.Type.Precision != nil && *col.Type.Precision > 0 {
					datasetsize = calc.GetNumericDatasetSize(int64(*col.Type.Precision), int64(*col.Type.Scale), int64(*col.Type.Radix))
				}

			case strings.Contains(baseType, "bpchar"), strings.Contains(baseType, "char"), strings.Contains(baseType, "varchar"):
				// Handle CHAR(n) / bpchar fixed lengths using Precision
				if col.Type.Length != nil {
					datasetsize = calc.TotalCombinations(52, float64(*col.Type.Length))
				}

			}

			if int64(count) > datasetsize {
				errs = append(errs,
					fmt.Errorf(
						"[%s] has a column [%s] with a UNIQUE constraint which limits possible values of type [%s] to [%d], however [%d] rows were requested",
						entity.Name,
						col.Name,
						col.Type.BaseType,
						datasetsize,
						count,
					),
				)
			}
		}
	}

	return errors.Join(errs...)
}
