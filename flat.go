package hashmaps

import "fmt"

type fBucket[K comparable, V any] struct {
	key   K
	value V
}

// Flat is a open addressing hash map implementation which uses linear probing
// as conflict resolution.
type Flat[K comparable, V any] struct {
	buckets   []fBucket[K, V]
	empty     K
	hasher    HashFn[K]
	capMinus1 uintptr
	length    uintptr
}

// go:inline
func newfBucketArray[K comparable, V any](capacity uintptr, empty K) []fBucket[K, V] {
	buckets := make([]fBucket[K, V], capacity)
	for i := range buckets {
		buckets[i].key = empty
	}
	return buckets
}

// NewFlat creates a new ready to use flat hash map.
//
// Note:
// This map has zero memory overhead per bucket and uses therefore
// the golang default variable initialization representation as tracking.
// This means in details a Get, Put or Remove call fails, if the key is:
//   - 0 (int, uint, uint64, ...)
//   - 0.0 (float32, float64)
//   - "" (string)
func NewFlat[K comparable, V any]() *Flat[K, V] {
	var empty K // uses default zero representation
	return NewFlatWithHasher[K, V](empty, GetHasher[K]())
}

// NewFlatWithHasher constructs a new map with the given hasher.
// Furthermore the representation for a empty bucket can be set.
func NewFlatWithHasher[K comparable, V any](empty K, hasher HashFn[K]) *Flat[K, V] {
	return &Flat[K, V]{
		buckets:   newfBucketArray[K, V](4, empty),
		capMinus1: 3,
		hasher:    hasher,
	}
}

// Get returns the value stored for this key, or false if not found.
func (m *Flat[K, V]) Get(key K) (V, bool) {
	if key == m.empty {
		panic(fmt.Sprintf("key %v is same as empty %v", key, m.empty))
	}
	hash := m.hasher(key)
	idx := hash & m.capMinus1

	for m.buckets[idx].key != m.empty {
		if m.buckets[idx].key == key {
			return m.buckets[idx].value, true
		}

		// next index
		idx = (idx + 1) & m.capMinus1
	}

	var v V
	return v, false
}

func (m *Flat[K, V]) resize(n uintptr) {
	newm := Flat[K, V]{
		capMinus1: n - 1,
		length:    m.length,
		empty:     m.empty,
		hasher:    m.hasher,
		buckets:   newfBucketArray[K, V](n, m.empty),
	}

	for i := range m.buckets {
		if m.buckets[i].key != m.empty {
			newm.emplace(m.buckets[i].key, m.buckets[i].value)
		}
	}
	m.capMinus1 = newm.capMinus1
	m.buckets = newm.buckets
}

// emplace does not check if the key is already in.
func (m *Flat[K, V]) emplace(key K, val V) {
	hash := m.hasher(key)
	idx := hash & m.capMinus1
	for {
		if m.buckets[idx].key == m.empty {
			break
		}

		// next index
		idx = (idx + 1) & m.capMinus1
	}

	// we have a position for emplacing
	m.buckets[idx].key = key
	m.buckets[idx].value = val
}

// Put maps the given key to the given value. If the key already exists its
// value will be overwritten with the new value.
func (m *Flat[K, V]) Put(key K, val V) bool {
	if key == m.empty {
		panic(fmt.Sprintf("key %v is same as empty %v", key, m.empty))
	}

	if m.length >= uintptr(cap(m.buckets))/2 {
		m.resize(uintptr(cap(m.buckets)) * 2)
	}

	hash := m.hasher(key)
	idx := hash & m.capMinus1

	for m.buckets[idx].key != m.empty {
		if m.buckets[idx].key == key {
			m.buckets[idx].value = val
			return false
		}
		// next index
		idx = (idx + 1) & m.capMinus1
	}

	m.buckets[idx].key = key
	m.buckets[idx].value = val
	m.length++
	return true
}

// Remove removes the specified key-value pair from the map.
func (m *Flat[K, V]) Remove(key K) bool {
	if key == m.empty {
		panic(fmt.Sprintf("key %v is same as empty %v", key, m.empty))
	}

	hash := m.hasher(key)
	idx := hash & m.capMinus1

	for m.buckets[idx].key != m.empty && !(m.buckets[idx].key == key) {
		idx = (idx + 1) & m.capMinus1
	}

	if m.buckets[idx].key == m.empty {
		return false
	}

	m.buckets[idx].key = m.empty
	m.length--

	for {
		idx = (idx + 1) & m.capMinus1
		if m.buckets[idx].key == m.empty {
			break
		}
		k := m.buckets[idx].key
		v := m.buckets[idx].value
		m.buckets[idx].key = m.empty
		m.emplace(k, v)
	}

	// shrink if it is 12.5% full or less
	// if m.length > 2 && m.length <= uintptr(cap(m.buckets))/8 {
	// 	m.resize(uintptr(cap(m.buckets)) / 2)
	// }
	return true
}

// Reserve sets the number of buckets to the most appropriate to contain at least n elements.
// If n is lower than that, the function may have no effect.
func (m *Flat[K, V]) Reserve(n uintptr) {
	newCap := uintptr(NextPowerOf2(uint64(2 * n)))
	if uintptr(cap(m.buckets)) < newCap {
		m.resize(newCap)
	}
}

// Clear removes all key-value pairs from the map.
func (m *Flat[K, V]) Clear() {
	for i := range m.buckets {
		m.buckets[i].key = m.empty
	}
	m.length = 0
}

// Size returns the number of items in the map.
func (m *Flat[K, V]) Size() int {
	return int(m.length)
}

// Load return the current load of the hash map.
func (m *Flat[K, V]) Load() float32 {
	return float32(m.length) / float32(len(m.buckets))
}

func (m *Flat[K, V]) Copy() *Flat[K, V] {
	newM := &Flat[K, V]{
		capMinus1: m.capMinus1,
		length:    m.length,
		hasher:    m.hasher,
		empty:     m.empty,
	}
	copy(newM.buckets, m.buckets)
	return newM
}

// Each calls 'fn' on every key-value pair in the hashmap in no particular order.
func (m *Flat[K, V]) Each(fn func(key K, val V) bool) {
	for i := range m.buckets {
		if m.buckets[i].key != m.empty {
			if stop := fn(m.buckets[i].key, m.buckets[i].value); stop {
				// stop iteration
				return
			}
		}
	}
}
