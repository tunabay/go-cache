// Copyright (c) 2022 Hirotsuna Mizuno. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package cache_test

import (
	"fmt"
	"sync"
	"time"

	"github.com/tunabay/go-cache"
)

func Example_simple() {
	// function to create a resource
	createResource := func(key uint8) (string, time.Time, error) {
		time.Sleep(time.Second >> 2) // slow creation
		return fmt.Sprintf("resource %d", key), time.Time{}, nil
	}

	// cache with uint8 key and string value
	myCache := cache.New[uint8, string](createResource)

	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			key := uint8(n & 0b11)
			val, cached, _, err := myCache.Get(key)
			if err != nil {
				panic(err)
			}
			var tag string
			if cached {
				tag = " (cached)"
			}
			fmt.Printf("%d -> %q%s\n", key, val, tag)
		}(i)
	}
	wg.Wait()

	// Unordered output:
	// 0 -> "resource 0"
	// 1 -> "resource 1"
	// 2 -> "resource 2"
	// 3 -> "resource 3"
	// 0 -> "resource 0" (cached)
	// 1 -> "resource 1" (cached)
	// 2 -> "resource 2" (cached)
	// 3 -> "resource 3" (cached)
	// 0 -> "resource 0" (cached)
	// 1 -> "resource 1" (cached)
	// 2 -> "resource 2" (cached)
	// 3 -> "resource 3" (cached)
}

func Example_struct() {
	type myResource struct{ data string }

	createResource := func(key string) (*myResource, time.Time, error) {
		time.Sleep(time.Second >> 2) // slow creation
		return &myResource{data: "resource " + key}, time.Time{}, nil
	}

	// cache with string key and *myResource value
	myCache := cache.New[string, *myResource](createResource)

	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			key := fmt.Sprintf("key-%d", n&0b11)
			val, cached, _, err := myCache.Get(key)
			if err != nil {
				panic(err)
			}
			var tag string
			if cached {
				tag = " (cached)"
			}
			fmt.Printf("%s -> %q%s\n", key, val.data, tag)
		}(i)
	}
	wg.Wait()

	// Unordered output:
	// key-0 -> "resource key-0"
	// key-1 -> "resource key-1"
	// key-2 -> "resource key-2"
	// key-3 -> "resource key-3"
	// key-0 -> "resource key-0" (cached)
	// key-1 -> "resource key-1" (cached)
	// key-2 -> "resource key-2" (cached)
	// key-3 -> "resource key-3" (cached)
	// key-0 -> "resource key-0" (cached)
	// key-1 -> "resource key-1" (cached)
	// key-2 -> "resource key-2" (cached)
	// key-3 -> "resource key-3" (cached)
}
