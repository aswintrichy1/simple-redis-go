// Package command dispatches parsed RESP commands to the in-memory store and
// produces RESP reply values. It is transport-agnostic: it neither reads from
// nor writes to any connection, which keeps command semantics easy to test.
package command

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"simple-redis-go/internal/resp"
	"simple-redis-go/internal/store"
)

// Handler executes commands against a Store.
type Handler struct {
	store *store.Store
}

// NewHandler returns a Handler backed by s.
func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

// Execute runs a single command. args[0] is the command name (case-insensitive)
// and the remaining elements are its arguments. It always returns a RESP value,
// using an Error value for any failure.
func (h *Handler) Execute(args []string) resp.Value {
	if len(args) == 0 {
		return resp.ErrorValue("ERR empty command")
	}

	switch strings.ToUpper(args[0]) {
	case "GET":
		return h.get(args)
	case "SET":
		return h.set(args)
	case "DEL":
		return h.del(args)
	case "EXPIRE":
		return h.expire(args)
	case "TTL":
		return h.ttl(args)
	case "KEYS":
		return h.keys(args)
	case "ZADD":
		return h.zadd(args)
	case "ZRANGE":
		return h.zrange(args)
	case "PING":
		return h.ping(args)
	case "COMMAND":
		// Minimal stub so real redis-cli can complete its handshake.
		return resp.ArrayValue(nil)
	default:
		return resp.ErrorValue(fmt.Sprintf("ERR unknown command '%s'", args[0]))
	}
}

func (h *Handler) get(args []string) resp.Value {
	if len(args) != 2 {
		return wrongArgs("get")
	}
	val, ok, err := h.store.Get(args[1])
	if err != nil {
		return resp.ErrorValue(err.Error())
	}
	if !ok {
		return resp.NullValue()
	}
	return resp.BulkStringValue(val)
}

// set implements the common form: SET key value.
func (h *Handler) set(args []string) resp.Value {
	if len(args) != 3 {
		return wrongArgs("set")
	}
	h.store.Set(args[1], args[2])
	return resp.SimpleStringValue("OK")
}

func (h *Handler) del(args []string) resp.Value {
	if len(args) < 2 {
		return wrongArgs("del")
	}
	return resp.IntegerValue(int64(h.store.Del(args[1:]...)))
}

// expire implements the common form: EXPIRE key seconds.
func (h *Handler) expire(args []string) resp.Value {
	if len(args) != 3 {
		return wrongArgs("expire")
	}
	seconds, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		return resp.ErrorValue("ERR value is not an integer or out of range")
	}
	if h.store.Expire(args[1], seconds) {
		return resp.IntegerValue(1)
	}
	return resp.IntegerValue(0)
}

func (h *Handler) ttl(args []string) resp.Value {
	if len(args) != 2 {
		return wrongArgs("ttl")
	}
	return resp.IntegerValue(h.store.TTL(args[1]))
}

func (h *Handler) keys(args []string) resp.Value {
	if len(args) != 2 {
		return wrongArgs("keys")
	}
	keys := h.store.Keys(args[1])
	items := make([]resp.Value, len(keys))
	for i, k := range keys {
		items[i] = resp.BulkStringValue(k)
	}
	return resp.ArrayValue(items)
}

// zadd implements the common form: ZADD key score member [score member ...].
func (h *Handler) zadd(args []string) resp.Value {
	if len(args) < 4 || (len(args)-2)%2 != 0 {
		return wrongArgs("zadd")
	}
	members := make([]store.ZMember, 0, (len(args)-2)/2)
	for i := 2; i < len(args); i += 2 {
		score, err := strconv.ParseFloat(args[i], 64)
		if err != nil || math.IsNaN(score) {
			return resp.ErrorValue("ERR value is not a valid float")
		}
		members = append(members, store.ZMember{Score: score, Member: args[i+1]})
	}
	added, err := h.store.ZAdd(args[1], members)
	if err != nil {
		return resp.ErrorValue(err.Error())
	}
	return resp.IntegerValue(int64(added))
}

// zrange implements the common form: ZRANGE key start stop [WITHSCORES].
func (h *Handler) zrange(args []string) resp.Value {
	if len(args) != 4 && len(args) != 5 {
		return wrongArgs("zrange")
	}
	withScores := false
	if len(args) == 5 {
		if !strings.EqualFold(args[4], "WITHSCORES") {
			return resp.ErrorValue("ERR syntax error")
		}
		withScores = true
	}
	start, err1 := strconv.Atoi(args[2])
	stop, err2 := strconv.Atoi(args[3])
	if err1 != nil || err2 != nil {
		return resp.ErrorValue("ERR value is not an integer or out of range")
	}
	members, err := h.store.ZRange(args[1], start, stop)
	if err != nil {
		return resp.ErrorValue(err.Error())
	}
	items := make([]resp.Value, 0, len(members))
	for _, m := range members {
		items = append(items, resp.BulkStringValue(m.Member))
		if withScores {
			items = append(items, resp.BulkStringValue(formatScore(m.Score)))
		}
	}
	return resp.ArrayValue(items)
}

func (h *Handler) ping(args []string) resp.Value {
	switch len(args) {
	case 1:
		return resp.SimpleStringValue("PONG")
	case 2:
		return resp.BulkStringValue(args[1])
	default:
		return wrongArgs("ping")
	}
}

func wrongArgs(cmd string) resp.Value {
	return resp.ErrorValue(fmt.Sprintf("ERR wrong number of arguments for '%s' command", cmd))
}

// formatScore renders a sorted-set score as the shortest string that
// round-trips, using Redis's "inf"/"-inf" spelling for infinities.
func formatScore(f float64) string {
	switch {
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	default:
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
}
