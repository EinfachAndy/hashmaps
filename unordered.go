package hashmaps

type linkedList[K comparable, V any] struct {
	head *listNode[K, V]
}

type listNode[K comparable, V any] struct {
	next  *listNode[K, V]
	key   K
	value V
}

// Unordered is a hash map implementation, where the elements are organized into buckets
// depending on their hash values. Collisions are chained in a single linked list.
// An inserted value keeps its memory address, means a element in a bucket will not copied
// or swapped. That supports holding points instead of copy by value. see: `Insert` and `lookup`.
type Unordered[K comparable, V any] struct {
	buckets []linkedList[K, V]
	hasher  HashFn[K]
	// length stores the current inserted elements
	length uintptr
	// capMinus1 is used for a bitwise AND on the hash value,
	// because the size of the underlying array is a power of two value
	capMinus1 uintptr
}

// NewUnordered creates a ready to use `unordered` hash map with default settings.
func NewUnordered[K comparable, V any]() *Unordered[K, V] {
	return NewUnorderedWithHasher[K, V](GetHasher[K]())
}

// NewUnorderedWithHasher same as `NewUnordered` but with a given hash function.
func NewUnorderedWithHasher[K comparable, V any](hasher HashFn[K]) *Unordered[K, V] {
	return &Unordered[K, V]{
		capMinus1: 3,
		buckets:   make([]linkedList[K, V], 4),
		hasher:    hasher,
	}
}

// Get returns the value stored for this key, or false if not found.
func (m *Unordered[K, V]) Get(key K) (V, bool) {
	idx := m.hasher(key) & m.capMinus1
	for current := m.buckets[idx].head; current != nil; current = current.next {
		if current.key == key {
			return current.value, true
		}
	}
	var v V
	return v, false
}

// Lookup returns a pointer to the stored value for this key or nil if not found.
// The pointer is valid until the key is part of the hash map.
// Note, use `Get` for small values.
func (m *Unordered[K, V]) Lookup(key K) *V {
	idx := m.hasher(key) & m.capMinus1
	for current := m.buckets[idx].head; current != nil; current = current.next {
		if current.key == key {
			return &(current.value)
		}
	}
	return nil
}

// Insert returns a pointer to a zero allocated value. These pointer is valid until
// the key is part of the hash map. Note, use `Put` for small values.
func (m *Unordered[K, V]) Insert(key K) (*V, bool) {
	if m.length >= uintptr(cap(m.buckets)) {
		m.grow()
	}

	idx := m.hasher(key) & m.capMinus1
	// check head
	if m.buckets[idx].head == nil {
		newNode := &listNode[K, V]{key: key}
		m.buckets[idx].head = newNode
		m.length++
		return &newNode.value, true
	}

	// search
	for current := m.buckets[idx].head; ; current = current.next {
		if current.key == key {
			return &current.value, false
		}
		// reached end of list, so insert
		if current.next == nil {
			m.length++
			newNode := &listNode[K, V]{key: key}
			current.next = newNode
			return &newNode.value, true
		}
	}
}

func (m *Unordered[K, V]) rehash(n uintptr) {
	m.capMinus1 = n - 1
	newBuckets := make([]linkedList[K, V], n)
	for _, l := range m.buckets {
		for current := l.head; current != nil; {
			newElem := current
			current = current.next
			newElem.next = nil // unlink from old

			// push newElem to front of the list
			newIdx := m.hasher(newElem.key) & m.capMinus1
			head := newBuckets[newIdx].head
			headRef := &newBuckets[newIdx].head
			*headRef = newElem
			newElem.next = head
		}
	}
	m.buckets = newBuckets
}

// Clear removes all key-value pairs from the map.
func (m *Unordered[K, V]) Clear() {
	for idx := range m.buckets {
		m.buckets[idx].head = nil
	}
	m.length = 0
}

// Size returns the number of items in the map.
func (m *Unordered[K, V]) Size() int {
	return int(m.length)
}

// Load return the current load of the hash map.
func (m *Unordered[K, V]) Load() float32 {
	return float32(m.length) / float32(len(m.buckets))
}

func (m *Unordered[K, V]) grow() {
	newSize := uintptr(cap(m.buckets) * 2)
	m.rehash(newSize)
}

// Reserve sets the number of buckets to the most appropriate to contain at least n elements.
// If n is lower than that, the function may have no effect.
func (m *Unordered[K, V]) Reserve(n uintptr) {
	newCap := uintptr(NextPowerOf2(uint64(n)))
	if uintptr(cap(m.buckets)) < newCap {
		m.rehash(newCap)
	}
}

// Put maps the given key to the given value. If the key already exists its
// value will be overwritten with the new value.
// Returns true, if the element is a new item in the hash map.
// go:inline
func (m *Unordered[K, V]) Put(key K, val V) bool {
	v, isNew := m.Insert(key)
	*v = val
	return isNew
}

// Remove removes the specified key-value pair from the map.
// Returns true, if the element was in the hash map.
func (m *Unordered[K, V]) Remove(key K) bool {
	var (
		idx                     = m.hasher(key) & m.capMinus1
		current                 = m.buckets[idx].head
		prev    *listNode[K, V] = nil
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

// Each calls 'fn' on every key-value pair in the hash map in no particular order.
// If 'fn' returns true, the iteration stops.
func (m *Unordered[K, V]) Each(fn func(key K, val V) bool) {
	for _, l := range m.buckets {
		for current := l.head; current != nil; current = current.next {
			if stop := fn(current.key, current.value); stop {
				// stop iteration
				return
			}
		}
	}
}
