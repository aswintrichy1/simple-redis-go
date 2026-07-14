# simple-redis-go Demo

Use this walkthrough to demo both components of the assignment:

1. The in-memory Redis-like server.
2. The Go client and CLI that communicate with the server using RESP.

The examples below use port `6381` to avoid conflicting with a real Redis server on `6379`.

## 1. Start The Server

Open Terminal 1:

```sh
export PATH="/tmp/go/bin:$PATH"
cd /Users/aswsures/simple-redis-go
go run ./cmd/redis-server --addr 127.0.0.1:6381
```

Expected output:

```text
simple-redis-go listening on 127.0.0.1:6381
```

What to explain:

- This starts the in-memory database server.
- The server listens on TCP address `127.0.0.1:6381`.
- Clients send Redis commands encoded using RESP2.
- The server decodes RESP, executes the command against the in-memory store, and sends a RESP reply.

## 2. Check Connectivity

Open Terminal 2:

```sh
export PATH="/tmp/go/bin:$PATH"
cd /Users/aswsures/simple-redis-go
go run ./cmd/redis-cli --addr 127.0.0.1:6381 PING
```

Expected output:

```text
PONG
```

What to explain:

- `PING` is a simple connectivity check.
- It proves the client can connect to the server and receive a RESP reply.

## 3. Demo String Commands: SET And GET

```sh
go run ./cmd/redis-cli --addr 127.0.0.1:6381 SET greeting "hello redis"
go run ./cmd/redis-cli --addr 127.0.0.1:6381 GET greeting
```

Expected output:

```text
OK
"hello redis"
```

What to explain:

- `SET greeting "hello redis"` stores a string value in memory.
- The server replies with `OK`, encoded as a RESP simple string: `+OK\r\n`.
- `GET greeting` returns the stored value as a RESP bulk string.
- If a key is missing, `GET` returns RESP nil, not an empty string.

## 4. Demo TTL Semantics: TTL And EXPIRE

```sh
go run ./cmd/redis-cli --addr 127.0.0.1:6381 TTL greeting
go run ./cmd/redis-cli --addr 127.0.0.1:6381 EXPIRE greeting 30
go run ./cmd/redis-cli --addr 127.0.0.1:6381 TTL greeting
```

Expected output:

```text
(integer) -1
(integer) 1
(integer) 30
```

What to explain:

- `TTL greeting` returns `-1` because the key exists but has no expiry.
- `EXPIRE greeting 30` applies a 30-second timeout and returns `1`.
- The next `TTL greeting` returns the remaining lifetime in seconds.
- The store uses lazy expiration: expired keys are deleted when touched by a command.

## 5. Demo KEYS And DEL

```sh
go run ./cmd/redis-cli --addr 127.0.0.1:6381 KEYS '*'
go run ./cmd/redis-cli --addr 127.0.0.1:6381 DEL missing greeting
go run ./cmd/redis-cli --addr 127.0.0.1:6381 GET greeting
go run ./cmd/redis-cli --addr 127.0.0.1:6381 TTL greeting
```

Expected output:

```text
1) "greeting"
(integer) 1
(nil)
(integer) -2
```

What to explain:

- `KEYS '*'` returns all non-expired keys matching the glob pattern.
- `DEL missing greeting` returns `1` because only `greeting` existed.
- `GET greeting` returns `(nil)` after deletion.
- `TTL greeting` returns `-2`, which means the key does not exist.

## 6. Demo Sorted Sets: ZADD And ZRANGE

```sh
go run ./cmd/redis-cli --addr 127.0.0.1:6381 ZADD leaderboard 20 bob 10 alice 20 carol
go run ./cmd/redis-cli --addr 127.0.0.1:6381 ZRANGE leaderboard 0 -1
go run ./cmd/redis-cli --addr 127.0.0.1:6381 ZRANGE leaderboard 0 -1 WITHSCORES
```

Expected output:

```text
(integer) 3

1) "alice"
2) "bob"
3) "carol"

1) "alice"
2) "10"
3) "bob"
4) "20"
5) "carol"
6) "20"
```

What to explain:

- `ZADD` creates a sorted set called `leaderboard`.
- It returns `3` because three new members were added.
- `ZRANGE leaderboard 0 -1` returns all members.
- Sorted-set ordering is by score ascending.
- If scores tie, members are ordered lexicographically, so `bob` comes before `carol` when both have score `20`.
- `WITHSCORES` returns member/score pairs in a flat RESP array.

