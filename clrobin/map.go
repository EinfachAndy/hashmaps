package clrobin

import (
	"sync"

	"github.com/EinfachAndy/hashmaps/shared"
)

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
	sync.RWMutex

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

//go:inline
func newBucketArray[K comparable, V comparable](capacity uintptr) []bucket[K, V] {
	buckets := make([]bucket[K, V], capacity)

	for i := range buckets {
		buckets[i].psl = emptyBucket
	}

	return buckets
}

//go:inline
func (m *CLRobin[K, V]) grow() {
	m.resize((m.capMinus1 + 1) * 2)
}

//go:inline
func (m *CLRobin[K, V]) resize(n uintptr) {
	newm := CLRobin[K, V]{
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

// emplace applies the Robin Hood creed to all following buckets until a empty is found.
// Robin Hood creed: "takes from the rich and gives to the poor".
// rich means, low psl
// poor means, higher psl
//
// The result is a better distribution of the PSL values,
// where the expected length of the longest PSL is O(log(n)).
//
//go:inline
func (m *CLRobin[K, V]) emplace(current *bucket[K, V], idx uintptr) {
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

// Reserve sets the number of buckets to the most appropriate to contain at least n elements.
// If n is lower than that, the function may have no effect.
func (m *CLRobin[K, V]) Reserve(n uintptr) {
	m.Lock()
	defer m.Unlock()

	var (
		needed = uintptr(float32(n) / m.maxLoad)
		newCap = uintptr(shared.NextPowerOf2(uint64(needed)))
	)

	if uintptr(cap(m.buckets)) < newCap {
		m.resize(newCap)
	}
}

func (m *CLRobin[K, V]) Size() int {
	m.RLock()
	defer m.RUnlock()

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
	m.RLock()
	defer m.RUnlock()

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

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (m *CLRobin[K, V]) LoadAndDelete(key K) (V, bool) {
	m.Lock()
	defer m.Unlock()

	var (
		idx     = m.hasher(key) & m.capMinus1
		current *bucket[K, V]
		v       V
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
		return v, false
	}
	v = current.value

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

	return v, true
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
func (m *CLRobin[K, V]) Swap(key K, value V) (V, bool) {
	m.Lock()
	defer m.Unlock()

	if m.length >= m.nextResize {
		m.grow()
	}

	var (
		idx = m.hasher(key) & m.capMinus1
		psl = int8(0)
	)

	// search for the key
	for ; psl <= m.buckets[idx].psl; psl++ {
		if m.buckets[idx].key == key {
			old := m.buckets[idx].value
			m.buckets[idx].value = value
			return old, true // update already existing value
		}
		// next index
		idx = (idx + 1) & m.capMinus1
	}

	m.length++

	newBucket := bucket[K, V]{key: key, value: value, psl: psl}
	m.emplace(&newBucket, idx)

	return value, false
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
	m.RLock()
	defer m.RUnlock()

	for i := range m.buckets {
		if m.buckets[i].psl != emptyBucket {
			if stop := f(m.buckets[i].key, m.buckets[i].value); stop {
				// stop iteration
				return
			}
		}
	}
}

// Clear deletes all the entries, resulting in an empty Map.
func (m *CLRobin[K, V]) Clear() {
	m.Lock()
	defer m.Unlock()

	for i := range m.buckets {
		m.buckets[i].psl = emptyBucket
	}

	m.length = 0
}
