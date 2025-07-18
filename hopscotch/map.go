package hopscotch

import (
	"fmt"

	"github.com/EinfachAndy/hashmaps/shared"
)

// Hopscotch is a hashmap implementation which uses open addressing,
// where collisions are managed within a limited neighborhood. That is
// implemented as a dynamically growing bitmap with a default
// size of 4 and a upper bound of 63. From this it follows a constant
// lookup time for the Get function. To achieve this invariant
// linear probing is used for finding an empty slot in the hashmap,
// if the next empty slot is not within the size of the neighborhood,
// subsequent swap of closer buckets are done or the size of the
// neighborhood is increased.
type Hopscotch[K comparable, V any] struct {
	buckets []bucket[K, V]
	hasher  shared.HashFn[K]
	// length stores the current inserted elements
	length uintptr
	// capMinus1 is used for a bitwise AND on the hash value,
	// because the size of the underlying array is a power of two value
	capMinus1        uintptr
	neighborhoodSize uintptr
	nextResize       uintptr
	maxLoad          float32
}

// New creates a ready to use `Hopscotch` hashmap with default settings.
func New[K comparable, V any]() *Hopscotch[K, V] {
	return NewWithHasher[K, V](shared.GetHasher[K]())
}

// NewWithHasher same as `NewHopscotch` but with a given hash function.
func NewWithHasher[K comparable, V any](hasher shared.HashFn[K]) *Hopscotch[K, V] {
	const (
		DefaultNeighborhoodSize = 4 // must be pow of 2
	)

	m := &Hopscotch[K, V]{
		hasher:           hasher,
		neighborhoodSize: DefaultNeighborhoodSize,
		maxLoad:          shared.DefaultMaxLoad,
	}

	m.Reserve(shared.DefaultSize)

	return m
}

// grow doubles the size size of the hashmap.
//
//go:inline
func (m *Hopscotch[K, V]) grow() {
	m.resize(2 * (m.capMinus1 + 1))
}

func (m *Hopscotch[K, V]) resize(n uintptr) {
	nmap := Hopscotch[K, V]{
		buckets:          make([]bucket[K, V], n+m.neighborhoodSize),
		hasher:           m.hasher,
		length:           m.length,
		capMinus1:        n - 1,
		neighborhoodSize: m.neighborhoodSize,
		maxLoad:          m.maxLoad,
		nextResize:       uintptr(float32(n) * m.maxLoad),
	}

	for i := range m.buckets {
		if !m.buckets[i].isEmpty() {
			homeIdx := nmap.hasher(m.buckets[i].key) & nmap.capMinus1
			nmap.emplace(m.buckets[i].key, m.buckets[i].val, homeIdx)
		}
	}

	// update current map
	m.buckets = nmap.buckets
	m.capMinus1 = nmap.capMinus1
	m.nextResize = nmap.nextResize
	m.neighborhoodSize = nmap.neighborhoodSize
}

// Reserve sets the number of buckets to the most appropriate to contain at least n elements.
// If n is lower than that, the function may have no effect.
func (m *Hopscotch[K, V]) Reserve(n uintptr) {
	var (
		needed = uintptr(float32(n) / m.maxLoad)
		newCap = uintptr(shared.NextPowerOf2(uint64(needed)))
	)

	if uintptr(cap(m.buckets)) < newCap {
		m.resize(newCap)
	}
}

// search looks within the neighborhood of the home bucket to find the desired key.
// This function has a constant runtime.
//
//go:inline
func (m *Hopscotch[K, V]) search(homeIdx uintptr, key K) (uintptr, bool) {
	neighborhood := m.buckets[homeIdx].getNeighborhood()
	for neighborhood != 0 {
		if (neighborhood & 1) == 1 {
			if m.buckets[homeIdx].key == key {
				return homeIdx, true
			}
		}

		homeIdx++

		neighborhood >>= 1
	}

	return 0, false
}

// Get returns the value stored for this key, or false if there is no such value.
func (m *Hopscotch[K, V]) Get(key K) (V, bool) {
	var (
		homeIdx    = m.hasher(key) & m.capMinus1
		idx, found = m.search(homeIdx, key)
		v          V
	)

	if found {
		return m.buckets[idx].val, true
	}

	return v, false
}

// moveCloser tries to achieve the neighborhood invariant by moving
// the given empty bucket closer to its home bucket. Therefore another
// buckets are moved more far-off. The parameter `emptyIdx`
// is a in-out variable, that is updated, if the movement was successful.
//
//go:inline
func (m *Hopscotch[K, V]) moveCloser(emptyIdx *uintptr) bool {
	start := *emptyIdx - (m.neighborhoodSize - 1)

	for homeIdx := start; homeIdx < *emptyIdx; homeIdx++ {
		neighborhood := m.buckets[homeIdx].getNeighborhood()
		for cIdx := homeIdx; neighborhood != 0 && cIdx < *emptyIdx; cIdx++ {
			if (neighborhood & 1) == 1 {
				distance := cIdx - homeIdx
				// found a candidate, mark it as empty
				m.buckets[cIdx].release()

				// move the candidate to the empty bucket
				m.buckets[*emptyIdx].occupy()
				m.buckets[*emptyIdx].key = m.buckets[cIdx].key
				m.buckets[*emptyIdx].val = m.buckets[cIdx].val

				// update the neighborhood of the home bucket,
				// because we moved the empty bucket closer
				m.buckets[homeIdx].set(distance, false)
				m.buckets[homeIdx].set(*emptyIdx-homeIdx, true)

				// announce the new empty index
				*emptyIdx = cIdx

				return true
			}

			neighborhood >>= 1
		}
	}

	return false
}

