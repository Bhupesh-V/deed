package postgres

// Constraint represents a standard table constraint structure captured via JSONB.
type Constraint struct {
	Name             string  `json:"name"`
	Type             string  `json:"type"`
	ReferencedTable  *string `json:"referenced_table"`  // Will be populated if Type == "FOREIGN KEY"
	ReferencedColumn *string `json:"referenced_column"` // Will be populated if Type == "FOREIGN KEY"
}

// Column represents the metadata for a single table column.
type Column struct {
	Name                   string
	Default                *string // Can be null
	IsPrimaryKey           bool
	DataType               string
	UdtName                string
	IsNullable             string
	IsIdentity             string
	IdentityGeneration     *string
	CharacterMaximumLength *int32 // Can be null
	NumericPrecision       *int   // Can be null
	NumericPrecisionRadix  *int   // Can be null
	NumericPrecisionScale  *int
	DatetimePrecision      *int32 // Can be null
	IsSelfReferencing      string
	DtdIdentifier          string
	GenerationExpression   *string // Can be null
	MaximumCardinality     *int32  // Can be null
	Constraints            []Constraint
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
	TableName              string
	ColumnName             string
	IsPrimaryKey           bool
	ColumnDefault          *string
	DataType               string
	UdtName                string
	IsNullable             string
	IsIdentity             string
	IdentityGeneration     *string
	CharacterMaximumLength *int32
	NumericPrecision       *int
	NumericPrecisionRadix  *int
	NumericPrecisionScale  *int
	DatetimePrecision      *int32
	IsSelfReferencing      string
	DtdIdentifier          string
	GenerationExpression   *string
	MaximumCardinality     *int32
	HasIndexes             bool
	HasTriggers            bool
	ColumnConstraints      []byte // Captures raw jsonb bytes from the query
}
