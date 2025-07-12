// Package hashmaps implements several types of hash tables.
package hashmaps

import (
	"github.com/EinfachAndy/hashmaps/flat"
	"github.com/EinfachAndy/hashmaps/hopscotch"
	"github.com/EinfachAndy/hashmaps/robin"
	"github.com/EinfachAndy/hashmaps/shared"
	"github.com/EinfachAndy/hashmaps/unordered"
)

// HashMap collects the basic hash maps operations as function points.
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

type Type int

const (
	Hopscotch      = 0
	Robin     Type = 1
	Unordered      = 2
	Flat           = 3
)

type Config[K comparable, V any] struct {
	Type    Type
	Size    uintptr
	MaxLoad float32
	Hasher  shared.HashFn[K]
	Empty   K
}

// Factory is a generic function to instantiate different kind of
// hash map implementations.
func Factory[K comparable, V any](cfg Config[K, V]) *HashMap[K, V] {
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
		res.Put = m.Put
		res.Remove = m.Remove
		res.Reserve = m.Reserve
		res.Size = m.Size
	}

	if cfg.MaxLoad > 0 {
		res.MaxLoad(cfg.MaxLoad)
	}

	if cfg.Size > 0 {
		res.Reserve(cfg.Size)
	}

	return res
}
