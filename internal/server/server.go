// Package server exposes the in-memory store over TCP using the RESP protocol.
// Each client connection is handled by its own goroutine; concurrency safety is
// provided by the underlying store.
package server

import (
	"errors"
	"io"
	"net"

	"simple-redis-go/internal/command"
	"simple-redis-go/internal/resp"
	"simple-redis-go/internal/store"
)

// Server accepts RESP client connections and serves commands from a Store.
type Server struct {
	handler  *command.Handler
	listener net.Listener
}

// New returns a Server backed by s.
func New(s *store.Store) *Server {
	return &Server{handler: command.NewHandler(s)}
}

// Bind starts listening on addr without accepting connections. It is useful for
// tests that need the resolved address (e.g. when binding to port 0) before
// serving.
func (s *Server) Bind(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.listener = ln
	return nil
}

// Addr returns the address the server is listening on, or "" if not bound.
func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Serve accepts connections until the listener is closed. It returns nil on a
// clean shutdown via Close.
func (s *Server) Serve() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		go s.handleConn(conn)
	}
}

// Listen binds to addr and serves until the server is closed.
func (s *Server) Listen(addr string) error {
	if err := s.Bind(addr); err != nil {
		return err
	}
	return s.Serve()
}

// Close stops the server and unblocks Serve.
func (s *Server) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	reader := resp.NewReader(conn)
	writer := resp.NewWriter(conn)

	for {
		args, err := reader.ReadCommand()
		if err != nil {
			if err == io.EOF || errors.Is(err, net.ErrClosed) {
				return
			}
			// Report protocol errors to the client, then close the connection
			// since the stream can no longer be reliably framed.
			_ = writer.WriteValue(resp.ErrorValue("ERR " + err.Error()))
			return
		}
		if len(args) == 0 {
			continue
		}
		if err := writer.WriteValue(s.handler.Execute(args)); err != nil {
			return
		}
	}
}
