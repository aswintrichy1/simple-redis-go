// Package client is a small RESP client for the simple-redis-go server. It
// offers typed helpers for the supported commands plus a generic Do method for
// arbitrary commands.
package client

import (
	"errors"
	"net"
	"strconv"

	"simple-redis-go/internal/resp"
)

// Client is a synchronous connection to a simple-redis-go server. It is not
// safe for concurrent use by multiple goroutines.
type Client struct {
	conn   net.Conn
	reader *resp.Reader
	writer *resp.Writer
}

// ZMember is a scored member of a sorted set.
type ZMember struct {
	Score  float64
	Member string
}

// Dial connects to the server at addr.
func Dial(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:   conn,
		reader: resp.NewReader(conn),
		writer: resp.NewWriter(conn),
	}, nil
}

// Close closes the underlying connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Do sends a command and returns the raw RESP reply. Error replies are returned
// as values (Type == resp.Error), not Go errors; the transport error is
// returned separately.
func (c *Client) Do(args ...string) (resp.Value, error) {
	if err := c.writer.WriteCommand(args...); err != nil {
		return resp.Value{}, err
	}
	return c.reader.ReadValue()
}

// Get returns the value at key. The boolean reports whether the key exists.
func (c *Client) Get(key string) (string, bool, error) {
	v, err := c.Do("GET", key)
	if err != nil {
		return "", false, err
	}
	if err := asError(v); err != nil {
		return "", false, err
	}
	if v.Null {
		return "", false, nil
	}
	return v.Str, true, nil
}

// Set stores value at key.
func (c *Client) Set(key, value string) error {
	v, err := c.Do("SET", key, value)
	if err != nil {
		return err
	}
	return asError(v)
}

// Del deletes keys and returns the number removed.
func (c *Client) Del(keys ...string) (int64, error) {
	v, err := c.Do(append([]string{"DEL"}, keys...)...)
	if err != nil {
		return 0, err
	}
	if err := asError(v); err != nil {
		return 0, err
	}
	return v.Int, nil
}

// Expire sets a TTL of seconds on key, returning true if it was applied.
func (c *Client) Expire(key string, seconds int64) (bool, error) {
	v, err := c.Do("EXPIRE", key, strconv.FormatInt(seconds, 10))
	if err != nil {
		return false, err
	}
	if err := asError(v); err != nil {
		return false, err
	}
	return v.Int == 1, nil
}

// TTL returns the remaining TTL in seconds (-1 no expiry, -2 missing key).
func (c *Client) TTL(key string) (int64, error) {
	v, err := c.Do("TTL", key)
	if err != nil {
		return 0, err
	}
	if err := asError(v); err != nil {
		return 0, err
	}
	return v.Int, nil
}

// Keys returns all keys matching the glob pattern.
func (c *Client) Keys(pattern string) ([]string, error) {
	v, err := c.Do("KEYS", pattern)
	if err != nil {
		return nil, err
	}
	if err := asError(v); err != nil {
		return nil, err
	}
	return bulkStrings(v), nil
}

// ZAdd adds members to the sorted set at key, returning the number newly added.
func (c *Client) ZAdd(key string, members ...ZMember) (int64, error) {
	args := []string{"ZADD", key}
	for _, m := range members {
		args = append(args, strconv.FormatFloat(m.Score, 'g', -1, 64), m.Member)
	}
	v, err := c.Do(args...)
	if err != nil {
		return 0, err
	}
	if err := asError(v); err != nil {
		return 0, err
	}
	return v.Int, nil
}

// ZRange returns members in the inclusive index range [start, stop]. When
// withScores is true, the result interleaves members and their scores.
func (c *Client) ZRange(key string, start, stop int, withScores bool) ([]string, error) {
	args := []string{"ZRANGE", key, strconv.Itoa(start), strconv.Itoa(stop)}
	if withScores {
		args = append(args, "WITHSCORES")
	}
	v, err := c.Do(args...)
	if err != nil {
		return nil, err
	}
	if err := asError(v); err != nil {
		return nil, err
	}
	return bulkStrings(v), nil
}

func asError(v resp.Value) error {
	if v.Type == resp.Error {
		return errors.New(v.Str)
	}
	return nil
}

func bulkStrings(v resp.Value) []string {
	out := make([]string, len(v.Array))
	for i, item := range v.Array {
		out[i] = item.Str
	}
	return out
}
