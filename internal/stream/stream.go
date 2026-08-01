package stream

// RowStream is an abstract stream of generated mock data rows.
type RowStream interface {
	// Next advances the stream. Returns false when generation is complete.
	Next() bool
	// Values returns the current row's field values in column order.
	Values() ([]any, error)
	// Err returns any error encountered during row generation.
	Err() error
}
