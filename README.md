# Golang Hash Maps

[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/EinfachAndy/hashmaps/blob/main/LICENSE)

This package collects several hash maps. The first one is a robin hood hash map implementation.

# Getting Started

```bash
go get -u github.com/EinfachAndy/hashmaps
```

## Robin Hood Hash Map

The example code can be try out [here](https://go.dev/play/p/ZeKzsiGXlh7).

```go
package main

import (
	"fmt"

	"github.com/EinfachAndy/hashmaps"
)

func main() {
	m := hashmaps.NewRobinHood[int, int]()

	// insert some elements
	m.Put(2, 2)
	isNew := m.Put(1, 5)
	if !isNew {
		panic("broken")
	}

	// lookup a key
	val, found := m.Get(1)
	if !found || val != 5 {
		panic("broken")
	}

	// iterate the map
	m.Each(func(key int, val int) bool {
		fmt.Println(key, "->", val)
		return false
	})

	// Print some metrics
	fmt.Println("Size:", m.Size(), "load:", m.Load())

	// remove keys
	wasIn := m.Remove(1)
	if !wasIn {
		panic("broken")
	}
}
```

# Benchmarks

The benchmarks are implemented and maintained [here](https://github.com/EinfachAndy/bench-hashmaps).

# Contributing

If you would like to contribute a new feature or maps, please let me know first what
you would like to add (via email or issue tracker).