// increaseNeighborhood returns true if the Neighborhood could be increased.
//
//go:inline
func (m *Hopscotch[K, V]) increaseNeighborhood() bool {
	// move closer does not work, we need to find another solution!
	const lastPow2 = 32
	if m.neighborhoodSize < lastPow2 {
		m.neighborhoodSize = 2 * m.neighborhoodSize
		return true
	}
	if m.neighborhoodSize == lastPow2 {
		m.neighborhoodSize = maxNeighborhoodSize
		return true
	}

	return false
}

// emplace adds the key-value pair to the hashmap. It does not check
// the occurrence, so it expects that the give key is not already
// in. Furthermore a resize or rehash can happen to achieve
// the neighborhood invariant.
func (m *Hopscotch[K, V]) emplace(key K, val V, homeIdx uintptr) {
START:
	emptyIdx := homeIdx

	// linear probing for the next empty bucket
	for ; ; emptyIdx++ {
		if emptyIdx == uintptr(cap(m.buckets)) {
			// we reached the end of the bucket array, so we need to resize it
			m.grow()
			goto EMPLACE_AFTER_REHASH
		}

		if m.buckets[emptyIdx].isEmpty() {
			// we found a empty bucket for the next insert, we are done
			break
		}
	}

	// Try to emplace the key-value pair.
	// If the distance is outer the size of the
	// neighborhood, move another bucket closer if possible
	for {
		distance := emptyIdx - homeIdx
		if distance < m.neighborhoodSize {
			// we found an empty bucket within the neighborhood.
			// we are finished and can emplace the key-value pair.
			m.buckets[emptyIdx].occupy()
			m.buckets[emptyIdx].key = key
			m.buckets[emptyIdx].val = val
			m.buckets[homeIdx].set(distance, true)

			return
		}

		// try to move the another bucket closer, so that it is within the
		// neighborhood size of the home bucket.
		if !m.moveCloser(&emptyIdx) {
			break
		}
	}

	// move closer does not work, we need to find another solution!
	if !m.increaseNeighborhood() {
		// that is the last hope to achieve the neighborhood invariant,
		// but this case should happen really rare.
		// Note: it is also possible to change the hash function here!
		m.grow()
	}

EMPLACE_AFTER_REHASH:
	homeIdx = m.hasher(key) & m.capMinus1
	goto START
}

// Put adds the given key-value pair to the hashmap. If the key already exists its
// value will be overwritten with the new value.
// Returns true, if the element is a new item in the hashmap.
func (m *Hopscotch[K, V]) Put(key K, val V) bool {
	// check for resize
	if m.length >= m.nextResize {
		m.grow()
	}

	var (
		homeIdx    = m.hasher(key) & m.capMinus1
		idx, found = m.search(homeIdx, key)
	)

	if found {
		// already inserted, update
		m.buckets[idx].val = val
		return false
	}

	// emplace new key-value pair
	m.length++
	m.emplace(key, val, homeIdx)

	return true
}

// Remove removes the specified key-value pair from the hashmap.
// Returns true, if the element was in the hashmap.
func (m *Hopscotch[K, V]) Remove(key K) bool {
	var (
		homeIdx    = m.hasher(key) & m.capMinus1
		idx, found = m.search(homeIdx, key)
	)

	if !found {
		return false
	}

	distance := idx - homeIdx

	m.buckets[homeIdx].set(distance, false)
	m.buckets[idx].release()
	m.length--

	return true
}

// Clear removes all key-value pairs from the hashmap.
func (m *Hopscotch[K, V]) Clear() {
	for i := range m.buckets {
		m.buckets[i].hopInfo = 0
	}

	m.length = 0
}

// MaxLoad forces resizing if the ratio is reached.
// Useful values are in range [0.5-0.9].
// Returns ErrOutOfRange if `lf` is not in the open range (0.0,1.0).
func (m *Hopscotch[K, V]) MaxLoad(lf float32) error {
	if lf <= 0.0 || lf >= 1.0 {
		return fmt.Errorf("%f: %w", lf, shared.ErrOutOfRange)
	}

	m.maxLoad = lf
	m.nextResize = uintptr(float32(cap(m.buckets)) * lf)

	return nil
}

// Load return the current load of the hashmap.
func (m *Hopscotch[K, V]) Load() float32 {
	return float32(m.length) / float32(cap(m.buckets))
}

// Size returns the number of items in the hashmap.
func (m *Hopscotch[K, V]) Size() int {
	return int(m.length)
}

// Copy returns a copy of this hashmap.
func (m *Hopscotch[K, V]) Copy() *Hopscotch[K, V] {
	newM := &Hopscotch[K, V]{
		buckets:          make([]bucket[K, V], cap(m.buckets)),
		capMinus1:        m.capMinus1,
		length:           m.length,
		hasher:           m.hasher,
		neighborhoodSize: m.neighborhoodSize,
		maxLoad:          m.maxLoad,
		nextResize:       m.nextResize,
	}

	copy(newM.buckets, m.buckets)

	return newM
}

// Each calls 'fn' on every key-value pair in the hash map in no particular order.
// If 'fn' returns true, the iteration stops.
func (m *Hopscotch[K, V]) Each(fn func(key K, val V) bool) {
	for i := range m.buckets {
		if !m.buckets[i].isEmpty() {
			if stop := fn(m.buckets[i].key, m.buckets[i].val); stop {
				// stop iteration
				return
			}
		}
	}
}
