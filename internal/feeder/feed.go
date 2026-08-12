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

func New(db database.Database, config *config.Config, input *models.Input, entities map[string]*models.Entity) (*Feeder, error) {
	batchSize := 500

	s := &Feeder{
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

func (s *Feeder) validateSchema() error {
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

func (f *Feeder) Prepare(
	ctx context.Context,
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

	if tableCfg, ok := f.config.Rules.Rules.Tables[entity.Name]; ok {
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

	const channelBatchSize = 5000

	st := stream.New(
		ctx,
		totalRows,
		channelBatchSize,
		targetCols,
		entity,
		entities,
		bounds,
		rules,
	)

	return colNames, st, nil
}

func (f *Feeder) GetTableCount(table string) int64 {
	entity, ok := f.entities[table]
	if !ok {
		return f.input.Count
	}
	if tableCfg, ok := f.config.Rules.Rules.Tables[entity.Name]; ok && tableCfg.Count > 0 {
		return tableCfg.Count
	}
	return f.input.Count
}
