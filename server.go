package revproxy

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/mux"
)

type server struct {
	opt    *serverOpt
	router *mux.Router
	svr    *http.Server
}

func NewServer(funcs ...serverOptFunc) *server {
	opt := new(serverOpt)
	for _, fn := range funcs {
		fn(opt)
	}
	initOpt(opt)

	router := mux.NewRouter()
	svr := &http.Server{
		Handler:           router,
		ReadTimeout:       opt.rto,
		WriteTimeout:      opt.wto,
		IdleTimeout:       opt.ito,
		ReadHeaderTimeout: opt.rhto,
	}
	return &server{
		opt:    opt,
		router: router,
		svr:    svr,
	}
}

func (s *server) Start(template string, kv KeyValue) error {
	if err := applyTemplate(template, kv, s.router, s.opt.allowHeaders); err != nil {
		return err
	}
	listener, err := net.Listen("tcp", s.opt.listenAddr)
	if err != nil {
		return err
	}
	log.Printf("info: server listen %s/tcp", s.opt.listenAddr)
	return s.svr.Serve(listener)
}

func (s *server) Stop(ctx context.Context) error {
	return s.svr.Shutdown(ctx)
}
