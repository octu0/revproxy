package revproxy

import (
	"time"
)

type serverOptFunc func(*serverOpt)

type serverOpt struct {
	listenAddr   string
	allowHeaders []string
	rto          time.Duration
	wto          time.Duration
	ito          time.Duration
	rhto         time.Duration
}

func ListenAddr(addr string) serverOptFunc {
	return func(opt *serverOpt) {
		opt.listenAddr = addr
	}
}

func AllowHeaders(headers []string) serverOptFunc {
	return func(opt *serverOpt) {
		opt.allowHeaders = headers
	}
}

func ReadTimeout(dur time.Duration) serverOptFunc {
	return func(opt *serverOpt) {
		opt.rto = dur
	}
}

func WriteTimeout(dur time.Duration) serverOptFunc {
	return func(opt *serverOpt) {
		opt.wto = dur
	}
}

func IdleTimeout(dur time.Duration) serverOptFunc {
	return func(opt *serverOpt) {
		opt.ito = dur
	}
}

func ReadHeaderTimeout(dur time.Duration) serverOptFunc {
	return func(opt *serverOpt) {
		opt.rhto = dur
	}
}

func initOpt(opt *serverOpt) {
	if len(opt.listenAddr) < 1 {
		opt.listenAddr = "[0.0.0.0]:8080"
	}
	if opt.rto < 1 {
		opt.rto = 60 * time.Second
	}
	if opt.wto < 1 {
		opt.wto = 60 * time.Second
	}
	if opt.ito < 1 {
		opt.wto = 60 * time.Second
	}
	if opt.rhto < 1 {
		opt.wto = 10 * time.Second
	}
}
