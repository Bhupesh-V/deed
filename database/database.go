package database

import (
	"context"
	"deed/internal/models"
	"deed/internal/stream"
)

type Database interface {
	GetEntities(context.Context) ([]models.Entity, error)
	BulkInsert(
		ctx context.Context,
		entity *models.Entity,
		columns []string,
		stream stream.RowStream,
	) (int64, error)
}
