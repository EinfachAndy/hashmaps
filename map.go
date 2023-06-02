// Package hashmaps implements several types of hash tables.
package hashmaps

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
