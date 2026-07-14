package resp

import (
	"bufio"
	"io"
	"strconv"
)

// Writer encodes RESP2 values onto an underlying stream.
type Writer struct {
	w *bufio.Writer
}

// NewWriter wraps w in a buffered RESP writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: bufio.NewWriter(w)}
}

// WriteValue encodes v and flushes it to the underlying stream.
func (w *Writer) WriteValue(v Value) error {
	if err := w.encode(v); err != nil {
		return err
	}
	return w.w.Flush()
}

// WriteCommand encodes args as a RESP array of bulk strings and flushes it.
// It is used by the client to send commands to the server.
func (w *Writer) WriteCommand(args ...string) error {
	items := make([]Value, len(args))
	for i, a := range args {
		items[i] = BulkStringValue(a)
	}
	return w.WriteValue(ArrayValue(items))
}

func (w *Writer) encode(v Value) error {
	switch v.Type {
	case SimpleString:
		return w.writeLine('+', v.Str)
	case Error:
		return w.writeLine('-', v.Str)
	case Integer:
		return w.writeLine(':', strconv.FormatInt(v.Int, 10))
	case BulkString:
		return w.encodeBulk(v)
	case Array:
		return w.encodeArray(v)
	default:
		return w.writeLine('-', "ERR unknown reply type")
	}
}

func (w *Writer) encodeBulk(v Value) error {
	if v.Null {
		_, err := w.w.WriteString("$-1\r\n")
		return err
	}
	if _, err := w.w.WriteString("$" + strconv.Itoa(len(v.Str)) + "\r\n"); err != nil {
		return err
	}
	if _, err := w.w.WriteString(v.Str); err != nil {
		return err
	}
	_, err := w.w.WriteString("\r\n")
	return err
}

func (w *Writer) encodeArray(v Value) error {
	if v.Null {
		_, err := w.w.WriteString("*-1\r\n")
		return err
	}
	if _, err := w.w.WriteString("*" + strconv.Itoa(len(v.Array)) + "\r\n"); err != nil {
		return err
	}
	for _, item := range v.Array {
		if err := w.encode(item); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) writeLine(prefix byte, s string) error {
	if err := w.w.WriteByte(prefix); err != nil {
		return err
	}
	if _, err := w.w.WriteString(s); err != nil {
		return err
	}
	_, err := w.w.WriteString("\r\n")
	return err
}
