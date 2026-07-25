package models

// ConstraintType enforces type safety using readable string constants.
type ConstraintType string

const (
	PrimaryKey ConstraintType = "PRIMARY KEY"
	ForeignKey ConstraintType = "FOREIGN KEY"
	Unique     ConstraintType = "UNIQUE"
	Check      ConstraintType = "CHECK"
	Exclusion  ConstraintType = "EXCLUSION"
)

func (c ConstraintType) String() string {
	return string(c)
}
