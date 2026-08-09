package stream

import (
	"context"
	"deed/internal/models"
	"deed/pkg/calc"
	"deed/pkg/fake"
	"deed/pkg/uuid"
	"encoding/json"
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/brianvoe/gofakeit/v6"
)

// RowStream is an abstract stream of generated mock data rows used by postgres copy from
type RowStream interface {
	// Next advances the stream. Returns false when generation is complete.
	Next() bool
	// Values returns the current row's field values in column order.
	Values() ([]any, error)
	// Err returns any error encountered during row generation.
	Err() error
}

type Stream struct {
	// Schema & Generator State
	targetCols     []models.Column
	rules          map[string]models.GenerationRule
	totalCount     int64
	entity         *models.Entity
	entities       map[string]*models.Entity
	bounds         *sync.Map
	countsPerTable map[string]int64
	uniqueCounter  sync.Map
	faker          *fake.Fake

	// Concurrency & Channel Buffering
	batchChan    chan [][]any
	currentBatch [][]any
	batchIdx     int
	currentRow   []any
	err          atomic.Pointer[error]
}

func New(
	ctx context.Context,
	totalRows int64,
	batchSize int,
	targetCols []models.Column,
	entity *models.Entity,
	entities map[string]*models.Entity,
	bounds *sync.Map,
	rules map[string]models.GenerationRule,
) *Stream {
	workers := runtime.NumCPU()
	batchChan := make(chan [][]any, workers*4)

	st := &Stream{
		targetCols: targetCols,
		totalCount: totalRows,
		entity:     entity,
		entities:   entities,
		bounds:     bounds,
		rules:      rules,
		faker:      fake.New(),
		batchChan:  batchChan,
	}

	go st.startWorkerPool(ctx, totalRows, batchSize, workers)

	return st
}

func (st *Stream) startWorkerPool(ctx context.Context, totalRows int64, batchSize int, workers int) {
	defer close(st.batchChan)

	var wg sync.WaitGroup
	chunkSize := int64(batchSize)
	totalBatches := (totalRows + chunkSize - 1) / chunkSize

	workChan := make(chan int64, totalBatches)
	for b := range totalBatches {
		workChan <- b
	}
	close(workChan)

	for range workers {
		wg.Go(func() {
			for batchIdx := range workChan {
				select {
				case <-ctx.Done():
					return
				default:
				}

				startRow := batchIdx * chunkSize
				endRow := min(startRow+chunkSize, totalRows)
				currentBatchSize := endRow - startRow

				batch := make([][]any, currentBatchSize)
				for i := range currentBatchSize {
					rowIndex := startRow + i + 1
					row := make([]any, len(st.targetCols))
					for cIdx, col := range st.targetCols {
						row[cIdx] = st.generate(col, rowIndex)
					}
					batch[i] = row
				}

				select {
				case st.batchChan <- batch:
				case <-ctx.Done():
					return
				}
			}
		})
	}

	wg.Wait()
}

func (st *Stream) Next() bool {
	if st.currentBatch != nil && st.batchIdx < len(st.currentBatch) {
		st.currentRow = st.currentBatch[st.batchIdx]
		st.batchIdx++
		return true
	}

	batch, ok := <-st.batchChan
	if !ok {
		return false
	}

	st.currentBatch = batch
	st.currentRow = batch[0]
	st.batchIdx = 1
	return true
}

func (st *Stream) Values() ([]any, error) { return st.currentRow, nil }

func (st *Stream) Err() error {
	if errPtr := st.err.Load(); errPtr != nil {
		return *errPtr
	}
	return nil
}

func (st *Stream) generate(col models.Column, rowIndex int64) any {
	// User Rule takes precedence
	if rule, exists := st.rules[col.Name]; exists {
		// TODO: fix for UNIQUE
		if rule.Type == "regex" {
			pattern := rule.RegexPattern
			return gofakeit.Regex(pattern)
		}
	}

	uniqueCounterKey := fmt.Sprintf("%s:%s", st.entity.Name, col.Name)

	if parentTable, ok := col.FK(); ok {
		parent := st.entities[parentTable]

		actual, _ := st.uniqueCounter.LoadOrStore(uniqueCounterKey, new(atomic.Int64))
		counter := actual.(*atomic.Int64).Add(1) - 1

		var bds *models.Bound
		if valBound, ok := st.bounds.Load(parentTable); ok {
			bds = valBound.(*models.Bound)
		}

		lowerId := int64(bds.Lower)
		upperId := int64(bds.Upper)
		parentCount := upperId - lowerId + 1

		if parent.PK().IsOrdered() {
			// Integer/Serial PKs: 1-based bounds [lowerId, upperId]
			return calc.HashCounter(counter+lowerId, lowerId, upperId)

		} else if parent.PK().Type.BaseType == "uuid" {
			// UUID PKs: 1-based bounds [1, parentCount]
			valInt := calc.HashCounter(counter+1, 1, parentCount)
			return uuid.SeqIdToUUID(uint64(valInt))

		} else {
			// String PKs: 0-based bounds [0, parentCount - 1]
			parentIdx := calc.HashCounter(counter, 0, parentCount-1)

			seqIdx, err := st.faker.SeqIdToLetterN(big.NewInt(parentIdx), uint(*parent.PK().Type.Length))
			if err != nil {
				errVal := fmt.Errorf("Failed to decode string for FK %s: %v", col.Name, err)
				if st.err.CompareAndSwap(nil, &errVal) {
					return ""
					// cancel() // Signal context cancellation to stop all worker goroutines immediately
				}

			}
			return seqIdx
		}
	}

	// Fallback to base type defaults
	baseType := strings.ToLower(col.Type.BaseType)

	switch baseType {
	case "int", "int8", "int4":
		return rowIndex + 1

	case "uuid":
		return uuid.SeqIdToUUID(uint64(rowIndex))

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
			val, err := st.faker.LetterN(uniqueCounterKey, uint(*col.Type.Length))
			if err != nil {
				if st.err.CompareAndSwap(nil, &err) {
					// cancel() // Signal context cancellation to stop all worker goroutines immediately
					return ""
				}
			}
			return val
		}
	}

	return ""
}
