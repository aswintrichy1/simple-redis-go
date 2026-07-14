package store

import "sort"

// ZMember is a scored member of a sorted set.
type ZMember struct {
	Score  float64
	Member string
}

// ZSet is a sorted set: a collection of unique members each associated with a
// float64 score. Ordering is by score ascending, ties broken by member name.
type ZSet struct {
	scores map[string]float64
}

// NewZSet returns an empty sorted set.
func NewZSet() *ZSet {
	return &ZSet{scores: make(map[string]float64)}
}

// Add sets member's score, returning true if the member was newly added and
// false if an existing member's score was updated.
func (z *ZSet) Add(member string, score float64) bool {
	_, existed := z.scores[member]
	z.scores[member] = score
	return !existed
}

// sorted returns all members ordered by score then member name.
func (z *ZSet) sorted() []ZMember {
	members := make([]ZMember, 0, len(z.scores))
	for m, sc := range z.scores {
		members = append(members, ZMember{Score: sc, Member: m})
	}
	sort.Slice(members, func(i, j int) bool {
		if members[i].Score != members[j].Score {
			return members[i].Score < members[j].Score
		}
		return members[i].Member < members[j].Member
	})
	return members
}

// Range returns the members within the inclusive index range [start, stop],
// following Redis ZRANGE index semantics (negative indices count from the end).
func (z *ZSet) Range(start, stop int) []ZMember {
	all := z.sorted()
	n := len(all)

	if start < 0 {
		start = n + start
	}
	if stop < 0 {
		stop = n + stop
	}
	if start < 0 {
		start = 0
	}
	if start > stop || start >= n {
		return []ZMember{}
	}
	if stop >= n {
		stop = n - 1
	}
	return all[start : stop+1]
}
