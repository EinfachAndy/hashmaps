package shared

const (
	// DefaultMaxLoad is the default value for the load factor for:
	// -Hopscotch
	// -Robin Hood
	// hashmaps, which can be changed with MaxLoad(). This value is a
	// trade-off of runtime and memory consumption.
	DefaultMaxLoad = 0.7

	DefaultSize = 4
)
