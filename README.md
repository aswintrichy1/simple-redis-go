# simple-redis-go

A small, in-memory subset of Redis written in Go. It has two parts:

- **Server** (`cmd/redis-server`): an in-memory key/value store served over TCP.
- **Client** (`client` package + `cmd/redis-cli`): a reusable Go client library and a small command-line client.

Communication uses the [RESP2 protocol](https://redis.io/docs/reference/protocol-spec/), so the server also works with the official `redis-cli` for the supported commands.

## Supported commands

Command semantics follow current Redis. Only the common form of each command is
implemented (see [Deliberately omitted](#deliberately-omitted)).

| Command | Form | Reply |
| --- | --- | --- |
| `GET` | `GET key` | bulk string, or nil if missing |
| `SET` | `SET key value` | `+OK` (also clears any existing TTL) |
| `DEL` | `DEL key [key ...]` | integer: number of keys removed |
| `EXPIRE` | `EXPIRE key seconds` | `1` if applied, `0` if the key is missing; a non-positive TTL deletes the key |
| `TTL` | `TTL key` | remaining seconds, `-1` if no expiry, `-2` if missing |
| `KEYS` | `KEYS pattern` | array of matching keys (glob: `*`, `?`, `[...]`) |
| `ZADD` | `ZADD key score member [score member ...]` | integer: number of newly added members |
| `ZRANGE` | `ZRANGE key start stop [WITHSCORES]` | array of members (by rank), optionally interleaved with scores |

Sorted sets order by score ascending, ties broken lexicographically by member.
`ZRANGE` indexes are zero-based and inclusive; negative indexes count from the end.

`PING` is also supported for connectivity checks.

## Layout

```
cmd/redis-server   server entry point
cmd/redis-cli      command-line client
client             reusable Go client library
internal/resp      RESP2 reader/writer
internal/store     in-memory keyspace (strings, sorted sets, TTLs)
internal/command   command dispatch and validation
internal/server    TCP server / connection loop
```

The store is transport-agnostic and the command layer neither reads nor writes
sockets, so both can be tested without a network. Concurrency safety is provided
by a single mutex around the store; every connection is handled in its own
goroutine.

## Run

Start the server:

```sh
go run ./cmd/redis-server --addr 127.0.0.1:6379
```

Run one-off commands with the bundled CLI:

```sh
go run ./cmd/redis-cli SET hello world
go run ./cmd/redis-cli GET hello
go run ./cmd/redis-cli EXPIRE hello 60
go run ./cmd/redis-cli TTL hello
go run ./cmd/redis-cli ZADD ranks 2 bob 1 alice
go run ./cmd/redis-cli ZRANGE ranks 0 -1 WITHSCORES
```

Or start an interactive session (no command arguments):

```sh
go run ./cmd/redis-cli
127.0.0.1:6379> SET hello world
OK
127.0.0.1:6379> GET hello
"world"
```

### Using the client library

```go
c, err := client.Dial("127.0.0.1:6379")
if err != nil {
    log.Fatal(err)
}
defer c.Close()

c.Set("hello", "world")
val, ok, _ := c.Get("hello")   // "world", true
c.ZAdd("ranks", client.ZMember{Score: 1, Member: "alice"})
members, _ := c.ZRange("ranks", 0, -1, true)
```

## Test

```sh
go test ./...
go test -race ./...
```

## Deliberately omitted

To keep the implementation simple, options beyond the common command form are
not implemented:

- `SET` options: `NX`, `XX`, `GET`, `EX`, `PX`, `EXAT`, `PXAT`, `KEEPTTL`
- `EXPIRE` options: `NX`, `XX`, `GT`, `LT`
- `ZADD` options: `NX`, `XX`, `GT`, `LT`, `CH`, `INCR`
- `ZRANGE` options: `BYSCORE`, `BYLEX`, `REV`, `LIMIT`

Persistence, replication, clustering, authentication, transactions, pub/sub, and
eviction are also out of scope. The protocol is RESP2 only.
