package seeder

import (
	"context"
	"deed/database"
	"fmt"
)

type fkFetcher struct {
	db        database.Database
	batchSize int
	buffers   map[string][]any
	cursors   map[string]int
}

func newFKFetcher(db database.Database, batchSize int) *fkFetcher {
	return &fkFetcher{
		db:        db,
		batchSize: batchSize,
		buffers:   make(map[string][]any),
		cursors:   make(map[string]int),
	}
}

func (f *fkFetcher) GetNextID(table, column string) (any, error) {
	key := fmt.Sprintf("%s.%s", table, column)

	cursor := f.cursors[key]
	buffer := f.buffers[key]

	if cursor >= len(buffer) {
		freshBatch, err := f.db.GetRandomIDs(context.Background(), table, column, f.batchSize)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch foreign key batch for %s: %w", key, err)
		}

		if len(freshBatch) == 0 {
			return nil, fmt.Errorf("parent table %s has no rows to populate foreign key %s", table, column)
		}

		f.buffers[key] = freshBatch
		f.cursors[key] = 0
		buffer = freshBatch
		cursor = 0
	}

	val := buffer[cursor]
	f.cursors[key] = cursor + 1

	return val, nil
}
