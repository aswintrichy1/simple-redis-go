// Package resp implements the subset of the RESP2 protocol used by the
// simple-redis-go server and client. Only the value types required by the
// supported commands are modelled.
package resp

// Type identifies a RESP2 value type by its leading byte.
type Type byte

const (
	SimpleString Type = '+'
	Error        Type = '-'
	Integer      Type = ':'
	BulkString   Type = '$'
	Array        Type = '*'
)

// Value is a decoded (or to-be-encoded) RESP2 value. Only the fields relevant
// to Type are populated.
type Value struct {
	Type  Type
	Str   string  // SimpleString, Error, BulkString
	Int   int64   // Integer
	Array []Value // Array
	Null  bool    // null BulkString ($-1) or null Array (*-1)
}

// SimpleStringValue returns a RESP simple string, e.g. +OK.
func SimpleStringValue(s string) Value { return Value{Type: SimpleString, Str: s} }

// ErrorValue returns a RESP error reply, e.g. -ERR ...
func ErrorValue(s string) Value { return Value{Type: Error, Str: s} }

// IntegerValue returns a RESP integer reply.
func IntegerValue(i int64) Value { return Value{Type: Integer, Int: i} }

// BulkStringValue returns a RESP bulk string.
func BulkStringValue(s string) Value { return Value{Type: BulkString, Str: s} }

// NullValue returns a RESP null bulk string ($-1), used for missing keys.
func NullValue() Value { return Value{Type: BulkString, Null: true} }

// ArrayValue returns a RESP array reply.
func ArrayValue(items []Value) Value { return Value{Type: Array, Array: items} }
