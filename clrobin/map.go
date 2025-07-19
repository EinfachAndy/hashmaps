package clrobin

import "github.com/EinfachAndy/hashmaps/shared"

const (
	emptyBucket = -1
)

type bucket[K comparable, V comparable] struct {
	key K
	// psl is the probe sequence length (PSL), which is the distance value from
	// the optimum insertion. -1 or `emptyBucket` signals a free slot.
	// inspired from:
	//  - https://programming.guide/robin-hood-hashing.html
	//  - https://cs.uwaterloo.ca/research/tr/1986/CS-86-14.pdf
	psl   int8
	value V
}

// CLRobin is a concurrent locked robin hood hashmap.
type CLRobin[K comparable, V comparable] struct {
	hasher shared.HashFn[K]

	// length stores the current inserted elements
	length     uintptr
	nextResize uintptr
	capMinus1  uintptr
	buckets    []bucket[K, V]
	maxLoad    float32
}

// New creates a new ready to use concurrent locked robin hood hashmap.
func New[K comparable, V comparable]() *CLRobin[K, V] {
	return NewWithHasher[K, V](shared.GetHasher[K]())
}

// NewWithHasher constructs a new hashmap with the given hasher.
func NewWithHasher[K comparable, V comparable](hasher shared.HashFn[K]) *CLRobin[K, V] {
	m := &CLRobin[K, V]{
		hasher:  hasher,
		maxLoad: shared.DefaultMaxLoad,
	}
	m.Reserve(shared.DefaultSize)

	return m
}

func (m *CLRobin[K, V]) grow() {
	m.resize((m.capMinus1 + 1) * 2)
}

func (m *CLRobin[K, V]) resize(n uintptr) {
}

// Reserve sets the number of buckets to the most appropriate to contain at least n elements.
// If n is lower than that, the function may have no effect.
func (m *CLRobin[K, V]) Reserve(n uintptr) {
	var (
		needed = uintptr(float32(n) / m.maxLoad)
		newCap = uintptr(shared.NextPowerOf2(uint64(needed)))
	)

	if uintptr(cap(m.buckets)) < newCap {
		m.resize(newCap)
	}
}

func (m *CLRobin[K, V]) Size() int {
	return int(m.length)
}

/*****************************************
 * Golang concurrent interface functions
 *****************************************/

// Delete deletes the value for a key.
func (m *CLRobin[K, V]) Delete(key K) {
	_, _ = m.LoadAndDelete(key)
}

// Load returns the value stored in the map for a key,
// or nil if no value is present.
// The ok result indicates whether value was found in the map.
func (m *CLRobin[K, V]) Load(key K) (V, bool) {
	var v V

	return v, false
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (m *CLRobin[K, V]) LoadAndDelete(key K) (V, bool) {
	var v V

	return v, false
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (m *CLRobin[K, V]) LoadOrStore(key K, value V) (V, bool) {
	var v V

	return v, false
}

// Store sets the value for a key.
func (m *CLRobin[K, V]) Store(key K, value V) {
	_, _ = m.Swap(key, value)
}

// Swap swaps the value for a key and returns the previous value if any.
// The loaded result reports whether the key was present.
func (m *CLRobin[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	var v V

	return v, false
}

// CompareAndSwap swaps the old and new values for key
// if the value stored in the map is equal to old.
// The old value must be of a comparable type.
func (m *CLRobin[K, V]) CompareAndSwap(key K, old, new V) (swapped bool) {
	return false
}

func (m *CLRobin[K, V]) CompareAndDelete(key K, old V) (deleted bool) {
	return false
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
//
// Range does not necessarily correspond to any consistent snapshot of the Map's
// contents: no key will be visited more than once, but if the value
// for any key is stored or deleted concurrently (including by f),
// Range may reflect any mapping for that key from any point during the Range call.
// Range does not block other methods on the receiver; even f itself may call any method on m.
//
// Range may be O(N) with the number of elements in the map even
// if f returns false after a constant number of calls.
func (m *CLRobin[K, V]) Range(f func(key K, value V) bool) {
}

// Clear deletes all the entries, resulting in an empty Map.
func (m *CLRobin[K, V]) Clear() {
}
