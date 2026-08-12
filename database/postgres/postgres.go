package postgres

import (
	"context"
	"deed/database"
	"deed/internal/models"
	"deed/internal/stream"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgres struct {
	// connection details
	pg *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (database.Database, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pgx config: %w", err)
	}

	config.MaxConns = 20
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute
	config.ConnConfig.ConnectTimeout = 5 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	repo := &postgres{pg: pool}
	return repo, nil
}

func (p *postgres) GetEntities(ctx context.Context) ([]models.Entity, error) {
	query := `
	SELECT
		c.table_name,
		c.column_name,
		COALESCE(cons.is_primary_key, false) AS is_primary_key,
		c.column_default,
		c.data_type,
		c.udt_name,
		COALESCE(enums.enum_values, ARRAY[]::TEXT[]) AS enum_values,
		c.is_nullable,
		c.is_identity,
		c.identity_generation,
		c.character_maximum_length,
		c.numeric_precision,
		c.numeric_precision_radix,
		c.numeric_scale,
		c.datetime_precision,
		c.is_self_referencing,
		c.dtd_identifier,
		c.generation_expression,
		c.maximum_cardinality,
		COALESCE(t.hasindexes, FALSE) AS hasindexes,
		COALESCE(t.hastriggers, FALSE) AS hastriggers,
		COALESCE(cons.constraints, '[]'::jsonb) AS column_constraints
	FROM
		information_schema.columns AS c
		LEFT JOIN pg_catalog.pg_tables AS t ON c.table_name = t.tablename
		AND c.table_schema = t.schemaname
		LEFT JOIN (
			SELECT
				t.typname,
				n.nspname AS schema_name,
				array_agg(
					e.enumlabel
					ORDER BY
						e.enumsortorder
				) AS enum_values
			FROM
				pg_catalog.pg_type t
				JOIN pg_catalog.pg_enum e ON t.oid = e.enumtypid
				JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
			GROUP BY
				t.typname,
				n.nspname
		) AS enums ON c.udt_schema = enums.schema_name
		AND c.udt_name = enums.typname
		LEFT JOIN (
			SELECT
				tc.table_schema,
				tc.table_name,
				ccu.column_name,
				bool_or(tc.constraint_type = 'PRIMARY KEY') AS is_primary_key,
				jsonb_agg(
					jsonb_build_object(
						'name',
						tc.constraint_name,
						'type',
						tc.constraint_type,
						'check_clause',
						ch.check_clause,
						'referenced_table',
						ref_ccu.table_name,
						'referenced_column',
						ref_ccu.column_name
					)
				) AS constraints
			FROM
				information_schema.table_constraints AS tc
				JOIN information_schema.constraint_column_usage AS ccu ON tc.constraint_name = ccu.constraint_name
				AND tc.table_schema = ccu.constraint_schema
				LEFT JOIN information_schema.check_constraints AS ch ON tc.constraint_name = ch.constraint_name
				AND tc.table_schema = ch.constraint_schema
				LEFT JOIN information_schema.referential_constraints AS rc ON tc.constraint_name = rc.constraint_name
				AND tc.table_schema = rc.constraint_schema
				LEFT JOIN information_schema.constraint_column_usage AS ref_ccu ON rc.unique_constraint_name = ref_ccu.constraint_name
				AND rc.unique_constraint_schema = ref_ccu.table_schema
			WHERE
				(
					ch.check_clause IS NULL
					OR ch.check_clause NOT ILIKE '%IS NOT NULL'
				)
			GROUP BY
				tc.table_schema,
				tc.table_name,
				ccu.column_name
		) AS cons ON c.table_schema = cons.table_schema
		AND c.table_name = cons.table_name
		AND c.column_name = cons.column_name
	WHERE
		c.table_schema = $1
	ORDER BY
		c.table_schema,
		c.table_name,
		c.ordinal_position;
	`

	rows, err := p.pg.Query(ctx, query, "public")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	flatRows, err := pgx.CollectRows(rows, pgx.RowToStructByPos[FlatRow])
	if err != nil {
		log.Fatalf("Failed to collect rows: %v\n", err)
		return nil, err
	}

	tableMap := make(map[string]*Table)
	var orderedTableNames []string

	for _, r := range flatRows {
		// Parse out JSON constraints for this column
		var constraints []Constraint
		if len(r.ColumnConstraints) > 0 {
			if err := json.Unmarshal(r.ColumnConstraints, &constraints); err != nil {
				log.Fatalf("Failed to parse constraint JSON for column %s: %v\n", r.ColumnName, err)
				return nil, err
			}
		}

		// If the table hasn't been initialized in our map yet, create it
		if _, exists := tableMap[r.TableName]; !exists {
			tableMap[r.TableName] = &Table{
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
			IsPrimaryKey:           r.IsPrimaryKey,
			DataType:               r.DataType,
			UdtName:                r.UdtName,
			EnumValues:             r.EnumValues,
			IsNullable:             r.IsNullable,
			IsIdentity:             r.IsIdentity,
			IdentityGeneration:     r.IdentityGeneration,
			CharacterMaximumLength: r.CharacterMaximumLength,
			NumericPrecision:       r.NumericPrecision,
			NumericPrecisionRadix:  r.NumericPrecisionRadix,
			NumericPrecisionScale:  r.NumericPrecisionScale,
			DatetimePrecision:      r.DatetimePrecision,
			IsSelfReferencing:      r.IsSelfReferencing,
			DtdIdentifier:          r.DtdIdentifier,
			GenerationExpression:   r.GenerationExpression,
			MaximumCardinality:     r.MaximumCardinality,
			Constraints:            constraints,
		})
	}

	// Compile everything back into a cleanly ordered slice of Table structs
	tables := make([]Table, 0, len(tableMap))
	for _, name := range orderedTableNames {
		tables = append(tables, *tableMap[name])
	}

	entities := []models.Entity{}
	for _, t := range tables {
		cols := []models.Column{}

		for _, c := range t.Columns {
			constraints := []models.Constraint{}

			for _, ctr := range c.Constraints {
				constraints = append(constraints, models.Constraint{
					Name:             ctr.Name,
					Type:             ctr.Type,
					ReferencedTable:  ctr.ReferencedTable,
					ReferencedColumn: ctr.ReferencedColumn,
				})
			}

			var nullable bool
			if c.IsNullable == "YES" {
				nullable = true
			}
			var hasIdentity bool
			if c.IsIdentity == "YES" && c.IdentityGeneration != nil {
				hasIdentity = true
			}

			cols = append(cols, models.Column{
				Name: c.Name,
				Type: models.DataType{
					BaseType:  c.UdtName,
					DBType:    c.DataType,
					Length:    c.CharacterMaximumLength,
					Precision: c.NumericPrecision,
					Scale:     c.NumericPrecisionScale,
					Radix:     c.NumericPrecisionRadix,
				},
				Constraint:   constraints,
				Nullable:     nullable,
				IsPrimaryKey: c.IsPrimaryKey,
				Default:      c.Default,
				HasIdentity:  hasIdentity,
				EnumValues:   c.EnumValues,
			})
		}

		entities = append(entities, models.Entity{
			Name:    t.Name,
			Columns: cols,
			// Map t.Columns, t.HasIndexes, or t.HasTriggers here if models.Entity has fields for them
		})
	}
	return entities, nil
}

// A 50% faster query to get entities and their metadata
func (p *postgres) getentitiesqueryV2() string {
	query := `
	SELECT
		t.relname AS table_name,
		a.attname AS column_name,
		pg_get_expr(d.adbin, d.adrelid) AS column_default,
		format_type(a.atttypid, a.atttypmod) AS data_type,
		col_type.typname AS udt_name,
		CASE WHEN a.attnotnull THEN 'NO' ELSE 'YES' END AS is_nullable,
		CASE 
			WHEN a.atttypmod > 4 THEN a.atttypmod - 4 
			ELSE NULL 
		END AS character_maximum_length,
		CASE 
			WHEN a.atttypid IN (1700) THEN ((a.atttypmod - 4) >> 16) & 65535 
			ELSE NULL 
		END AS numeric_precision,
		CASE 
			WHEN a.atttypid IN (1700) THEN 10 
			ELSE NULL 
		END AS numeric_precision_radix,
		CASE 
			WHEN a.atttypid IN (1082, 1083, 1114, 1184) THEN CASE WHEN a.atttypmod >= 0 THEN a.atttypmod ELSE 6 END
			ELSE NULL 
		END AS datetime_precision,
		'NO' AS is_self_referencing, -- Static fallback for mapping parity
		a.attnum::text AS dtd_identifier,
		CASE WHEN a.attgenerated <> '' THEN 'ALWAYS' ELSE NULL END AS generation_expression,
		NULL::integer AS maximum_cardinality, -- Default optimization assignment
		t.relhasindex AS hasindexes,
		(SELECT count(*) > 0 FROM pg_catalog.pg_trigger WHERE tgrelid = t.oid AND NOT tgisinternal) AS hastriggers,
		COALESCE(cons.constraints, '[]'::jsonb) AS column_constraints
	FROM
		pg_catalog.pg_attribute a
	JOIN 
		pg_catalog.pg_class t ON a.attrelid = t.oid
	JOIN 
		pg_catalog.pg_namespace n ON t.relnamespace = n.oid
	JOIN 
		pg_catalog.pg_type col_type ON a.atttypid = col_type.oid
	LEFT JOIN 
		pg_catalog.pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
	LEFT JOIN (
		-- Aggregated Constraints using pg_catalog matrices
		SELECT 
			con.conrelid,
			con_att.attnum AS local_attnum,
			jsonb_agg(jsonb_build_object(
				'name', con.conname,
				'type', CASE con.contype 
							WHEN 'p' THEN 'PRIMARY KEY' 
							WHEN 'f' THEN 'FOREIGN KEY' 
							WHEN 'u' THEN 'UNIQUE' 
							WHEN 'c' THEN 'CHECK' 
						END,
				'referenced_table', ref_t.relname,
				'referenced_column', ref_a.attname
			)) AS constraints
		FROM 
			pg_catalog.pg_constraint con
		-- Unnest arrays to handle single and multi-column keys cleanly
		CROSS JOIN LATERAL 
			unnest(con.conkey) WITH ORDINALITY AS con_att(attnum, ord)
		LEFT JOIN 
			pg_catalog.pg_class ref_t ON con.confrelid = ref_t.oid
		LEFT JOIN LATERAL 
			unnest(con.confkey) WITH ORDINALITY AS conf_att(attnum, ord) ON con_att.ord = conf_att.ord
		LEFT JOIN 
			pg_catalog.pg_attribute ref_a ON ref_a.attrelid = con.confrelid AND ref_a.attnum = conf_att.attnum
		GROUP BY 
			con.conrelid, con_att.attnum
	) AS cons ON cons.conrelid = t.oid AND cons.local_attnum = a.attnum
	WHERE
		n.nspname = $1                    -- Targets schema ($1 e.g., 'public')
		AND t.relkind = 'r'               -- Ordinary tables only (excludes views/indexes)
		AND a.attnum > 0                  -- Drops system columns like xmin, ctid
		AND NOT a.attisdropped            -- Excludes deleted schema fragments
	ORDER BY
		t.relname,
		a.attnum;
	`

	return query
}

func (p *postgres) Ingest(
	ctx context.Context,
	entity string,
	columns []string,
	stream stream.RowStream,
	onProgress func(n int),
) (int64, error) {
	const chunkSize = 500000 // 500k rows per transaction batch
	var totalInserted int64
	for {
		if err := ctx.Err(); err != nil {
			return totalInserted, err
		}

		adapter := newChunkAdapter(stream, chunkSize, onProgress)

		insertedRows, err := p.pg.CopyFrom(
			ctx,
			pgx.Identifier{entity},
			columns,
			adapter,
		)
		if err != nil {
			return totalInserted, fmt.Errorf("ingestion failed at %d rows for '%s': %w", totalInserted, entity, err)
		}

		totalInserted += insertedRows

		// Stop looping when the underlying stream has run out of data
		if adapter.exhausted {
			break
		}
	}

	return totalInserted, nil
}

func (p *postgres) GetBounds(ctx context.Context, tableName string, colName string) (int, int, error) {
	table := pgx.Identifier{tableName}.Sanitize()
	// col := pgx.Identifier{colName}.Sanitize()

	// query := fmt.Sprintf(`
	// 	SELECT
	// 		COALESCE(MIN(seq), 0),
	// 		COALESCE(MAX(seq), 0)
	// 	FROM (
	// 		SELECT (row_number() OVER (ORDER BY %s)) AS seq
	// 		FROM %s
	// 	) sub
	// `, col, table)
	query := fmt.Sprintf(`
	    SELECT
	        CASE WHEN COUNT(*) = 0 THEN 0 ELSE 1 END,
	        COUNT(*)
	    FROM %s
	`, table)

	var minVal, maxVal int
	err := p.pg.QueryRow(ctx, query).Scan(&minVal, &maxVal)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get bounds for %s.%s: %w", tableName, colName, err)
	}

	return minVal, maxVal, nil
}

// chunkAdapter wraps your RowStream and limits execution to chunkSize rows per COPY pass.
type chunkAdapter struct {
	stream     stream.RowStream
	limit      int
	count      int
	exhausted  bool // true when the underlying stream naturally completes
	onProgress func(n int)
	unreported int
}

func newChunkAdapter(s stream.RowStream, chunkSize int, onProgress func(n int)) *chunkAdapter {
	return &chunkAdapter{
		stream:     s,
		limit:      chunkSize,
		onProgress: onProgress,
	}
}

func (a *chunkAdapter) Next() bool {
	if a.count >= a.limit {
		a.flushProgress()
		return false
	}

	if !a.stream.Next() {
		a.exhausted = true
		a.flushProgress()
		return false
	}

	a.count++
	a.unreported++

	// Flush progress every 100 rows to ensure immediate updates for small counts
	if a.unreported >= 100 {
		a.flushProgress()
	}

	return true
}

func (a *chunkAdapter) flushProgress() {
	if a.unreported > 0 && a.onProgress != nil {
		a.onProgress(a.unreported)
		a.unreported = 0
	}
}

func (a *chunkAdapter) Values() ([]any, error) { return a.stream.Values() }
func (a *chunkAdapter) Err() error             { return a.stream.Err() }
