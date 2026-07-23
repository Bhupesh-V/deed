package database

import (
	"context"
	"deed/internal/models"
)

type Database interface {
	GetEntities(context.Context) ([]models.Entity, error)
	BulkInsert(ctx context.Context)
}
