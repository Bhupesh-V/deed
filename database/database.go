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
		entity *models.Entity,
		columns []string,
		stream stream.RowStream,
	) (int64, error)
	// GetRandomIDs returns 'limit' randomly sampled values from the given table and column.
	SampleSavedIDs(ctx context.Context, tableName string, columnName string, limit int) ([]any, error)
}
