package feeder

import (
	"context"
	"deed/database"
	"deed/internal/config"
	"deed/internal/models"
	"deed/internal/stream"
	"errors"
	"fmt"
	"strings"
	"sync"

	"deed/pkg/calc"
	"deed/pkg/fake"
)

type Feeder struct {
	db        database.Database
	config    *config.Config
	input     *models.Input
	entities  map[string]*models.Entity
	batchSize int
}

func New(db database.Database, config *config.Config, input *models.Input, entities map[string]*models.Entity, tablesToIngest []string) (*Feeder, error) {
	batchSize := 500

	s := &Feeder{
		db:        db,
		batchSize: batchSize,
		config:    config,
		input:     input,
		entities:  entities,
	}

	err := s.validateSchema(tablesToIngest)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// validateSchema only checks tables that will actually be seeded this run
// (tablesToIngest), since the rest of the schema won't receive any rows.
func (s *Feeder) validateSchema(tablesToIngest []string) error {
	var errs []error

	for _, table := range tablesToIngest {
		e, ok := s.entities[table]
		if !ok {
			continue
		}

		count := s.GetRowCount(table)
		tr := s.config.TableRule(table)

		for _, col := range e.Columns {
			parent, isFK := col.FK()

			// A UNIQUE FK column can hold at most as many distinct values as
			// its parent table has rows (e.g. a 1:1 profile-to-user
			// relationship). GetRowCount already caps the default case, so
			// this only fires when an explicit config override still
			// exceeds the parent's row count.
			if isFK && col.HasUniqueConstraint() {
				parentCount := s.GetRowCount(parent)
				if count > parentCount {
					errs = append(errs,
						fmt.Errorf(
							"[%s] has a UNIQUE column [%s] referencing [%s] which only has [%d] rows, however [%d] rows were requested",
							e.Name,
							col.Name,
							parent,
							parentCount,
							count,
						),
					)
				}
				continue
			}

			// Plain FK columns are resolved against the parent table's row
			// range at generation time (see stream.generate), not generated
			// from their own type's cardinality, so they're exempt from the
			// checks below.
			if isFK {
				continue
			}

			// Only UNIQUE columns can actually run out of distinct values;
			// duplicates are harmless everywhere else.
			if !col.HasUniqueConstraint() {
				continue
			}

			var datasetsize int64
			var sizeDesc string

			if colRule, ok := tr.Columns[col.Name]; ok && colRule.Pattern != "" {
				// A configured pattern overrides the type-based generator
				// (see stream.generate), so its own reachable value space —
				// not the column's declared length — is what actually
				// governs collision risk here.
				size, err := fake.EstimateRegexCapacity(colRule.Pattern)
				if err != nil {
					errs = append(errs, fmt.Errorf("[%s] column [%s] has an invalid pattern %q: %w", e.Name, col.Name, colRule.Pattern, err))
					continue
				}
				datasetsize = size
				sizeDesc = fmt.Sprintf("pattern %q", colRule.Pattern)
			} else {
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
				sizeDesc = fmt.Sprintf("type [%s]", col.Type.BaseType)
			}

			if count > datasetsize {
				errs = append(errs,
					fmt.Errorf(
						"[%s] has a UNIQUE column [%s] whose %s limits possible values to [%d], however [%d] rows were requested",
						e.Name,
						col.Name,
						sizeDesc,
						datasetsize,
						count,
					),
				)
			}
		}
	}

	return errors.Join(errs...)
}

// useDBDefault reports whether a not-null boolean column with a schema
// default should be left out of the insert entirely, letting the database
// apply its own default instead of deed generating a value for it. This only
// kicks in when the user hasn't configured any true/false split for the
// column — otherwise their config takes precedence.
func useDBDefault(col models.Column, tr *config.TableRule) bool {
	if col.Default == nil || col.Nullable {
		return false
	}

	if strings.ToLower(col.Type.BaseType) != "bool" {
		return false
	}

	rule, ok := tr.Columns[col.Name]
	if !ok {
		return true
	}

	return rule.TruePercentage == 0 && rule.FalsePercentage == 0
}

func (f *Feeder) Prepare(
	ctx context.Context,
	table string,
	entities map[string]*models.Entity,
	bounds *sync.Map,
	config *config.Config,
) ([]string, stream.RowStream, error) {

	var targetCols []models.Column
	var colNames []string

	entity := entities[table]
	if entity == nil {
		return nil, nil, fmt.Errorf("unable to find entity")
	}

	totalRows := f.GetRowCount(table)
	tr := f.config.TableRule(entity.Name)

	for _, col := range entity.Columns {
		if col.IsAutoIncrement() {
			continue
		}
		if useDBDefault(col, tr) {
			continue
		}
		targetCols = append(targetCols, col)
		colNames = append(colNames, col.Name)
	}

	const channelBatchSize = 5000

	st := stream.New(
		ctx,
		totalRows,
		channelBatchSize,
		targetCols,
		entity,
		entities,
		bounds,
		config,
	)

	return colNames, st, nil
}

// GetRowCount resolves the target row count for a table. Explicit config
// overrides win outright; otherwise it falls back to the global --count,
// capped by any UNIQUE FK parent's row count so a 1:1-related table (e.g. a
// profile table keyed 1:1 to users) can't default to more rows than its
// parent actually has.
func (f *Feeder) GetRowCount(table string) int64 {
	entity, ok := f.entities[table]
	if !ok {
		return f.input.Count
	}
	if tableCfg, ok := f.config.Rules.Tables[entity.Name]; ok && tableCfg.Count > 0 {
		return tableCfg.Count
	}

	count := f.input.Count

	for _, col := range entity.Columns {
		parent, isFK := col.FK()
		if !isFK || parent == table || !col.HasUniqueConstraint() {
			continue
		}
		if parentCount := f.GetRowCount(parent); parentCount < count {
			count = parentCount
		}
	}

	return count
}

// CountCap describes a table whose row count was silently reduced by the
// UNIQUE FK capping in GetRowCount, so callers can warn the user up front.
type CountCap struct {
	Table       string
	ParentTable string
	Requested   int64
	Capped      int64
}

// AutoCappedCounts reports every table in tablesToIngest whose default row
// count (no explicit config override) got reduced below the global --count
// by a UNIQUE FK parent's smaller row count.
func (f *Feeder) AutoCappedCounts(tablesToIngest []string) []CountCap {
	var caps []CountCap

	for _, table := range tablesToIngest {
		entity, ok := f.entities[table]
		if !ok {
			continue
		}

		// An explicit override always wins in GetRowCount, so it can't be capped.
		if tableCfg, ok := f.config.Rules.Tables[entity.Name]; ok && tableCfg.Count > 0 {
			continue
		}

		requested := f.input.Count
		actual := f.GetRowCount(table)
		if actual >= requested {
			continue
		}

		var parent string
		for _, col := range entity.Columns {
			p, isFK := col.FK()
			if !isFK || p == table || !col.HasUniqueConstraint() {
				continue
			}
			if f.GetRowCount(p) == actual {
				parent = p
				break
			}
		}

		caps = append(caps, CountCap{
			Table:       table,
			ParentTable: parent,
			Requested:   requested,
			Capped:      actual,
		})
	}

	return caps
}
