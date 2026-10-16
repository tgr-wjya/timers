package ipc

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
)

type HandlerFunc func(req Request) Response

type Server struct {
	socketPath string
	listener   net.Listener
	handler    HandlerFunc
	stopOnce   sync.Once
	stopChan   chan struct{}
	mu         sync.Mutex
	conns      map[net.Conn]struct{}
}

func NewServer(socketPath string, handler HandlerFunc) (*Server, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Remove stale socket if exists
	_ = os.Remove(socketPath)

	l, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}

	s := &Server{
		socketPath: socketPath,
		listener:   l,
		handler:    handler,
		stopChan:   make(chan struct{}),
		conns:      make(map[net.Conn]struct{}),
	}

	go s.serve()

	return s, nil
}

func (s *Server) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopChan:
				return
			default:
				return
			}
		}

		s.mu.Lock()
		s.conns[conn] = struct{}{}
		s.mu.Unlock()

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer func() {
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
		conn.Close()
	}()

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			return
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			resp := Response{Status: "error", Message: "invalid json"}
			_ = json.NewEncoder(conn).Encode(resp)
			return
		}

		resp := s.handler(req)
		_ = json.NewEncoder(conn).Encode(resp)
	}
}

func (s *Server) Close() error {
	var err error
	s.stopOnce.Do(func() {
		close(s.stopChan)
		if s.listener != nil {
			err = s.listener.Close()
		}
		s.mu.Lock()
		for c := range s.conns {
			_ = c.Close()
		}
		s.conns = make(map[net.Conn]struct{})
		s.mu.Unlock()
		_ = os.Remove(s.socketPath)
	})
	return err
}
