package cache

import (
	"sync"
	"time"
)

var zeroTTL = time.Duration(0)

type MemoryCacheTTL struct {
	// Mutex is used for handling the concurrent
	// read/write requests for cache
	sync.Mutex

	// cache holds the cache data
	cache *MemoryCacheNoTS

	// setAts holds the time that related item's set at
	setAts map[string]time.Time

	// ttl is a duration for a cache key to expire
	ttl time.Duration

	// gcInterval is a duration for garbage collection
	gcInterval time.Duration
}

// NewMemoryCache creates an inmemory cache system
// Which everytime will return the true values about a cache hit
// and never will leak memory
// ttl is used for expiration of a key from cache
func NewMemoryWithTTL(ttl time.Duration) *MemoryCacheTTL {
	return &MemoryCacheTTL{
		cache:  NewMemoryNoTS(),
		setAts: map[string]time.Time{},
		ttl:    ttl,
	}
}

// StartGC starts the garbage collection process in a go routine
func (r *MemoryCacheTTL) StartGC(gcInterval time.Duration) {
	r.gcInterval = gcInterval
	go func() {
		for _ = range time.Tick(gcInterval) {
			for key, _ := range r.cache.items {
				if !r.isValid(key) {
					r.Delete(key)
				}
			}
		}
	}()
}

// Get returns a value of a given key if it exists
// and valid for the time being
func (r *MemoryCacheTTL) Get(key string) (interface{}, error) {
	r.Lock()
	defer r.Unlock()

	if !r.isValid(key) {
		r.delete(key)
		return nil, ErrNotFound
	}

	value, err := r.cache.Get(key)
	if err != nil {
		return nil, err
	}

	return value, nil
}

// Set will persist a value to the cache or
// override existing one with the new one
func (r *MemoryCacheTTL) Set(key string, value interface{}) error {
	r.Lock()
	defer r.Unlock()

	r.cache.Set(key, value)
	r.setAts[key] = time.Now()
	return nil
}

// Delete deletes a given key if exists
func (r *MemoryCacheTTL) Delete(key string) error {
	r.Lock()
	defer r.Unlock()

	r.delete(key)
	return nil
}

func (r *MemoryCacheTTL) delete(key string) {
	r.cache.Delete(key)
	delete(r.setAts, key)
}

func (r *MemoryCacheTTL) isValid(key string) bool {
	setAt, ok := r.setAts[key]
	if !ok {
		return false
	}

	if r.ttl == zeroTTL {
		return true
	}

	return setAt.Add(r.ttl).After(time.Now())
}
