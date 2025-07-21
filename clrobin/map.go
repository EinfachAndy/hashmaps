package clrobin

import (
	"sync"
	"sync/atomic"

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

//go:inline
func (b *bucket[K, V]) isEmpty() bool {
	return b.psl == emptyBucket
}

type storage[K comparable, V comparable] struct {
	length     atomic.Int64
	nextResize int
	capMinus1  int
	buckets    []bucket[K, V]
}

// CLRobin is a concurrent locked robin hood hashmap.
type CLRobin[K comparable, V comparable] struct {
	sync.RWMutex

	hasher  shared.HashFn[K]
	maxLoad float32

	storage atomic.Pointer[storage[K, V]]
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
	m.storage.Store(newStorage[K, V](shared.DefaultSize, m.maxLoad))

	return m
}

//go:inline
func newStorage[K comparable, V comparable](capacity int, maxLoad float32) *storage[K, V] {
	buckets := make([]bucket[K, V], capacity)

	for i := range buckets {
		buckets[i].psl = emptyBucket
	}

	return &storage[K, V]{
		capMinus1:  capacity - 1,
		buckets:    buckets,
		nextResize: int(float32(capacity) * maxLoad),
	}
}

//go:inline
func (m *CLRobin[K, V]) checkForResize() *storage[K, V] {
	// next 3 lines must be atomic without mutex
	s := m.storage.Load()
	if int(s.length.Load()) >= s.nextResize {
		m.resize((s.capMinus1 + 1) * 2)
		s = m.storage.Load()
	}

	return s
}

//go:inline
func (m *CLRobin[K, V]) resize(n int) {
	var (
		new = newStorage[K, V](n, m.maxLoad)
		old = m.storage.Load()
	)

	new.length.Store(old.length.Load())

	for i := range old.buckets {
		if !old.buckets[i].isEmpty() {
			b := old.buckets[i]
			b.psl = 0
			idx := m.hasher(b.key) & uintptr(new.capMinus1)
			new.emplace(&b, idx)
		}
	}

	m.storage.Store(new)
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
func (s *storage[K, V]) emplace(current *bucket[K, V], idx uintptr) {
	for ; ; current.psl++ {
		if s.buckets[idx].isEmpty() {
			// emplace the element, a valid bucket was found
			s.buckets[idx] = *current
			return
		}

		if current.psl > s.buckets[idx].psl {
			// swap values, apply the Robin Hood creed
			*current, s.buckets[idx] = s.buckets[idx], *current
		}

		// next index
		idx = (idx + 1) & uintptr(s.capMinus1)
	}
}

// Reserve sets the number of buckets to the most appropriate to contain at least n elements.
// If n is lower than that, the function may have no effect.
func (m *CLRobin[K, V]) Reserve(n uintptr) {
	m.Lock()
	defer m.Unlock()

	var (
		needed = int(float32(n) / m.maxLoad)
		newCap = int(shared.NextPowerOf2(uint64(needed)))
	)

	if cap(m.storage.Load().buckets) < newCap {
		m.resize(newCap)
	}
}

func (m *CLRobin[K, V]) Size() int {
	m.RLock()
	defer m.RUnlock()

	return int(m.storage.Load().length.Load())
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
		s   = m.storage.Load()
		idx = m.hasher(key) & uintptr(s.capMinus1)
		v   V
	)

	for psl := int8(0); psl <= s.buckets[idx].psl; psl++ {
		if s.buckets[idx].key == key {
			return s.buckets[idx].value, true
		}
		// next index
		idx = (idx + 1) & uintptr(s.capMinus1)
	}

	return v, false
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (m *CLRobin[K, V]) LoadAndDelete(key K) (V, bool) {
	m.Lock()
	defer m.Unlock()

	var (
		s       = m.storage.Load()
		idx     = m.hasher(key) & uintptr(s.capMinus1)
		current *bucket[K, V]
		v       V
	)

	// search for the key
	for psl := int8(0); psl <= s.buckets[idx].psl; psl++ {
		if s.buckets[idx].key == key {
			current = &s.buckets[idx]
			break
		}
		// next index
		idx = (idx + 1) & uintptr(s.capMinus1)
	}

	if current == nil {
		return v, false
	}
	v = current.value

	// remove the key
	s.length.Add(-1)
	// mark as empty, because we want to remove it
	current.psl = emptyBucket

	idx = (idx + 1) & uintptr(s.capMinus1)
	next := &s.buckets[idx]
	// now, back shift all buckets until we found a optimum or empty one
	for next.psl > 0 {
		next.psl--
		*current, *next = *next, *current // swap values
		current = next
		idx = (idx + 1) & uintptr(s.capMinus1)
		next = &s.buckets[idx]
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

	s := m.checkForResize()

	var (
		idx = m.hasher(key) & uintptr(s.capMinus1)
		psl = int8(0)
	)

	// search for the key
	for ; psl <= s.buckets[idx].psl; psl++ {
		if s.buckets[idx].key == key {
			old := s.buckets[idx].value
			s.buckets[idx].value = value
			return old, true // update already existing value
		}
		// next index
		idx = (idx + 1) & uintptr(s.capMinus1)
	}

	s.length.Add(1)
	newBucket := bucket[K, V]{key: key, value: value, psl: psl}
	s.emplace(&newBucket, idx)

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

	s := m.storage.Load()
	for i := range s.buckets {
		if !s.buckets[i].isEmpty() {
			if stop := f(s.buckets[i].key, s.buckets[i].value); stop {
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

	m.storage.Store(newStorage[K, V](shared.DefaultSize, m.maxLoad))
}
