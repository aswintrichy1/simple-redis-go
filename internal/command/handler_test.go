package command_test

import (
	"testing"

	"simple-redis-go/internal/command"
	"simple-redis-go/internal/resp"
	"simple-redis-go/internal/store"
)

func newHandler() *command.Handler {
	return command.NewHandler(store.New())
}

func TestSetGet(t *testing.T) {
	h := newHandler()
	if got := h.Execute([]string{"SET", "k", "v"}); got.Type != resp.SimpleString || got.Str != "OK" {
		t.Fatalf("SET = %+v, want +OK", got)
	}
	got := h.Execute([]string{"GET", "k"})
	if got.Type != resp.BulkString || got.Null || got.Str != "v" {
		t.Fatalf("GET = %+v, want bulk v", got)
	}
}

func TestGetMissingIsNull(t *testing.T) {
	got := newHandler().Execute([]string{"GET", "missing"})
	if got.Type != resp.BulkString || !got.Null {
		t.Fatalf("GET missing = %+v, want null bulk", got)
	}
}

func TestCaseInsensitive(t *testing.T) {
	h := newHandler()
	h.Execute([]string{"set", "k", "v"})
	if got := h.Execute([]string{"gEt", "k"}); got.Str != "v" {
		t.Fatalf("case-insensitive GET = %+v", got)
	}
}

func TestDel(t *testing.T) {
	h := newHandler()
	h.Execute([]string{"SET", "a", "1"})
	h.Execute([]string{"SET", "b", "2"})
	if got := h.Execute([]string{"DEL", "a", "b", "c"}); got.Type != resp.Integer || got.Int != 2 {
		t.Fatalf("DEL = %+v, want :2", got)
	}
}

func TestExpireTTL(t *testing.T) {
	h := newHandler()
	if got := h.Execute([]string{"TTL", "missing"}); got.Int != -2 {
		t.Fatalf("TTL missing = %+v, want -2", got)
	}
	h.Execute([]string{"SET", "k", "v"})
	if got := h.Execute([]string{"TTL", "k"}); got.Int != -1 {
		t.Fatalf("TTL no-expiry = %+v, want -1", got)
	}
	if got := h.Execute([]string{"EXPIRE", "k", "100"}); got.Int != 1 {
		t.Fatalf("EXPIRE = %+v, want :1", got)
	}
	if got := h.Execute([]string{"TTL", "k"}); got.Int != 100 {
		t.Fatalf("TTL = %+v, want 100", got)
	}
	if got := h.Execute([]string{"EXPIRE", "missing", "10"}); got.Int != 0 {
		t.Fatalf("EXPIRE missing = %+v, want :0", got)
	}
}

func TestKeys(t *testing.T) {
	h := newHandler()
	h.Execute([]string{"SET", "hello", "1"})
	h.Execute([]string{"SET", "help", "1"})
	h.Execute([]string{"SET", "xyz", "1"})
	got := h.Execute([]string{"KEYS", "hel*"})
	if got.Type != resp.Array || len(got.Array) != 2 {
		t.Fatalf("KEYS = %+v, want 2 elements", got)
	}
	if got.Array[0].Str != "hello" || got.Array[1].Str != "help" {
		t.Fatalf("KEYS order = %+v", got.Array)
	}
}

func TestZAddZRange(t *testing.T) {
	h := newHandler()
	if got := h.Execute([]string{"ZADD", "z", "2", "bob", "1", "alice", "3", "carol"}); got.Type != resp.Integer || got.Int != 3 {
		t.Fatalf("ZADD = %+v, want :3", got)
	}
	if got := h.Execute([]string{"ZADD", "z", "5", "bob"}); got.Int != 0 {
		t.Fatalf("ZADD update = %+v, want :0", got)
	}
	got := bulkStrs(h.Execute([]string{"ZRANGE", "z", "0", "-1"}))
	if want := []string{"alice", "carol", "bob"}; !eqStrs(got, want) {
		t.Fatalf("ZRANGE = %v, want %v", got, want)
	}
	got = bulkStrs(h.Execute([]string{"ZRANGE", "z", "0", "-1", "WITHSCORES"}))
	if want := []string{"alice", "1", "carol", "3", "bob", "5"}; !eqStrs(got, want) {
		t.Fatalf("ZRANGE WITHSCORES = %v, want %v", got, want)
	}
}

func TestZRangeMissingKeyIsEmptyArray(t *testing.T) {
	got := newHandler().Execute([]string{"ZRANGE", "nope", "0", "-1"})
	if got.Type != resp.Array || len(got.Array) != 0 {
		t.Fatalf("ZRANGE missing = %+v, want empty array", got)
	}
}

func TestZAddZRangeInfinity(t *testing.T) {
	h := newHandler()
	h.Execute([]string{"ZADD", "z", "inf", "top", "-inf", "bottom", "3.5", "mid"})
	got := bulkStrs(h.Execute([]string{"ZRANGE", "z", "0", "-1", "WITHSCORES"}))
	if want := []string{"bottom", "-inf", "mid", "3.5", "top", "inf"}; !eqStrs(got, want) {
		t.Fatalf("ZRANGE WITHSCORES = %v, want %v", got, want)
	}
}

func TestWrongTypeError(t *testing.T) {
	h := newHandler()
	h.Execute([]string{"ZADD", "z", "1", "a"})
	if got := h.Execute([]string{"GET", "z"}); got.Type != resp.Error {
		t.Fatalf("GET on zset = %+v, want error", got)
	}
}

func TestArityErrors(t *testing.T) {
	h := newHandler()
	cmds := [][]string{
		{"GET"},
		{"GET", "a", "b"},
		{"SET", "k"},
		{"DEL"},
		{"EXPIRE", "k"},
		{"TTL"},
		{"KEYS"},
		{"ZADD", "z", "1"},           // missing member
		{"ZADD", "z", "1", "a", "2"}, // odd score/member pairs
		{"ZRANGE", "z", "0"},
	}
	for _, c := range cmds {
		if got := h.Execute(c); got.Type != resp.Error {
			t.Errorf("Execute(%v) = %+v, want error", c, got)
		}
	}
}

func TestInvalidNumbers(t *testing.T) {
	h := newHandler()
	if got := h.Execute([]string{"EXPIRE", "k", "abc"}); got.Type != resp.Error {
		t.Errorf("EXPIRE non-int = %+v", got)
	}
	if got := h.Execute([]string{"ZADD", "z", "notafloat", "m"}); got.Type != resp.Error {
		t.Errorf("ZADD non-float = %+v", got)
	}
	if got := h.Execute([]string{"ZADD", "z", "NaN", "m"}); got.Type != resp.Error {
		t.Errorf("ZADD NaN = %+v", got)
	}
	if got := h.Execute([]string{"ZRANGE", "z", "x", "y"}); got.Type != resp.Error {
		t.Errorf("ZRANGE non-int = %+v", got)
	}
	if got := h.Execute([]string{"ZRANGE", "z", "0", "1", "BADOPT"}); got.Type != resp.Error {
		t.Errorf("ZRANGE bad option = %+v", got)
	}
}

func TestUnknownCommand(t *testing.T) {
	if got := newHandler().Execute([]string{"FLUSHALL"}); got.Type != resp.Error {
		t.Errorf("unknown command = %+v, want error", got)
	}
}

func bulkStrs(v resp.Value) []string {
	out := make([]string, len(v.Array))
	for i, item := range v.Array {
		out[i] = item.Str
	}
	return out
}

func eqStrs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
