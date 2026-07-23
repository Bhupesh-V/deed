package models

// ConstraintType enforces type safety using readable string constants.
type ConstraintType string

const (
	PrimaryKey ConstraintType = "PRIMARY_KEY"
	ForeignKey ConstraintType = "FOREIGN_KEY"
	Unique     ConstraintType = "UNIQUE"
	Check      ConstraintType = "CHECK"
	Exclusion  ConstraintType = "EXCLUSION"
)
