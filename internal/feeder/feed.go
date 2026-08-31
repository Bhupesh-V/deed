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

		for _, col := range e.Columns {
			// FK columns are resolved against the parent table's row range at
			// generation time (see stream.generate), not generated from their
			// own type's cardinality, so they're exempt from this check.
			if _, isFK := col.FK(); isFK {
				continue
			}

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
	count int64,
	entities map[string]*models.Entity,
	bounds *sync.Map,
	config *config.Config,
) ([]string, stream.RowStream, error) {

	var targetCols []models.Column
	var colNames []string
	var totalRows int64 = count

	entity := entities[table]
	if entity == nil {
		return nil, nil, fmt.Errorf("unable to find entity")
	}

	tr := f.config.TableRule(entity.Name)
	if tr.Count > 0 {
		totalRows = tr.Count
	}

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

func (f *Feeder) GetRowCount(table string) int64 {
	entity, ok := f.entities[table]
	if !ok {
		return f.input.Count
	}
	if tableCfg, ok := f.config.Rules.Tables[entity.Name]; ok && tableCfg.Count > 0 {
		return tableCfg.Count
	}
	return f.input.Count
}
