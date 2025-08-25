package server

import (
	"net"
	"net/http"
	"time"
)

type Option func(*http.Server)

func Port(port string) Option {
	return func(s *http.Server) {
		s.Addr = net.JoinHostPort("", port)
	}
}

func ReadTimeout(timeout time.Duration) Option {
	return func(s *http.Server) {
		s.ReadTimeout = timeout * time.Second
	}
}

func WriteTimeout(timeout time.Duration) Option {
	return func(s *http.Server) {
		s.WriteTimeout = timeout * time.Second
	}
}

func IdleTimeout(timeout time.Duration) Option {
	return func(s *http.Server) {
		s.IdleTimeout = timeout * time.Second
	}
}

func ReadHeaderTimeout(timeout time.Duration) Option {
	return func(s *http.Server) {
		s.ReadHeaderTimeout = timeout * time.Second
	}
}
