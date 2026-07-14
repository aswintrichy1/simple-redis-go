package store

import "testing"

func TestZAddNewAndUpdate(t *testing.T) {
	z := NewZSet()
	if !z.Add("a", 1) {
		t.Fatal("Add new should return true")
	}
	if z.Add("a", 2) {
		t.Fatal("Add existing should return false")
	}
	if got := z.scores["a"]; got != 2 {
		t.Fatalf("score = %v, want 2", got)
	}
}

func TestZRangeOrdering(t *testing.T) {
	z := NewZSet()
	z.Add("b", 2)
	z.Add("a", 1)
	z.Add("c", 2) // tie with b -> ordered by member name
	got := z.Range(0, -1)
	want := []ZMember{{1, "a"}, {2, "b"}, {2, "c"}}
	if len(got) != len(want) {
		t.Fatalf("Range = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %v want %v", i, got[i], want[i])
		}
	}
}

func TestZRangeIndices(t *testing.T) {
	z := NewZSet()
	for i, m := range []string{"a", "b", "c", "d"} {
		z.Add(m, float64(i))
	}
	tests := []struct {
		start, stop int
		want        []string
	}{
		{0, -1, []string{"a", "b", "c", "d"}},
		{0, 0, []string{"a"}},
		{-2, -1, []string{"c", "d"}},
		{1, 2, []string{"b", "c"}},
		{2, 1, nil},  // start > stop
		{5, 10, nil}, // out of range
		{-100, 0, []string{"a"}},
	}
	for _, tt := range tests {
		got := z.Range(tt.start, tt.stop)
		names := make([]string, len(got))
		for i, m := range got {
			names[i] = m.Member
		}
		if !equalStrings(names, tt.want) {
			t.Errorf("Range(%d,%d) = %v, want %v", tt.start, tt.stop, names, tt.want)
		}
	}
}
