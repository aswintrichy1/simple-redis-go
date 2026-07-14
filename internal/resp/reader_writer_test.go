package resp

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteReadRoundTrip(t *testing.T) {
	cases := []Value{
		SimpleStringValue("OK"),
		ErrorValue("ERR bad"),
		IntegerValue(42),
		IntegerValue(-7),
		BulkStringValue("hello"),
		BulkStringValue(""),
		NullValue(),
		ArrayValue([]Value{BulkStringValue("a"), BulkStringValue("b")}),
		ArrayValue(nil),
		ArrayValue([]Value{IntegerValue(1), NullValue(), SimpleStringValue("x")}),
	}
	for _, want := range cases {
		var buf bytes.Buffer
		if err := NewWriter(&buf).WriteValue(want); err != nil {
			t.Fatalf("WriteValue(%+v): %v", want, err)
		}
		got, err := NewReader(&buf).ReadValue()
		if err != nil {
			t.Fatalf("ReadValue: %v", err)
		}
		if !valuesEqual(got, want) {
			t.Errorf("round trip mismatch: got %+v want %+v", got, want)
		}
	}
}

func TestReadCommand(t *testing.T) {
	raw := "*3\r\n$3\r\nSET\r\n$5\r\nmykey\r\n$5\r\nhello\r\n"
	args, err := NewReader(strings.NewReader(raw)).ReadCommand()
	if err != nil {
		t.Fatalf("ReadCommand: %v", err)
	}
	want := []string{"SET", "mykey", "hello"}
	if len(args) != len(want) {
		t.Fatalf("got %v want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("arg %d: got %q want %q", i, args[i], want[i])
		}
	}
}

func TestWriteCommand(t *testing.T) {
	var buf bytes.Buffer
	if err := NewWriter(&buf).WriteCommand("GET", "key"); err != nil {
		t.Fatal(err)
	}
	want := "*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n"
	if buf.String() != want {
		t.Errorf("got %q want %q", buf.String(), want)
	}
}

func TestEncodingBytes(t *testing.T) {
	tests := []struct {
		value Value
		want  string
	}{
		{SimpleStringValue("OK"), "+OK\r\n"},
		{ErrorValue("ERR x"), "-ERR x\r\n"},
		{IntegerValue(10), ":10\r\n"},
		{BulkStringValue("hello"), "$5\r\nhello\r\n"},
		{NullValue(), "$-1\r\n"},
		{ArrayValue([]Value{BulkStringValue("one"), BulkStringValue("two")}), "*2\r\n$3\r\none\r\n$3\r\ntwo\r\n"},
	}
	for _, tt := range tests {
		var buf bytes.Buffer
		if err := NewWriter(&buf).WriteValue(tt.value); err != nil {
			t.Fatalf("WriteValue: %v", err)
		}
		if buf.String() != tt.want {
			t.Errorf("encode %+v = %q, want %q", tt.value, buf.String(), tt.want)
		}
	}
}

func TestReadValueErrors(t *testing.T) {
	cases := []string{
		"!bad\r\n",           // unknown type byte
		"$5\r\nhi\r\n",       // bulk shorter than declared length
		"$2\r\nhixx",         // bulk not terminated by CRLF
		"*1\r\n$3\r\nab\r\n", // array element too short
		":notanumber\r\n",    // invalid integer
	}
	for _, in := range cases {
		if _, err := NewReader(strings.NewReader(in)).ReadValue(); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}
}

func TestReadValueRejectsOversizedLengths(t *testing.T) {
	// Declared lengths exceeding the limits must be rejected before any
	// allocation, guarding against memory-exhaustion attacks.
	cases := []string{
		"$536870913\r\n", // one byte over the 512 MB bulk limit
		"*1048577\r\n",   // one element over the array limit
	}
	for _, in := range cases {
		if _, err := NewReader(strings.NewReader(in)).ReadValue(); err == nil {
			t.Errorf("expected error for oversized length %q", in)
		}
	}
}

func valuesEqual(a, b Value) bool {
	if a.Type != b.Type || a.Null != b.Null {
		return false
	}
	switch a.Type {
	case Integer:
		return a.Int == b.Int
	case Array:
		if len(a.Array) != len(b.Array) {
			return false
		}
		for i := range a.Array {
			if !valuesEqual(a.Array[i], b.Array[i]) {
				return false
			}
		}
		return true
	default:
		return a.Str == b.Str
	}
}
