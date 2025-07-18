// Package hashmaps collects several types of hashmaps.
package hashmaps

import (
	"github.com/EinfachAndy/hashmaps/flat"
	"github.com/EinfachAndy/hashmaps/hopscotch"
	"github.com/EinfachAndy/hashmaps/robin"
	"github.com/EinfachAndy/hashmaps/shared"
	"github.com/EinfachAndy/hashmaps/unordered"
)

// HashMap is the basic hashmap interface as a set of function points.
type HashMap[K comparable, V any] struct {
	Get     func(key K) (V, bool)
	Put     func(key K, val V) bool
	Remove  func(key K) bool
	Reserve func(n uintptr)
	Load    func() float32
	Clear   func()
	Size    func() int
	Each    func(fn func(key K, val V) bool)
	MaxLoad func(lf float32) error
}

// Type specified the type of the hashmap.
type Type int

const (
	Hopscotch Type = 0
	Robin     Type = 1
	Unordered Type = 2
	Flat      Type = 3
)

// Config is used by the factory to create and configure a hashmap instance.
type Config[K comparable, V any] struct {
	Type Type
	// Size grows the hashmap to the desired size.
	// If unset `DefaultSize` is used.
	Size uintptr
	// MaxLoad changes the load factor of the hashmap.
	// This value is a trade-off between performance and memory consumption.
	// If unset `DefaultMaxLoad` is used.
	MaxLoad float32
	// Hasher that is used. Must be configured for complex data types or slices.
	// If unset a default hasher is used for golang basic types.
	Hasher shared.HashFn[K]
	// Empty is used by some hash hashmap implementations e.g.: flat hashmap
	// to track empty buckets.
	Empty K
}

// MustNewHashMap same as 'NewHashMap' but panics if and only if an error occurs.
func MustNewHashMap[K comparable, V any](cfg Config[K, V]) *HashMap[K, V] {
	m, err := NewHashMap(cfg)
	if err != nil {
		panic(err.Error())
	}
	return m
}

// NewHashMap is a factory function to instantiate different kind of generic
// hashmap implementations. A struct with function pointers is used as
// interface. In most cases the usage of the dedicate hashmap type is recommended.
func NewHashMap[K comparable, V any](cfg Config[K, V]) (*HashMap[K, V], error) {
	if cfg.Hasher == nil {
		cfg.Hasher = shared.GetHasher[K]()
	}

	res := &HashMap[K, V]{}

	switch cfg.Type {
	case Hopscotch:
		m := hopscotch.NewWithHasher[K, V](cfg.Hasher)
		res.Clear = m.Clear
		res.Each = m.Each
		res.Get = m.Get
		res.Load = m.Load
		res.MaxLoad = m.MaxLoad
		res.Put = m.Put
		res.Remove = m.Remove
		res.Reserve = m.Reserve
		res.Size = m.Size
	case Robin:
		m := robin.NewWithHasher[K, V](cfg.Hasher)
		res.Clear = m.Clear
		res.Each = m.Each
		res.Get = m.Get
		res.Load = m.Load
		res.MaxLoad = m.MaxLoad
		res.Put = m.Put
		res.Remove = m.Remove
		res.Reserve = m.Reserve
		res.Size = m.Size
	case Unordered:
		m := unordered.NewWithHasher[K, V](cfg.Hasher)
		res.Clear = m.Clear
		res.Each = m.Each
		res.Get = m.Get
		res.Load = m.Load
		res.MaxLoad = m.MaxLoad
		res.Put = m.Put
		res.Remove = m.Remove
		res.Reserve = m.Reserve
		res.Size = m.Size
	case Flat:
		m := flat.NewWithHasher[K, V](cfg.Empty, cfg.Hasher)
		res.Clear = m.Clear
		res.Each = m.Each
		res.Get = m.Get
		res.Load = m.Load
		res.MaxLoad = m.MaxLoad
		res.Put = m.Put
		res.Remove = m.Remove
		res.Reserve = m.Reserve
		res.Size = m.Size
	}

	if cfg.MaxLoad > 0 {
		if err := res.MaxLoad(cfg.MaxLoad); err != nil {
			return nil, err
		}
	}

	if cfg.Size > 0 {
		res.Reserve(cfg.Size)
	}

	return res, nil
}