## 7. Demo Sorted Set Updates

```sh
go run ./cmd/redis-cli --addr 127.0.0.1:6381 ZADD leaderboard 99 bob
go run ./cmd/redis-cli --addr 127.0.0.1:6381 ZRANGE leaderboard 0 -1 WITHSCORES
```

Expected output:

```text
(integer) 0

1) "alice"
2) "10"
3) "carol"
4) "20"
5) "bob"
6) "99"
```

What to explain:

- `ZADD leaderboard 99 bob` updates the existing member `bob`.
- Redis returns the count of newly added members by default, so the reply is `0`.
- `bob` moves after `carol` because his score changed from `20` to `99`.

## 8. Demo Wrong-Type Correctness

```sh
go run ./cmd/redis-cli --addr 127.0.0.1:6381 GET leaderboard
```

Expected output:

```text
(error) WRONGTYPE Operation against a key holding the wrong kind of value
```

What to explain:

- Redis has typed keys.
- `leaderboard` is a sorted set, not a string.
- `GET` only works on string keys.
- Returning a wrong-type error is more correct than pretending the key is missing.

## 9. Optional: Show Raw RESP Over TCP

This proves the server is not coupled to our client. It accepts RESP bytes over a plain TCP socket.

```sh
python3 - <<'PY'
import socket

payload = (
    b'*3\r\n$3\r\nSET\r\n$4\r\nwire\r\n$4\r\ndemo\r\n'
    b'*2\r\n$3\r\nGET\r\n$4\r\nwire\r\n'
)

with socket.create_connection(('127.0.0.1', 6381), timeout=2) as sock:
    sock.sendall(payload)
    sock.shutdown(socket.SHUT_WR)
    data = sock.recv(4096)

print(data.decode())
PY
```

Expected output:

```text
+OK
$4
demo
```

What to explain:

- The payload is raw RESP.
- `*3` means an array of three bulk strings: `SET`, `wire`, `demo`.
- `*2` means an array of two bulk strings: `GET`, `wire`.
- `+OK` is the RESP simple string reply from `SET`.
- `$4\r\ndemo\r\n` is the RESP bulk string reply from `GET`.

## 10. Optional: Interactive CLI Mode

```sh
go run ./cmd/redis-cli --addr 127.0.0.1:6381
```

Then type:

```text
SET city "San Jose"
GET city
ZADD scores 2 bob 1 alice
ZRANGE scores 0 -1 WITHSCORES
exit
```

What to explain:

- With no command arguments, the client enters interactive mode.
- The CLI parses the typed command, converts it into RESP, sends it to the server, and formats the RESP reply.

## 11. Stop The Server

Return to Terminal 1 and press:

```text
Ctrl+C
```

## End-To-End Story

Use this summary in the demo:

> The server is a TCP process that reads RESP arrays, dispatches them to a small in-memory typed store, and returns RESP replies. The client is a RESP writer/reader wrapper with typed helpers and a CLI on top. The implementation focuses on correctness for the supported Redis command subset, simplicity in the data structures, and testability through isolated protocol, store, command, and integration tests.

## Commands Covered In This Demo

| Command | Demonstrated behavior |
| --- | --- |
| `GET` | returns string value, nil for missing key, wrong-type error for sorted set |
| `SET` | stores string value and returns `OK` |
| `DEL` | deletes existing keys and returns count removed |
| `EXPIRE` | applies key timeout and returns `1` |
| `TTL` | returns `-1` for no expiry and `-2` for missing key |
| `KEYS` | matches keys with glob pattern |
| `ZADD` | creates sorted set, updates existing member, returns new-member count |
| `ZRANGE` | returns sorted-set members by rank, supports `WITHSCORES` |

## Why This Demo Proves The Assignment Requirements

Correctness:

- Shows Redis-like replies for strings, missing keys, TTLs, deletion, sorted sets, and wrong-type access.
- Shows sorted-set ordering by score and lexicographic tie-break.
- Shows raw RESP compatibility over TCP.

Simplicity:

- Server is one process, one TCP listener, one goroutine per client.
- Store is an in-memory map protected by a mutex.
- Sorted sets use a member-to-score map and sort on `ZRANGE`.
- Expiration is lazy and easy to reason about.

Testability:

- Protocol parsing/writing can be tested without the server.
- Store behavior can be tested without TCP.
- Command handlers can be tested with plain `[]string` inputs.
- Integration tests exercise real TCP server/client communication.
