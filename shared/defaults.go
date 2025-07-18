package shared

const (
	// DefaultMaxLoad is the default value for the load factor.
	// It can be changed with `MaxLoad`. This value is a
	// trade-off of runtime and memory consumption.
	DefaultMaxLoad = 0.7

	// DefaultSize is the default for the amount before
	// the first resize happens.
	DefaultSize = 4
)
