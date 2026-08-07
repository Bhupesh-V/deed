package seeder

import (
	"deed/database"
	"deed/internal/config"
	"deed/internal/models"
	"deed/internal/stream"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"deed/pkg/calc"
	"deed/pkg/fake"

	"github.com/brianvoe/gofakeit/v6"
)

type Seeder struct {
	db        database.Database
	config    *config.Config
	input     *models.Input
	entities  map[string]*models.Entity
	batchSize int
}

func New(db database.Database, config *config.Config, input *models.Input, entities map[string]*models.Entity) (*Seeder, error) {
	batchSize := 500

	s := &Seeder{
		db:        db,
		batchSize: batchSize,
		config:    config,
		input:     input,
		entities:  entities,
	}

	err := s.validateSchema()
	if err != nil {
		return nil, err
	}

	return s, nil
}

// TableStream implements stream.RowStream
type TableStream struct {
	seeder         *Seeder
	targetCols     []models.Column
	rules          map[string]models.GenerationRule
	totalCount     int64
	currentIndex   int64
	currentRow     []any
	fetcher        *fetcher
	faker          *fake.Fake
	uniqueCounter  sync.Map
	countsPerTable map[string]int64
	// entity for which we are streaming rows right now
	entity   *models.Entity
	entities map[string]*models.Entity
	bounds   *sync.Map
	err      error
}

// Prepare generates column names and returns a RowStream
func (s *Seeder) Prepare(
	table string,
	count int64,
	entities map[string]*models.Entity,
	bounds *sync.Map,
) ([]string, stream.RowStream, error) {

	var targetCols []models.Column
	var colNames []string
	var totalRows int64 = count

	rules := make(map[string]models.GenerationRule)
	countsPerTable := make(map[string]int64)

	entity := entities[table]

	if tableCfg, ok := s.config.Rules.Rules.Tables[entity.Name]; ok {
		if tableCfg.Count > 0 {
			totalRows = tableCfg.Count
		}

		countsPerTable[entity.Name] = int64(totalRows)

		for colName, colRule := range tableCfg.Columns {
			rules[colName] = models.GenerationRule{
				Type:         colRule.Type,
				RegexPattern: colRule.Pattern,
			}
		}
	}

	for _, col := range entity.Columns {
		if col.IsAutoIncrement() {
			continue
		}
		targetCols = append(targetCols, col)
		colNames = append(colNames, col.Name)
	}

	stream := &TableStream{
		seeder:         s,
		targetCols:     targetCols,
		rules:          rules,
		totalCount:     totalRows,
		currentIndex:   0,
		fetcher:        newFetcher(s.db, s.batchSize),
		faker:          fake.New(),
		countsPerTable: countsPerTable,
		entities:       entities,
		entity:         entity,
		bounds:         bounds,
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

func (ts *TableStream) generateValue(col models.Column, rowIndex int64) any {
	// User Rule takes precedence
	if rule, exists := ts.rules[col.Name]; exists {
		// TODO: fix for UNIQUE
		if rule.Type == "regex" {
			pattern := rule.RegexPattern
			return gofakeit.Regex(pattern)
		}
	}

	isColUnique := col.HasUniqueConstraint()
	uniqueCounterKey := fmt.Sprintf("%s:%s", ts.entity.Name, col.Name)

	if parentTable, ok := col.FK(); ok {
		var val any
		var err error

		parent := ts.entities[parentTable]

		// 1-1 mapping with FK column
		if isColUnique && parent.PK().IsOrdered() {
			actual, _ := ts.uniqueCounter.LoadOrStore(uniqueCounterKey, new(atomic.Int64))
			counter := actual.(*atomic.Int64).Add(1) - 1

			var bds *models.Bound
			// the parent table will always be processed before child so no way the bounds won't exist
			if val, ok := ts.bounds.Load(parentTable); ok {
				bds = val.(*models.Bound)
			}

			lowerId := int64(bds.Lower)
			upperId := int64(bds.Upper)

			val = calc.HashCounter(counter, lowerId, upperId)
		} else {
			// 1-M or M-N mapping with FK column
			val, err = ts.fetcher.GetParentId(parentTable, col.Name)
			if err != nil {
				ts.err = fmt.Errorf("failed to generate foreign key for %s: %w", col.Name, err)
			}
		}
		return val
	}

	// Fallback to base type defaults
	baseType := strings.ToLower(col.Type.BaseType)

	switch baseType {
	case "int", "int8", "int4":
		return rowIndex + 1

	case "uuid":
		return gofakeit.UUID()

	case "bool":
		// TODO: figure out bool Percentage
		return rowIndex%2 == 0

	case "timestamp", "timestamptz", "date":
		// TODO: incremental dates
		return gofakeit.Date()

	case "numeric", "decimal", "float", "real":
		return gofakeit.Float64Range(1.00, 500.00)

	case "jsonb":
		return json.RawMessage(`{"generated": true}`)

	case "inet":
		return gofakeit.IPv4Address()

	case "bpchar", "char", "varchar":
		if col.Type.Length != nil {
			val, err := ts.faker.LetterN(uniqueCounterKey, uint(*col.Type.Length))
			if err != nil {
				ts.err = err
				return ""
			}
			return val
		}
	}

	return ""
}

func (s *Seeder) validateSchema() error {
	var errs []error

	for _, e := range s.entities {
		var count int64

		if tableCfg, ok := s.config.Rules.Rules.Tables[e.Name]; ok {
			if tableCfg.Count > 0 {
				count = tableCfg.Count
			} else {
				count = s.input.Count
			}
		}

		for _, col := range e.Columns {
			// fmt.Println("came here for col", col.Name)
			var datasetsize int64

			baseType := strings.ToLower(col.Type.BaseType)

			switch baseType {
			case "numeric", "decimal", "int", "int4", "int8":
				if col.Type.Precision != nil && *col.Type.Precision > 0 {
					datasetsize = calc.GetNumericDatasetSize(int64(*col.Type.Precision), int64(*col.Type.Scale), int64(*col.Type.Radix))
				}

			case "bpchar", "char", "varchar":
				if col.Type.Length != nil {
					datasetsize = calc.TotalCombinations(52, float64(*col.Type.Length))
				}

			default:
				continue
			}

			// TODO: add special guard for UNIQUE
			if count > datasetsize {
				errs = append(errs,
					fmt.Errorf(
						"[%s] has a column [%s] of type [%s] which limits possible values to [%d], however [%d] rows were requested",
						e.Name,
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
