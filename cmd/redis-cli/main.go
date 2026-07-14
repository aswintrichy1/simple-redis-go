// Command redis-cli is a minimal client for simple-redis-go. It runs a single
// command supplied as arguments, or starts an interactive prompt when none are
// given.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"simple-redis-go/client"
	"simple-redis-go/internal/resp"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:6379", "server address (host:port)")
	flag.Parse()

	c, err := client.Dial(*addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not connect to %s: %v\n", *addr, err)
		os.Exit(1)
	}
	defer c.Close()

	if args := flag.Args(); len(args) > 0 {
		runOnce(c, args)
		return
	}
	runInteractive(c, *addr)
}

func runOnce(c *client.Client, args []string) {
	reply, err := c.Do(args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(formatReply(reply))
}

func runInteractive(c *client.Client, addr string) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("%s> ", addr)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			args, err := tokenize(line)
			switch {
			case err != nil:
				fmt.Printf("(error) %v\n", err)
			case strings.EqualFold(args[0], "quit"), strings.EqualFold(args[0], "exit"):
				return
			default:
				reply, err := c.Do(args...)
				if err != nil {
					fmt.Printf("(error) %v\n", err)
				} else {
					fmt.Println(formatReply(reply))
				}
			}
		}
		fmt.Printf("%s> ", addr)
	}
}

// formatReply renders a RESP value in a readable, redis-cli-like form.
func formatReply(v resp.Value) string {
	switch v.Type {
	case resp.SimpleString:
		return v.Str
	case resp.Error:
		return "(error) " + v.Str
	case resp.Integer:
		return fmt.Sprintf("(integer) %d", v.Int)
	case resp.BulkString:
		if v.Null {
			return "(nil)"
		}
		return fmt.Sprintf("%q", v.Str)
	case resp.Array:
		if v.Null {
			return "(nil)"
		}
		if len(v.Array) == 0 {
			return "(empty array)"
		}
		lines := make([]string, len(v.Array))
		for i, item := range v.Array {
			lines[i] = fmt.Sprintf("%d) %s", i+1, formatReply(item))
		}
		return strings.Join(lines, "\n")
	default:
		return ""
	}
}

// tokenize splits a command line into arguments, honoring single and double
// quotes so values may contain spaces.
func tokenize(line string) ([]string, error) {
	var (
		tokens             []string
		cur                strings.Builder
		inSingle, inDouble bool
		hasToken           bool
	)
	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch {
		case inSingle:
			if ch == '\'' {
				inSingle = false
			} else {
				cur.WriteByte(ch)
			}
		case inDouble:
			switch {
			case ch == '"':
				inDouble = false
			case ch == '\\' && i+1 < len(line):
				i++
				cur.WriteByte(line[i])
			default:
				cur.WriteByte(ch)
			}
		case ch == '\'':
			inSingle, hasToken = true, true
		case ch == '"':
			inDouble, hasToken = true, true
		case ch == ' ' || ch == '\t':
			if hasToken {
				tokens = append(tokens, cur.String())
				cur.Reset()
				hasToken = false
			}
		default:
			cur.WriteByte(ch)
			hasToken = true
		}
	}
	if inSingle || inDouble {
		return nil, fmt.Errorf("unbalanced quotes")
	}
	if hasToken {
		tokens = append(tokens, cur.String())
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	return tokens, nil
}
