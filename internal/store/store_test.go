package store

import (
	"testing"
	"time"
)

func TestSetGet(t *testing.T) {
	s := New()
	s.Set("k", "v")
	got, ok, err := s.Get("k")
	if err != nil || !ok || got != "v" {
		t.Fatalf("Get = (%q,%v,%v), want (v,true,nil)", got, ok, err)
	}
}

func TestGetMissing(t *testing.T) {
	s := New()
	if _, ok, err := s.Get("nope"); err != nil || ok {
		t.Fatalf("Get missing = (ok=%v,err=%v)", ok, err)
	}
}

func TestSetClearsTTL(t *testing.T) {
	s := New()
	s.Set("k", "v")
	s.Expire("k", 100)
	if ttl := s.TTL("k"); ttl != 100 {
		t.Fatalf("TTL before reset = %d, want 100", ttl)
	}
	s.Set("k", "v2")
	if ttl := s.TTL("k"); ttl != -1 {
		t.Fatalf("TTL after Set = %d, want -1", ttl)
	}
}

func TestDel(t *testing.T) {
	s := New()
	s.Set("a", "1")
	s.Set("b", "2")
	if n := s.Del("a", "b", "missing"); n != 2 {
		t.Fatalf("Del = %d, want 2", n)
	}
}

func TestExpireAndTTL(t *testing.T) {
	s := New()
	if s.Expire("missing", 10) {
		t.Fatal("Expire on missing key should return false")
	}
	if ttl := s.TTL("missing"); ttl != -2 {
		t.Fatalf("TTL missing = %d, want -2", ttl)
	}
	s.Set("k", "v")
	if ttl := s.TTL("k"); ttl != -1 {
		t.Fatalf("TTL no-expiry = %d, want -1", ttl)
	}
	if !s.Expire("k", 50) {
		t.Fatal("Expire should return true")
	}
	if ttl := s.TTL("k"); ttl != 50 {
		t.Fatalf("TTL = %d, want 50", ttl)
	}
}

func TestExpireImmediateDelete(t *testing.T) {
	s := New()
	s.Set("k", "v")
	if !s.Expire("k", 0) {
		t.Fatal("Expire 0 should return true for existing key")
	}
	if _, ok, _ := s.Get("k"); ok {
		t.Fatal("key should be deleted after Expire 0")
	}
}

func TestLazyExpiry(t *testing.T) {
	s := New()
	base := time.Now()
	s.now = func() time.Time { return base }
	s.Set("k", "v")
	s.Expire("k", 10)

	s.now = func() time.Time { return base.Add(11 * time.Second) }
	if _, ok, _ := s.Get("k"); ok {
		t.Fatal("expired key should be gone")
	}
	if ttl := s.TTL("k"); ttl != -2 {
		t.Fatalf("TTL after expiry = %d, want -2", ttl)
	}
}

func TestWrongType(t *testing.T) {
	s := New()
	if _, err := s.ZAdd("z", []ZMember{{Score: 1, Member: "a"}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Get("z"); err != ErrWrongType {
		t.Fatalf("Get on zset = %v, want ErrWrongType", err)
	}
	s.Set("str", "v")
	if _, err := s.ZRange("str", 0, -1); err != ErrWrongType {
		t.Fatalf("ZRange on string = %v, want ErrWrongType", err)
	}
}

func TestKeys(t *testing.T) {
	s := New()
	s.Set("hello", "1")
	s.Set("help", "1")
	s.Set("world", "1")
	if got := s.Keys("hel*"); !equalStrings(got, []string{"hello", "help"}) {
		t.Fatalf("Keys(hel*) = %v, want [hello help]", got)
	}
	if got := s.Keys("*"); len(got) != 3 {
		t.Fatalf("Keys(*) = %v, want 3 keys", got)
	}
}

func TestKeysSkipsExpired(t *testing.T) {
	s := New()
	base := time.Now()
	s.now = func() time.Time { return base }
	s.Set("live", "1")
	s.Set("dead", "1")
	s.Expire("dead", 5)
	s.now = func() time.Time { return base.Add(6 * time.Second) }
	if got := s.Keys("*"); !equalStrings(got, []string{"live"}) {
		t.Fatalf("Keys(*) = %v, want [live]", got)
	}
}

func equalStrings(a, b []string) bool {
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
