package server_test

import (
	"strconv"
	"sync"
	"testing"

	"simple-redis-go/client"
	"simple-redis-go/internal/server"
	"simple-redis-go/internal/store"
)

func startServer(t *testing.T) string {
	t.Helper()
	srv := server.New(store.New())
	if err := srv.Bind("127.0.0.1:0"); err != nil {
		t.Fatalf("Bind: %v", err)
	}
	go srv.Serve()
	t.Cleanup(func() { srv.Close() })
	return srv.Addr()
}

func dial(t *testing.T, addr string) *client.Client {
	t.Helper()
	c, err := client.Dial(addr)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestEndToEndStrings(t *testing.T) {
	c := dial(t, startServer(t))

	if err := c.Set("hello", "world"); err != nil {
		t.Fatal(err)
	}
	if val, ok, err := c.Get("hello"); err != nil || !ok || val != "world" {
		t.Fatalf("Get = (%q,%v,%v)", val, ok, err)
	}
	if _, ok, _ := c.Get("missing"); ok {
		t.Fatal("missing key should not exist")
	}
	if n, err := c.Del("hello"); err != nil || n != 1 {
		t.Fatalf("Del = (%d,%v)", n, err)
	}
}

func TestEndToEndExpire(t *testing.T) {
	c := dial(t, startServer(t))

	c.Set("k", "v")
	if ttl, _ := c.TTL("k"); ttl != -1 {
		t.Fatalf("TTL = %d, want -1", ttl)
	}
	if ok, _ := c.Expire("k", 100); !ok {
		t.Fatal("Expire should apply")
	}
	if ttl, _ := c.TTL("k"); ttl < 99 || ttl > 100 {
		t.Fatalf("TTL = %d, want ~100", ttl)
	}
}

func TestEndToEndExpireDeletes(t *testing.T) {
	c := dial(t, startServer(t))

	c.Set("k", "v")
	if ok, _ := c.Expire("k", -1); !ok {
		t.Fatal("Expire -1 should return true for existing key")
	}
	if _, ok, _ := c.Get("k"); ok {
		t.Fatal("key should be deleted")
	}
}

func TestEndToEndKeys(t *testing.T) {
	c := dial(t, startServer(t))

	c.Set("user:1", "a")
	c.Set("user:2", "b")
	c.Set("other", "c")
	keys, _ := c.Keys("user:*")
	if len(keys) != 2 || keys[0] != "user:1" || keys[1] != "user:2" {
		t.Fatalf("Keys = %v", keys)
	}
}

func TestEndToEndSortedSet(t *testing.T) {
	c := dial(t, startServer(t))

	n, err := c.ZAdd("ranks",
		client.ZMember{Score: 2, Member: "bob"},
		client.ZMember{Score: 1, Member: "alice"})
	if err != nil || n != 2 {
		t.Fatalf("ZAdd = (%d,%v)", n, err)
	}
	members, _ := c.ZRange("ranks", 0, -1, false)
	if len(members) != 2 || members[0] != "alice" || members[1] != "bob" {
		t.Fatalf("ZRange = %v", members)
	}
	ws, _ := c.ZRange("ranks", 0, -1, true)
	want := []string{"alice", "1", "bob", "2"}
	if len(ws) != len(want) {
		t.Fatalf("ZRange WITHSCORES = %v, want %v", ws, want)
	}
	for i := range want {
		if ws[i] != want[i] {
			t.Fatalf("ZRange WITHSCORES = %v, want %v", ws, want)
		}
	}
}

func TestEndToEndWrongType(t *testing.T) {
	c := dial(t, startServer(t))

	c.ZAdd("z", client.ZMember{Score: 1, Member: "a"})
	if _, _, err := c.Get("z"); err == nil {
		t.Fatal("Get on sorted set should return an error")
	}
}

func TestConcurrentClients(t *testing.T) {
	addr := startServer(t)

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			c, err := client.Dial(addr)
			if err != nil {
				t.Errorf("Dial: %v", err)
				return
			}
			defer c.Close()

			key := "key" + strconv.Itoa(i)
			if err := c.Set(key, "v"); err != nil {
				t.Errorf("Set: %v", err)
				return
			}
			if val, ok, err := c.Get(key); err != nil || !ok || val != "v" {
				t.Errorf("Get(%s) = (%q,%v,%v)", key, val, ok, err)
			}
		}(i)
	}
	wg.Wait()
}
