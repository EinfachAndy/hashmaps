# Golang hashmaps

[![GoDoc](https://pkg.go.dev/badge/github.com/EinfachAndy/hashmaps.svg)](https://pkg.go.dev/github.com/EinfachAndy/hashmaps)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/EinfachAndy/hashmaps/blob/main/LICENSE)

This package collects several hashmap implementations:

* `Unordered` hashmap is a classic hashmap with separate chaining in a single linked list per bucket to handle collisions.
* `Robin Hood` hashmap is an open addressing hashmap with robin hood hashing and back shifting.
* `Hopscotch` hashmap is an open addressing hashmap with worst case constant runtime for lookup and delete operations.
* `Flat` hashmap is an open addressing hashmap with linear probing. 

# Getting started

```bash
go get -u github.com/EinfachAndy/hashmaps
```

## Example usage

```go
package main

import (
	"fmt"

	"github.com/EinfachAndy/hashmaps/hopscotch"
)

func main() {
	m := hopscotch.New[int, int]()
	m.Reserve(100)
	m.Put(1, 1)
	fmt.Println(m.Get(1))
	m.Remove(1)
	fmt.Println(m.Get(1))

	// Output:
	// 1 true
	// 0 false
}

```

# Benchmarks

The benchmarks are implemented and maintained [here](https://github.com/EinfachAndy/bench-hashmaps).

# Contributing

If you would like to contribute a new feature or hashmap. please let me know first what
you would like to add (via email or issue tracker).
