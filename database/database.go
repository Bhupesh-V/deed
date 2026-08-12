package database

import (
	"context"
	"deed/internal/models"
	"deed/internal/stream"
)

type Database interface {
	GetEntities(context.Context) ([]models.Entity, error)
	Ingest(
		ctx context.Context,
		entity string,
		columns []string,
		stream stream.RowStream,
		onProgress func(n int),
	) (int64, error)
	// For a table with ordered column values return the MIN and MAX
	GetBounds(ctx context.Context, tableName string, colName string) (int, int, error)
}
