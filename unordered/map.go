package unordered

import (
	"fmt"

	"github.com/EinfachAndy/hashmaps/shared"
)

type linkedList[K comparable, V any] struct {
	head *node[K, V]
}

type node[K comparable, V any] struct {
	next  *node[K, V]
	key   K
	value V
}

// Unordered is a hash map implementation, where the elements are organized into buckets
// depending on their hash values. Collisions are chained in a single linked list.
// An inserted value keeps its memory address, means a element in a bucket will not copied
// or swapped. That supports holding points instead of copy by value. see: `Insert` and `lookup`.
type Unordered[K comparable, V any] struct {
	buckets []linkedList[K, V]
	hasher  shared.HashFn[K]
	// length stores the current inserted elements
	length uintptr
	// capMinus1 is used for a bitwise AND on the hash value,
	// because the size of the underlying array is a power of two value
	capMinus1 uintptr

	nextResize uintptr
	maxLoad    float32
}

// New creates a ready to use `unordered` hash map with default settings.
func New[K comparable, V any]() *Unordered[K, V] {
	return NewWithHasher[K, V](shared.GetHasher[K]())
}

// NewWithHasher same as `NewUnordered` but with a given hash function.
func NewWithHasher[K comparable, V any](hasher shared.HashFn[K]) *Unordered[K, V] {
	m := &Unordered[K, V]{
		hasher:  hasher,
		maxLoad: shared.DefaultMaxLoad,
	}
	m.Reserve(shared.DefaultSize)

	return m
}

//go:inline
func (m *Unordered[K, V]) search(key K, idx uintptr) *V {
	for current := m.buckets[idx].head; current != nil; current = current.next {
		if current.key == key {
			return &(current.value)
		}
	}
	return nil
}

// Get returns the value stored for this key, or false if not found.
func (m *Unordered[K, V]) Get(key K) (V, bool) {
	var (
		idx = m.hasher(key) & m.capMinus1
		v   V
	)

	ptr := m.search(key, idx)
	if ptr != nil {
		return *ptr, true
	}

	return v, false
}

// Lookup returns a pointer to the stored value for this key or nil if not found.
// The pointer is valid until the key is part of the hash map.
// Note, use `Get` for small values.
func (m *Unordered[K, V]) Lookup(key K) *V {
	idx := m.hasher(key) & m.capMinus1
	return m.search(key, idx)
}

//go:inline
func (m *Unordered[K, V]) pushFront(head **node[K, V], newNode *node[K, V]) {
	newNode.next = *head
	*head = newNode
}

// Insert returns a pointer to a zero allocated value. These pointer is valid until
// the key is part of the hash map. Note, use `Put` for small values.
func (m *Unordered[K, V]) Insert(key K) (*V, bool) {
	if m.length >= m.nextResize {
		m.grow()
	}

	idx := m.hasher(key) & m.capMinus1

	ptr := m.search(key, idx)
	if ptr != nil {
		return ptr, false
	}

	m.length++
	newNode := &node[K, V]{key: key}
	m.pushFront(&(m.buckets[idx].head), newNode)

	return &newNode.value, true
}

func (m *Unordered[K, V]) resize(n uintptr) {
	m.capMinus1 = n - 1
	oldBuckets := m.buckets
	m.buckets = make([]linkedList[K, V], n)
	m.nextResize = uintptr(float32(n) * m.maxLoad)

	for i := range oldBuckets {
		for current := oldBuckets[i].head; current != nil; {
			newElem := current
			current = current.next
			newElem.next = nil // unlink from old

			// push newElem to front of the list
			newIdx := m.hasher(newElem.key) & m.capMinus1
			m.pushFront(&(m.buckets[newIdx].head), newElem)
		}
	}
}

// Clear removes all key-value pairs from the map.
func (m *Unordered[K, V]) Clear() {
	for i := range m.buckets {
		m.buckets[i].head = nil
	}

	m.length = 0
}

// Size returns the number of items in the map.
func (m *Unordered[K, V]) Size() int {
	return int(m.length)
}

// Load return the current load of the hash map.
func (m *Unordered[K, V]) Load() float32 {
	return float32(m.length) / float32(cap(m.buckets))
}

func (m *Unordered[K, V]) grow() {
	newSize := uintptr(cap(m.buckets) * 2)
	m.resize(newSize)
}

// Reserve sets the number of buckets to the most appropriate to contain at least n elements.
// If n is lower than that, the function may have no effect.
func (m *Unordered[K, V]) Reserve(n uintptr) {
	var (
		needed = uintptr(float32(n) / m.maxLoad)
		newCap = uintptr(shared.NextPowerOf2(uint64(needed)))
	)

	if uintptr(cap(m.buckets)) < newCap {
		m.resize(newCap)
	}
}

// Put maps the given key to the given value. If the key already exists its
// value will be overwritten with the new value.
// Returns true, if the element is a new item in the hash map.
//
//go:inline
func (m *Unordered[K, V]) Put(key K, val V) bool {
	v, isNew := m.Insert(key)
	*v = val

	return isNew
}

// Remove removes the specified key-value pair from the map.
// Returns true, if the element was in the hash map.
func (m *Unordered[K, V]) Remove(key K) bool {
	var (
		idx     = m.hasher(key) & m.capMinus1
		current = m.buckets[idx].head
		prev    *node[K, V]
	)

	// check head
	if current != nil && current.key == key {
		m.buckets[idx].head = current.next
		m.length--

		return true
	}

	// search for the key
	for current != nil && current.key != key {
		prev = current
		current = current.next
	}

	if current == nil {
		return false // not found
	}

	// unlink
	prev.next = current.next
	m.length--

	return true
}

// Copy returns a copy of this map.
func (m *Unordered[K, V]) Copy() *Unordered[K, V] {
	newM := &Unordered[K, V]{
		buckets:   make([]linkedList[K, V], cap(m.buckets)),
		capMinus1: m.capMinus1,
		length:    m.length,
		hasher:    m.hasher,
	}

	m.Each(func(k K, v V) bool {
		newM.Put(k, v)
		return false
	})

	return newM
}

// MaxLoad forces resizing if the ratio is reached.
// Useful values are in range [0.7-1.0].
// Returns ErrOutOfRange if `lf` is not in the open range (0.0,1.0).
func (m *Unordered[K, V]) MaxLoad(lf float32) error {
	if lf <= 0.0 {
		return fmt.Errorf("%f: %w", lf, shared.ErrOutOfRange)
	}

	m.maxLoad = lf
	m.nextResize = uintptr(float32(cap(m.buckets)) * lf)

	return nil
}

// Each calls 'fn' on every key-value pair in the hash map in no particular order.
// If 'fn' returns true, the iteration stops.
func (m *Unordered[K, V]) Each(fn func(key K, val V) bool) {
	for i := range m.buckets {
		for current := m.buckets[i].head; current != nil; current = current.next {
			if stop := fn(current.key, current.value); stop {
				// stop iteration
				return
			}
		}
	}
}
