package engine

import (
	"sync"
	"time"
)

// ShardedLock provides a sharded lock to reduce contention.
// Uses multiple mutexes to distribute lock pressure across shards.
type ShardedLock struct {
	shards []*sync.RWMutex
	count  uint32
}

// NewShardedLock creates a new sharded lock with specified number of shards.
func NewShardedLock(shardCount int) *ShardedLock {
	if shardCount <= 0 {
		shardCount = 16
	}

	shards := make([]*sync.RWMutex, shardCount)
	for i := range shards {
		shards[i] = &sync.RWMutex{}
	}

	return &ShardedLock{
		shards: shards,
		count:  uint32(shardCount),
	}
}

// getShard returns the shard index for a given key.
func (sl *ShardedLock) getShard(key uint32) *sync.RWMutex {
	return sl.shards[key%sl.count]
}

// Lock acquires write lock for the shard corresponding to key.
func (sl *ShardedLock) Lock(key uint32) {
	sl.getShard(key).Lock()
}

// Unlock releases write lock for the shard corresponding to key.
func (sl *ShardedLock) Unlock(key uint32) {
	sl.getShard(key).Unlock()
}

// RLock acquires read lock for the shard corresponding to key.
func (sl *ShardedLock) RLock(key uint32) {
	sl.getShard(key).RLock()
}

// RUnlock releases read lock for the shard corresponding to key.
func (sl *ShardedLock) RUnlock(key uint32) {
	sl.getShard(key).RUnlock()
}

// ShardedMap provides a sharded map with per-shard locking.
type ShardedMap struct {
	shards []map[string]time.Time
	locks  []*sync.RWMutex
	count  uint32
}

// NewShardedMap creates a new sharded map.
func NewShardedMap(shardCount int) *ShardedMap {
	if shardCount <= 0 {
		shardCount = 16
	}

	shards := make([]map[string]time.Time, shardCount)
	locks := make([]*sync.RWMutex, shardCount)
	for i := range shards {
		shards[i] = make(map[string]time.Time)
		locks[i] = &sync.RWMutex{}
	}

	return &ShardedMap{
		shards: shards,
		locks:  locks,
		count:  uint32(shardCount),
	}
}

// hashKey computes a simple hash for string keys.
func (sm *ShardedMap) hashKey(key string) uint32 {
	h := uint32(0)
	for i := 0; i < len(key); i++ {
		h = h*31 + uint32(key[i])
	}
	return h
}

// getShard returns the shard index and lock for a key.
func (sm *ShardedMap) getShard(key string) (map[string]time.Time, *sync.RWMutex) {
	idx := sm.hashKey(key) % sm.count
	return sm.shards[idx], sm.locks[idx]
}

// Get retrieves a value from the map.
func (sm *ShardedMap) Get(key string) (time.Time, bool) {
	_, lock := sm.getShard(key)
	lock.RLock()
	defer lock.RUnlock()

	shard, _ := sm.getShard(key)
	val, ok := shard[key]
	return val, ok
}

// Set stores a value in the map.
func (sm *ShardedMap) Set(key string, value time.Time) {
	_, lock := sm.getShard(key)
	lock.Lock()
	defer lock.Unlock()

	shard, _ := sm.getShard(key)
	shard[key] = value
}

// Delete removes a key from the map.
func (sm *ShardedMap) Delete(key string) {
	_, lock := sm.getShard(key)
	lock.Lock()
	defer lock.Unlock()

	shard, _ := sm.getShard(key)
	delete(shard, key)
}

// Cleanup removes expired entries (where value + ttl < now).
func (sm *ShardedMap) Cleanup(ttl time.Duration) int {
	now := time.Now()
	removed := 0

	for i, shard := range sm.shards {
		lock := sm.locks[i]
		lock.Lock()
		for key, val := range shard {
			if now.Sub(val) > ttl {
				delete(shard, key)
				removed++
			}
		}
		lock.Unlock()
	}

	return removed
}

// Global sharded maps for high-concurrency scenarios.
var (
	// ShardedSuppressionWindows replaces suppressionWindows for better concurrency.
	ShardedSuppressionWindows = NewShardedMap(32)

	// ShardedAggLastSent replaces aggLastSent for better concurrency.
	ShardedAggLastSent = NewShardedMap(32)
)
