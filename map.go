// Package hashmaps implements several types of hash tables.
package hashmaps

import "errors"

const (
	// DefaultMaxLoad is the default value for the load factor for:
	// -Hopscotch
	// -Robin Hood
	// hashmaps, which can be changed with MaxLoad(). This value is a
	// trade-off of runtime and memory consumption.
	DefaultMaxLoad = 0.7
)

var (
	// ErrOutOfRange signals an out of range request.
	ErrOutOfRange = errors.New("out of range")
)

// IHashMap collects the basic hash maps operations as function points.
type IHashMap[K comparable, V any] struct {
	Get     func(key K) (V, bool)
	Reserve func(n uintptr)
	Load    func() float32
	Put     func(key K, val V) bool
	Remove  func(key K) bool
	Clear   func()
	Size    func() int
	Each    func(fn func(key K, val V) bool)
}
