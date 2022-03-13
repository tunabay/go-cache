# go-cache

[![Go Reference](https://pkg.go.dev/badge/github.com/tunabay/go-cache.svg)](https://pkg.go.dev/github.com/tunabay/go-cache)
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat)](LICENSE)

## Overview

Package cache implements a simple LRU cache with generics.

It provides a mechanism for caching resources that are time consuming to create
or retrieve. The main feature is that the creation process runs only once even
when multiple go-routines concurrently request for a key that does not exist in
the cache.

Originally implemented for a small web program that dynamically generates images
based on the requested URI.

## Usage

```go
import (
	"fmt"
	"sync"

	"github.com/tunabay/go-cache"
)

type MyResource struct {
	data string
}

func CreateResource(key string) (*Resource, time.Time, error) {
	time.Sleep(time.Second >> 2) // slow creation
	return &MyResource{data: "data " + key}, time.Time{}, nil
}

func main() {
	myCache := cache.New[string, *MyResource](CreateResource)

	showResource := func(key string) {
		val, cached, _, err := myCache.Get(key)
		if err != nil {
			panic(err)
		}
		var tag string
		if cached {
			tag = " (cached)"
		}
		fmt.Printf("%s -> %q%s\n", key, val.data, tag)
	}

	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(n int) {
			key := fmt.Sprintf("key-%d", n & 3)
			showResource(key)
		}(i)
	}
	wg.Wait()
}
```

## Documentation and more examples

- Read the [documentation](https://pkg.go.dev/github.com/tunabay/go-cache).

## License

go-cache is available under the MIT license. See the [LICENSE](LICENSE) file for more information.
