package postgres

// Column represents the metadata for a single table column.
type Column struct {
	Name                   string
	Default                *string // Can be null
	DataType               string
	UdtName                string
	IsNullable             string
	CharacterMaximumLength *int32 // Can be null
	NumericPrecision       *int32 // Can be null
	NumericPrecisionRadix  *int32 // Can be null
	DatetimePrecision      *int32 // Can be null
	IsSelfReferencing      string
	DtdIdentifier          string
	GenerationExpression   *string // Can be null
	MaximumCardinality     *int32  // Can be null
}

// Table represents the parent table containing its metadata and columns.
type Table struct {
	Schema      string
	Name        string
	HasIndexes  bool
	HasTriggers bool
	Columns     []Column
}

// FlatRow maps directly to the columns returned by the new joined SQL query.
// Order must match your SELECT statement exactly for RowToStructByPos.
type FlatRow struct {
	// TableSchema            string
	TableName              string
	ColumnName             string
	ColumnDefault          *string
	DataType               string
	UdtName                string
	IsNullable             string
	CharacterMaximumLength *int32
	NumericPrecision       *int32
	NumericPrecisionRadix  *int32
	DatetimePrecision      *int32
	IsSelfReferencing      string
	DtdIdentifier          string
	GenerationExpression   *string
	MaximumCardinality     *int32
	HasIndexes             bool
	HasTriggers            bool
}
