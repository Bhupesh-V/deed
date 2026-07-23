package postgres

import (
	"context"
	"deed/database"
	"deed/internal/models"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgres struct {
	// connection details
	pg *pgxpool.Pool
}

func New(con *pgxpool.Pool) database.Database {
	repo := &postgres{pg: con}
	return repo
}

func (p *postgres) GetEntities(ctx context.Context) ([]models.Entity, error) {
	query := `
	SELECT
		c.table_name,
		c.column_name,
		c.column_default,
		c.data_type,
		c.udt_name,
		c.is_nullable,
		c.character_maximum_length,
		c.numeric_precision,
		c.numeric_precision_radix,
		c.datetime_precision,
		c.is_self_referencing,
		c.dtd_identifier,
		c.generation_expression,
		c.maximum_cardinality,
		t.hasindexes,
		t.hastriggers
	FROM
		information_schema.columns AS c
	LEFT JOIN 
		pg_catalog.pg_tables AS t 
		ON c.table_name = t.tablename 
		AND c.table_schema = t.schemaname
	WHERE
		c.table_schema = $1
	ORDER BY
		c.table_schema,
		c.table_name,
		c.ordinal_position;
	`

	// Execute the query
	rows, err := p.pg.Query(ctx, query, "public")
	if err != nil {
		log.Fatalf("Query failed: %v\n", err)
	}

	flatRows, err := pgx.CollectRows(rows, pgx.RowToStructByPos[FlatRow])
	if err != nil {
		log.Fatalf("Failed to collect rows: %v\n", err)
		return nil, err
	}

	tableMap := make(map[string]*Table)
	var orderedTableNames []string

	for _, r := range flatRows {
		// If the table hasn't been initialized in our map yet, create it
		if _, exists := tableMap[r.TableName]; !exists {
			tableMap[r.TableName] = &Table{
				// Schema:      r.TableSchema,
				Name:        r.TableName,
				HasIndexes:  r.HasIndexes,
				HasTriggers: r.HasTriggers,
				Columns:     []Column{},
			}
			orderedTableNames = append(orderedTableNames, r.TableName)
		}

		// Append the current column metadata to the table's column slice
		tableMap[r.TableName].Columns = append(tableMap[r.TableName].Columns, Column{
			Name:                   r.ColumnName,
			Default:                r.ColumnDefault,
			DataType:               r.DataType,
			UdtName:                r.UdtName,
			IsNullable:             r.IsNullable,
			CharacterMaximumLength: r.CharacterMaximumLength,
			NumericPrecision:       r.NumericPrecision,
			NumericPrecisionRadix:  r.NumericPrecisionRadix,
			DatetimePrecision:      r.DatetimePrecision,
			IsSelfReferencing:      r.IsSelfReferencing,
			DtdIdentifier:          r.DtdIdentifier,
			GenerationExpression:   r.GenerationExpression,
			MaximumCardinality:     r.MaximumCardinality,
		})
	}

	// 3. Compile everything back into a cleanly ordered slice of Table structs
	tables := make([]Table, 0, len(tableMap))
	for _, name := range orderedTableNames {
		tables = append(tables, *tableMap[name])
	}
	entities := []models.Entity{}

	for _, t := range tables {
		entities = append(entities, models.Entity{
			Name: t.Name,
		})
	}
	return entities, nil
}

func (p *postgres) BulkInsert(ctx context.Context) {
	panic("not implemented") // TODO: Implement
}
