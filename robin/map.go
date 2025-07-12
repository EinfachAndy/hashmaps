package robin

import (
	"fmt"

	"github.com/EinfachAndy/hashmaps/shared"
)

const (
	emptyBucket = -1
)

type bucket[K comparable, V any] struct {
	key K
	// psl is the probe sequence length (PSL), which is the distance value from
	// the optimum insertion. -1 or `emptyBucket` signals a free slot.
	// inspired from:
	//  - https://programming.guide/robin-hood-hashing.html
	//  - https://cs.uwaterloo.ca/research/tr/1986/CS-86-14.pdf
	psl   int8
	value V
}

// RobinHood is a hash map that uses linear probing in combination with
// robin hood hashing as collision strategy. The map tracks the distance
// from the optimum bucket and minimized the variance over all buckets.
// The expected max PSL for a full robin hood hash map is O(ln(n)).
// The max load factor can be changed with `MaxLoad()`.
// The result is a good trade off between performance and memory consumption.
// Note that the performance for a open addressing hash map depends
// also on the key and value size. For higher storage sizes for the
// keys and values use a hashmap that uses another strategy
// like the golang std map or the Unordered map.
type RobinHood[K comparable, V any] struct {
	buckets []bucket[K, V]
	hasher  shared.HashFn[K]
	// length stores the current inserted elements
	length uintptr
	// capMinus1 is used for a bitwise AND on the hash value,
	// because the size of the underlying array is a power of two value
	capMinus1  uintptr
	nextResize uintptr

	maxLoad float32
}

//go:inline
func newBucketArray[K comparable, V any](capacity uintptr) []bucket[K, V] {
	buckets := make([]bucket[K, V], capacity)

	for i := range buckets {
		buckets[i].psl = emptyBucket
	}

	return buckets
}

// New creates a ready to use `RobinHood` hash map with default settings.
func New[K comparable, V any]() *RobinHood[K, V] {
	return NewWithHasher[K, V](shared.GetHasher[K]())
}

// NewWithHasher same as `NewRobinHood` but with a given hash function.
func NewWithHasher[K comparable, V any](hasher shared.HashFn[K]) *RobinHood[K, V] {
	m := &RobinHood[K, V]{
		hasher:  hasher,
		maxLoad: shared.DefaultMaxLoad,
	}
	m.Reserve(shared.DefaultSize)

	return m
}

// Get returns the value stored for this key, or false if there is no such value.
//
// Note:
//   - There exists also other search strategies like organ-pipe search
//     or smart search, where searching starts around the mean value
//     (mean, mean − 1, mean + 1, mean − 2, mean + 2, ...)
//   - Here it is used the simplest technic, which is more cache friendly and
//     does not track other metic values.
func (m *RobinHood[K, V]) Get(key K) (V, bool) {
	var (
		idx = m.hasher(key) & m.capMinus1
		v   V
	)

	for psl := int8(0); psl <= m.buckets[idx].psl; psl++ {
		if m.buckets[idx].key == key {
			return m.buckets[idx].value, true
		}
		// next index
		idx = (idx + 1) & m.capMinus1
	}

	return v, false
}

// Reserve sets the number of buckets to the most appropriate to contain at least n elements.
// If n is lower than that, the function may have no effect.
func (m *RobinHood[K, V]) Reserve(n uintptr) {
	var (
		needed = uintptr(float32(n) / m.maxLoad)
		newCap = uintptr(shared.NextPowerOf2(uint64(needed)))
	)

	if uintptr(cap(m.buckets)) < newCap {
		m.resize(newCap)
	}
}

func (m *RobinHood[K, V]) resize(n uintptr) {
	newm := RobinHood[K, V]{
		capMinus1:  n - 1,
		length:     m.length,
		buckets:    newBucketArray[K, V](n),
		hasher:     m.hasher,
		maxLoad:    m.maxLoad,
		nextResize: uintptr(float32(n) * m.maxLoad),
	}

	for i := range m.buckets {
		if m.buckets[i].psl != emptyBucket {
			idx := newm.hasher(m.buckets[i].key) & newm.capMinus1
			m.buckets[i].psl = 0
			newm.emplace(&m.buckets[i], idx)
		}
	}

	m.nextResize = newm.nextResize
	m.capMinus1 = newm.capMinus1
	m.buckets = newm.buckets
}

