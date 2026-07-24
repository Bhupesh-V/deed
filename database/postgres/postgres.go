package postgres

import (
	"context"
	"deed/database"
	"deed/internal/models"
	"encoding/json"
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
		t.hastriggers,
		COALESCE(cons.constraints, '[]'::jsonb) AS column_constraints
	FROM
		information_schema.columns AS c
	LEFT JOIN
		pg_catalog.pg_tables AS t 
		ON c.table_name = t.tablename 
		AND c.table_schema = t.schemaname
	LEFT JOIN (
		SELECT
			tc.table_schema,
			tc.table_name,
			kcu.column_name,
			jsonb_agg(jsonb_build_object(
				'name', tc.constraint_name,
				'type', tc.constraint_type,
				'referenced_table', ccu.table_name,
				'referenced_column', ccu.column_name
			)) AS constraints
		FROM
			information_schema.table_constraints AS tc
		JOIN
			information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		LEFT JOIN
			information_schema.referential_constraints AS rc
			ON tc.constraint_name = rc.constraint_name
			AND tc.table_schema = rc.constraint_schema
		LEFT JOIN
			information_schema.constraint_column_usage AS ccu
			ON rc.unique_constraint_name = ccu.constraint_name
			AND rc.unique_constraint_schema = ccu.table_schema
		GROUP BY
			tc.table_schema, tc.table_name, kcu.column_name
	) AS cons
		ON c.table_schema = cons.table_schema
		AND c.table_name = cons.table_name
		AND c.column_name = cons.column_name
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

			cols = append(cols, models.Column{
				Name: c.Name,
				Type: models.DataType{
					BaseType: c.UdtName,
				},
				Constraint: constraints,
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

func (p *postgres) BulkInsert(ctx context.Context) {
	panic("not implemented") // TODO: Implement
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
