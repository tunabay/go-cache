// Copyright (c) 2022 Hirotsuna Mizuno. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package cache

import (
	"sync"
	"time"
)

// Cache is a simple LRU cache implementation for cacheable object creation.
type Cache[K comparable, V any] struct {
	create     CreateFunc[K, V]
	remove     RemoveFunc[K, V]
	maxItems   int
	maxAge     time.Duration
	entries    map[K]*entry[K, V]
	head, tail entry[K, V]
	mu         sync.Mutex
}

// entry is the container node that holds a cache entry.
type entry[K comparable, V any] struct {
	key        K
	val        V
	deadline   time.Time // zero value for no deadline
	prev, next *entry[K, V]
	created    bool
	deleted    bool
	err        error
	cond       *sync.Cond
}

// Config is the config parameer set which is passed to NewWithConfig.
type Config[K comparable, V any] struct {
	// CreateFunc is the function to create a new cacheable object. It is
	// called when Get is called for a key that does not exist in the cache.
	CreateFunc CreateFunc[K, V]

	// RemoveFunc is the optional function that is called immediately after
	// a cache entry is removed from the cache.
	RemoveFunc RemoveFunc[K, V]

	// MaxItems is the maximum number of items that the cache can hold.
	// 0 indicates unlimited.
	MaxItems int

	// MaxAge is the maximum time since an item was created and cached.
	// 0 indicates unlimited.
	MaxAge time.Duration
}

// CreateFunc represents a function for object creation. It will be called when
// Get is called for a key that does not exist in the cache. It should return
// the created value and optionally a deadline. Use time.Time{} as deadline
// when no deadline is specified.
type CreateFunc[K comparable, V any] func(K) (V, time.Time, error)

// RemoveFunc represents a function to be called when an item is removed from
// the cache. It can be used to free resources such as files held by the value.
// This is called inside locks, so it is recommended to return immediately.
// It's better to start a go routine to process with files or databases.
// Note that if Get is called for the same key immediately after removal, a new
// value for the same key may be concurrently created and cached.
type RemoveFunc[K comparable, V any] func(K, V)

// New creates a Cache with the creation func create. It uses the default
// configuration with MaxItems = 1024. To specify other configuration, use
// NewWithConfig instead.
func New[K comparable, V any](create CreateFunc[K, V]) *Cache[K, V] {
	conf := &Config[K, V]{
		CreateFunc: create,
		MaxItems:   1024,
	}
	return NewWithConfig[K, V](conf)
}

// NewWithConfig creates a Cache with specified configuration.
func NewWithConfig[K comparable, V any](conf *Config[K, V]) *Cache[K, V] {
	c := &Cache[K, V]{
		create:   conf.CreateFunc,
		remove:   conf.RemoveFunc,
		maxItems: conf.MaxItems,
		maxAge:   conf.MaxAge,
		entries:  make(map[K]*entry[K, V]),
	}
	c.head.next, c.tail.prev = &c.tail, &c.head
	if conf != nil {
		c.maxAge = conf.MaxAge
		c.maxItems = conf.MaxItems
	}

	return c
}

// CheckAndExpire checks all the items in the cache and removes expired items.
// If only MaxItems is used and MaxAge or deadline is not used, calling this
// method has no effect.
// If MaxAge or the second return value of CreateFunc is used, it is better to
// call this method periodically to remove expired cache items.
func (c *Cache[K, V]) CheckAndExpire() {
	c.mu.Lock()
	for key, item := range c.entries {
		item.cond.L.Lock()
		if !item.created || item.deleted || item.deadline.IsZero() || item.deadline.After(time.Now()) {
			item.cond.L.Unlock()
			continue
		}
		item.prev.next, item.next.prev = item.next, item.prev
		delete(c.entries, key)
		item.deleted = true
		if c.remove != nil {
			c.remove(key, item.val)
		}
	}
	c.mu.Unlock()
}

// Get gets the value for the key from the cache. If it does not exist in the
// cache, call CreateFunc to create the new value.
// It returns the value, cached or not, the cache expiration time.
// Note that the cache expiration time 0 represents that it never expired,
// not uncachable.
func (c *Cache[K, V]) Get(key K) (V, bool, time.Time, error) {
	var wait bool

	c.mu.Lock()
	item, found := c.entries[key]
	if found {
		item.cond.L.Lock()
		if item.created {
			if item.deadline.IsZero() || item.deadline.After(time.Now()) {
				if c.head.next != item {
					item.prev.next, item.next.prev = item.next, item.prev
					item.next, c.head.next.prev = c.head.next, item
					c.head.next, item.prev = item, &c.head
				}
				item.cond.L.Unlock()
				c.mu.Unlock()

				return item.val, true, item.deadline, nil
			}

			// expired
			item.prev.next, item.next.prev = item.next, item.prev
			delete(c.entries, key)
			item.deleted = true
			if c.remove != nil {
				c.remove(key, item.val)
			}
			found = false
		} else {
			wait = true
		}
		item.cond.L.Unlock()
	}
	if !found {
		item = &entry[K, V]{
			cond: sync.NewCond(&sync.Mutex{}),
		}
		c.entries[key] = item
		if c.maxItems != 0 && c.maxItems < len(c.entries) {
			last := c.tail.prev
			last.prev.next, c.tail.prev = &c.tail, last.prev
			delete(c.entries, last.key)
			last.deleted = true
			if c.remove != nil {
				c.remove(last.key, last.val)
			}
		}
	}
	c.mu.Unlock()

	if wait {
		item.cond.L.Lock()
		for !item.created && !item.deleted {
			item.cond.Wait()
		}
		item.cond.L.Unlock()

		return item.val, true, item.deadline, item.err
	}

	val, deadline, err := c.create(key)
	if err != nil {
		cerr := &CreationError[K]{Key: key, Err: err}
		c.mu.Lock()
		item.cond.L.Lock()
		item.err = cerr
		delete(c.entries, key)
		item.deleted = true
		item.cond.Broadcast()
		item.cond.L.Unlock()
		c.mu.Unlock()

		return val, false, time.Time{}, cerr
	}

	c.mu.Lock()
	item.cond.L.Lock()
	item.val = val
	item.deadline = deadline
	if c.maxAge != 0 {
		expire := time.Now().Add(c.maxAge)
		if item.deadline.IsZero() || expire.Before(item.deadline) {
			item.deadline = expire
		}
	}
	item.created = true
	item.next, c.head.next.prev = c.head.next, item
	c.head.next, item.prev = item, &c.head
	item.cond.Broadcast()
	item.cond.L.Unlock()
	c.mu.Unlock()

	return item.val, false, item.deadline, nil
}
