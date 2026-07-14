// Package store implements the in-memory keyspace backing the server. It holds
// string and sorted-set values, tracks per-key expirations, and exposes the
// operations required by the supported commands. All methods are safe for
// concurrent use.
package store

import (
	"errors"
	"sort"
	"sync"
	"time"
)

// ErrWrongType is returned when a command is issued against a key holding a
// value of an incompatible type, mirroring Redis's WRONGTYPE error.
var ErrWrongType = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")

type valueType int

const (
	typeString valueType = iota
	typeZSet
)

// item is a single value in the keyspace along with its optional expiry.
type item struct {
	typ    valueType
	str    string
	zset   *ZSet
	expiry time.Time // zero means the key never expires
}

// Store is a concurrency-safe in-memory keyspace.
type Store struct {
	mu   sync.Mutex
	data map[string]*item
	// now is injectable so tests can control time; defaults to time.Now.
	now func() time.Time
}

// New returns an empty Store.
func New() *Store {
	return &Store{
		data: make(map[string]*item),
		now:  time.Now,
	}
}

// live returns the item for key if it exists and has not expired. Expired keys
// are deleted lazily. The caller must hold s.mu.
func (s *Store) live(key string) *item {
	it, ok := s.data[key]
	if !ok {
		return nil
	}
	if !it.expiry.IsZero() && s.now().After(it.expiry) {
		delete(s.data, key)
		return nil
	}
	return it
}

// Get returns the string value at key. The boolean reports whether the key
// exists; ErrWrongType is returned if the key holds a non-string value.
func (s *Store) Get(key string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	it := s.live(key)
	if it == nil {
		return "", false, nil
	}
	if it.typ != typeString {
		return "", false, ErrWrongType
	}
	return it.str, true, nil
}

// Set stores value as a string at key, discarding any previous value and TTL.
func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = &item{typ: typeString, str: value}
}

// Del removes the given keys and returns how many were actually deleted.
func (s *Store) Del(keys ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for _, key := range keys {
		if s.live(key) != nil {
			delete(s.data, key)
			count++
		}
	}
	return count
}

// Expire sets a TTL of seconds on key. A non-positive TTL deletes the key
// immediately. It returns true if the key existed and the timeout was applied
// (or the key was deleted), false if the key does not exist.
func (s *Store) Expire(key string, seconds int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	it := s.live(key)
	if it == nil {
		return false
	}
	if seconds <= 0 {
		delete(s.data, key)
		return true
	}
	it.expiry = s.now().Add(time.Duration(seconds) * time.Second)
	return true
}

// TTL returns the remaining time to live of key in seconds, -1 if the key
// exists but has no expiry, and -2 if the key does not exist.
func (s *Store) TTL(key string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	it := s.live(key)
	if it == nil {
		return -2
	}
	if it.expiry.IsZero() {
		return -1
	}
	remaining := it.expiry.Sub(s.now())
	if remaining < 0 {
		remaining = 0
	}
	// Round to the nearest second, matching Redis's TTL rounding.
	millis := int64(remaining / time.Millisecond)
	return (millis + 500) / 1000
}

// Keys returns all non-expired keys matching the glob-style pattern, sorted
// for deterministic output.
func (s *Store) Keys(pattern string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []string
	for key := range s.data {
		if s.live(key) == nil {
			continue
		}
		if matchPattern(pattern, key) {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}

// ZAdd adds the given members to the sorted set at key, creating it if needed.
// Existing members have their scores updated. It returns the number of members
// that were newly added (not counting score updates).
func (s *Store) ZAdd(key string, members []ZMember) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	it := s.live(key)
	if it == nil {
		it = &item{typ: typeZSet, zset: NewZSet()}
		s.data[key] = it
	} else if it.typ != typeZSet {
		return 0, ErrWrongType
	}

	added := 0
	for _, m := range members {
		if it.zset.Add(m.Member, m.Score) {
			added++
		}
	}
	return added, nil
}

// ZRange returns the members of the sorted set at key within the inclusive
// index range [start, stop], ordered by score then lexicographically. Negative
// indices count from the end. A missing key yields an empty slice.
func (s *Store) ZRange(key string, start, stop int) ([]ZMember, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	it := s.live(key)
	if it == nil {
		return nil, nil
	}
	if it.typ != typeZSet {
		return nil, ErrWrongType
	}
	return it.zset.Range(start, stop), nil
}
