package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

// Limits on client-declared lengths, guarding against a malicious client that
// declares a huge bulk string or array to force an unbounded allocation. The
// values mirror Redis's proto-max-bulk-len (512 MB) and multibulk element cap.
const (
	maxBulkLength  = 512 * 1024 * 1024
	maxArrayLength = 1024 * 1024
)

// Reader decodes RESP2 values from an underlying stream. A single Reader must
// be reused for the lifetime of a connection so that buffered bytes are not
// lost between reads.
type Reader struct {
	r *bufio.Reader
}

// NewReader wraps rd in a buffered RESP reader.
func NewReader(rd io.Reader) *Reader {
	return &Reader{r: bufio.NewReader(rd)}
}

// ReadValue reads and decodes a single RESP value.
func (r *Reader) ReadValue() (Value, error) {
	typ, err := r.r.ReadByte()
	if err != nil {
		return Value{}, err
	}

	switch Type(typ) {
	case SimpleString:
		line, err := r.readLine()
		if err != nil {
			return Value{}, err
		}
		return Value{Type: SimpleString, Str: string(line)}, nil
	case Error:
		line, err := r.readLine()
		if err != nil {
			return Value{}, err
		}
		return Value{Type: Error, Str: string(line)}, nil
	case Integer:
		n, err := r.readInt()
		if err != nil {
			return Value{}, err
		}
		return Value{Type: Integer, Int: n}, nil
	case BulkString:
		return r.readBulkString()
	case Array:
		return r.readArray()
	default:
		return Value{}, fmt.Errorf("protocol error: unexpected type byte %q", string(typ))
	}
}

// ReadCommand reads a client command, which RESP encodes as an array of bulk
// strings, and returns the arguments as plain strings.
func (r *Reader) ReadCommand() ([]string, error) {
	v, err := r.ReadValue()
	if err != nil {
		return nil, err
	}
	if v.Type != Array || v.Null {
		return nil, fmt.Errorf("protocol error: expected array")
	}
	args := make([]string, len(v.Array))
	for i, item := range v.Array {
		if item.Type != BulkString || item.Null {
			return nil, fmt.Errorf("protocol error: expected bulk string argument")
		}
		args[i] = item.Str
	}
	return args, nil
}

func (r *Reader) readBulkString() (Value, error) {
	n, err := r.readInt()
	if err != nil {
		return Value{}, err
	}
	if n == -1 {
		return Value{Type: BulkString, Null: true}, nil
	}
	if n < 0 || n > maxBulkLength {
		return Value{}, fmt.Errorf("protocol error: invalid bulk length %d", n)
	}
	buf := make([]byte, n+2) // include trailing CRLF
	if _, err := io.ReadFull(r.r, buf); err != nil {
		return Value{}, err
	}
	if buf[n] != '\r' || buf[n+1] != '\n' {
		return Value{}, fmt.Errorf("protocol error: bulk string not terminated by CRLF")
	}
	return Value{Type: BulkString, Str: string(buf[:n])}, nil
}

func (r *Reader) readArray() (Value, error) {
	n, err := r.readInt()
	if err != nil {
		return Value{}, err
	}
	if n == -1 {
		return Value{Type: Array, Null: true}, nil
	}
	if n < 0 || n > maxArrayLength {
		return Value{}, fmt.Errorf("protocol error: invalid array length %d", n)
	}
	items := make([]Value, n)
	for i := int64(0); i < n; i++ {
		v, err := r.ReadValue()
		if err != nil {
			return Value{}, err
		}
		items[i] = v
	}
	return Value{Type: Array, Array: items}, nil
}

// readLine reads up to and including CRLF and returns the content without the
// trailing CRLF.
func (r *Reader) readLine() ([]byte, error) {
	line, err := r.r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(line) < 2 || line[len(line)-2] != '\r' {
		return nil, fmt.Errorf("protocol error: line not terminated by CRLF")
	}
	return line[:len(line)-2], nil
}

// readInt reads a CRLF-terminated integer line.
func (r *Reader) readInt() (int64, error) {
	line, err := r.readLine()
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("protocol error: invalid integer %q", string(line))
	}
	return n, nil
}
