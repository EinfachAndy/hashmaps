package clrobin

import (
	"sync/atomic"

	"github.com/EinfachAndy/hashmaps/shared"
)

// CLRobin is a concurrent locked robin hood hashmap.
type CLRobin[K comparable, V comparable] struct {
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
func (m *CLRobin[K, V]) checkForResize() *storage[K, V] {
	// next 3 lines must be atomic without mutex
	s := m.storage.Load()
	if int(s.length.Load()) >= s.nextResize {
		return m.resize((s.capMinus1 + 1) * 2)
	}

	return s
}

//go:inline
func (m *CLRobin[K, V]) resize(n int) *storage[K, V] {
	old := m.storage.Load()
	new := newStorage[K, V](n, m.maxLoad)
	new.length.Store(old.length.Load())

	for i := range old.buckets {
		ob := &old.buckets[i]
		ob.Lock()
		if !ob.b.isEmpty() {
			cpy := ob.b
			cpy.psl = 0
			start := m.hasher(cpy.key) & uintptr(new.capMinus1)
			new.buckets[start].Lock()
			end := new.emplace(&cpy, start)
			new.unlock(start, end)
		}
		ob.Unlock()
	}

	m.storage.Store(new)

	return new
}

// Reserve sets the number of buckets to the most appropriate to contain at least n elements.
// If n is lower than that, the function may have no effect.
func (m *CLRobin[K, V]) Reserve(n uintptr) {
	var (
		needed = int(float32(n) / m.maxLoad)
		newCap = int(shared.NextPowerOf2(uint64(needed)))
	)

	if cap(m.storage.Load().buckets) < newCap {
		m.resize(newCap)
	}
}

func (m *CLRobin[K, V]) Size() int {
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
	var (
		s     = m.storage.Load()
		start = m.hasher(key) & uintptr(s.capMinus1)
		v     V
	)

	b, idx, _ := s.search(start, key)
	defer s.unlock(start, idx)
	if b != nil {
		v = b.value
		return v, true
	}

	return v, false
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (m *CLRobin[K, V]) LoadAndDelete(key K) (V, bool) {
	var (
		s     = m.storage.Load()
		start = m.hasher(key) & uintptr(s.capMinus1)
		v     V
	)

	current, idx, _ := s.search(start, key)
	defer s.unlock(start, idx)
	if current == nil {
		return v, false
	}
	v = current.value

	s.remove(idx)

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
	var (
		s     = m.checkForResize()
		start = m.hasher(key) & uintptr(s.capMinus1)
	)

	b, idx, psl := s.search(start, key)
	if b != nil {
		old := b.value
		b.value = value
		s.unlock(start, idx)
		return old, true // update already existing value
	}

	s.length.Add(1)
	newBucket := bucket[K, V]{key: key, value: value, psl: psl}
	idx = s.emplace(&newBucket, idx)
	s.unlock(start, idx)

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
	s := m.storage.Load()
	for i := range s.buckets {
		b := &s.buckets[i].b
		if !b.isEmpty() {
			if stop := f(b.key, b.value); stop {
				// stop iteration
				return
			}
		}
	}
}

// Clear deletes all the entries, resulting in an empty Map.
func (m *CLRobin[K, V]) Clear() {
	m.storage.Store(newStorage[K, V](shared.DefaultSize, m.maxLoad))
}
