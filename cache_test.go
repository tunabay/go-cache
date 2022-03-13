// Copyright (c) 2022 Hirotsuna Mizuno. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package cache_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/tunabay/go-cache"
)

func TestGet_1(t *testing.T) {
	cfunc := func(key string) (string, time.Time, error) {
		time.Sleep(time.Second >> 2) // slow creation
		if key == "KEY-0" {
			return "", time.Time{}, fmt.Errorf("test error")
		}
		return fmt.Sprintf("VALUE(%s)", key), time.Now().Add(time.Second >> 2), nil
	}
	c := cache.New[string, string](cfunc)

	lookup := func(keyNo int) {
		key := fmt.Sprintf("KEY-%d", keyNo%4)
		val, cached, _, err := c.Get(key)
		var tag string
		if cached {
			tag = " (cached)"
		}
		if err != nil {
			t.Logf("%q -> ERROR: %v%s", key, err, tag)
			return
		}
		t.Logf("%q -> %q%s", key, val, tag)
	}

	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			lookup(n % 4)
		}(i)
	}
	wg.Wait()

	c.CheckAndExpire()

	lookup(0)
	lookup(1)
	lookup(2)
	lookup(3)
	lookup(3)

	c.CheckAndExpire()
}