// Put maps the given key to the given value. If the key already exists its
// value will be overwritten with the new value.
// Returns true, if the element is a new item in the hash map.
func (m *RobinHood[K, V]) Put(key K, val V) bool {
	if m.length >= m.nextResize {
		m.resize(uintptr(cap(m.buckets)) * 2)
	}

	var (
		idx = m.hasher(key) & m.capMinus1
		psl = int8(0)
	)

	// search for the key
	for ; psl <= m.buckets[idx].psl; psl++ {
		if m.buckets[idx].key == key {
			m.buckets[idx].value = val
			return false // update already existing value
		}
		// next index
		idx = (idx + 1) & m.capMinus1
	}

	m.length++

	newBucket := bucket[K, V]{key: key, value: val, psl: psl}
	m.emplace(&newBucket, idx)

	return true
}

// emplace applies the Robin Hood creed to all following buckets until a empty is found.
// Robin Hood creed: "takes from the rich and gives to the poor".
// rich means, low psl
// poor means, higher psl
//
// The result is a better distribution of the PSL values,
// where the expected length of the longest PSL is O(log(n)).
//
//go:inline
func (m *RobinHood[K, V]) emplace(current *bucket[K, V], idx uintptr) {
	for ; ; current.psl++ {
		if m.buckets[idx].psl == emptyBucket {
			// emplace the element, a valid bucket was found
			m.buckets[idx] = *current
			return
		}

		if current.psl > m.buckets[idx].psl {
			// swap values, apply the Robin Hood creed
			*current, m.buckets[idx] = m.buckets[idx], *current
		}

		// next index
		idx = (idx + 1) & m.capMinus1
	}
}

// Remove removes the specified key-value pair from the map.
// Returns true, if the element was in the hash map.
func (m *RobinHood[K, V]) Remove(key K) bool {
	var (
		idx     = m.hasher(key) & m.capMinus1
		current *bucket[K, V]
	)

	// search for the key
	for psl := int8(0); psl <= m.buckets[idx].psl; psl++ {
		if m.buckets[idx].key == key {
			current = &m.buckets[idx]
			break
		}
		// next index
		idx = (idx + 1) & m.capMinus1
	}

	if current == nil {
		return false
	}

	// remove the key
	m.length--
	// mark as empty, because we want to remove it
	current.psl = emptyBucket

	idx = (idx + 1) & m.capMinus1
	next := &m.buckets[idx]
	// now, back shift all buckets until we found a optimum or empty one
	for next.psl > 0 {
		next.psl--
		*current, *next = *next, *current // swap values
		current = next
		idx = (idx + 1) & m.capMinus1
		next = &m.buckets[idx]
	}

	return true
}

// Clear removes all key-value pairs from the map.
func (m *RobinHood[K, V]) Clear() {
	for i := range m.buckets {
		m.buckets[i].psl = emptyBucket
	}

	m.length = 0
}

// Load return the current load of the hash map.
func (m *RobinHood[K, V]) Load() float32 {
	return float32(m.length) / float32(cap(m.buckets))
}

// MaxLoad forces resizing if the ratio is reached.
// Useful values are in range [0.5-0.9].
// Returns ErrOutOfRange if `lf` is not in the open range (0.0,1.0).
func (m *RobinHood[K, V]) MaxLoad(lf float32) error {
	if lf <= 0.0 || lf >= 1.0 {
		return fmt.Errorf("%f: %w", lf, shared.ErrOutOfRange)
	}

	m.maxLoad = lf
	m.nextResize = uintptr(float32(cap(m.buckets)) * lf)

	return nil
}

// Size returns the number of items in the map.
func (m *RobinHood[K, V]) Size() int {
	return int(m.length)
}

// Copy returns a copy of this map.
func (m *RobinHood[K, V]) Copy() *RobinHood[K, V] {
	newM := &RobinHood[K, V]{
		buckets:    make([]bucket[K, V], cap(m.buckets)),
		capMinus1:  m.capMinus1,
		length:     m.length,
		hasher:     m.hasher,
		maxLoad:    m.maxLoad,
		nextResize: m.nextResize,
	}

	copy(newM.buckets, m.buckets)

	return newM
}

// Each calls 'fn' on every key-value pair in the hash map in no particular order.
// If 'fn' returns true, the iteration stops.
func (m *RobinHood[K, V]) Each(fn func(key K, val V) bool) {
	for i := range m.buckets {
		if m.buckets[i].psl != emptyBucket {
			if stop := fn(m.buckets[i].key, m.buckets[i].value); stop {
				// stop iteration
				return
			}
		}
	}
}
