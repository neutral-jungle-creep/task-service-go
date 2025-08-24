package server

import (
	"net/http"
	"time"
)

const (
	defaultReadTimeout       = 10 * time.Second
	defaultWriteTimeout      = 10 * time.Second
	defaultIdleTimout        = 60 * time.Second
	defaultReadHeaderTimeout = 10 * time.Second
	defaultAddr              = ":8008"
)

type Config struct {
	ListenPort        string        `json:"apiListenPort"`
	KeepAliveTime     time.Duration `json:"keepAliveTime"`
	KeepAliveTimeout  time.Duration `json:"keepAliveTimeout"`
	ReadHeaderTimeout time.Duration `json:"keepAliveReadHeaderTimeout"`
	ReadTimeout       time.Duration `json:"readTimeout"`
}

func NewServer(handler http.Handler, opts ...Option) *http.Server {
	s := &http.Server{
		Addr:              defaultAddr,
		Handler:           handler,
		IdleTimeout:       defaultIdleTimout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}
