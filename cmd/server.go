package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/urfave/cli.v1"

	"github.com/octu0/revproxy"
)

func serverAction(c *cli.Context) error {
	ctx, err := prepareAction(c.Command.Name, c)
	if err != nil {
		return err
	}

	host := c.String("host")
	port := c.Int("port")
	headers := c.StringSlice("header")
	kvs := c.StringSlice("value")
	_ = c.String("template")

	kv := make(revproxy.KeyValue)
	for _, vv := range kvs {
		keyvalue := strings.Split(vv, "=")
		if len(keyvalue) < 1 {
			continue
		}

		key, value := keyvalue[0], keyvalue[1]
		kv[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	sctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go watchSignal(cancel)

	svr := revproxy.NewServer(
		revproxy.ListenAddr(net.JoinHostPort(host, strconv.Itoa(port))),
		revproxy.AllowHeaders(headers),
	)

	log.Printf("info: server start")

	go svr.Start("", nil)

	run := true
	for run {
		select {
		case <-sctx.Done():
			run = false
		}
	}
	t, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	svr.Stop(t)
	log.Printf("info: server stop")
	return nil
}

func watchSignal(cancel context.CancelFunc) {
	trap := make(chan os.Signal)
	signal.Notify(trap, syscall.SIGTERM)
	signal.Notify(trap, syscall.SIGHUP)
	signal.Notify(trap, syscall.SIGQUIT)
	signal.Notify(trap, syscall.SIGINT)

	for {
		select {
		case sig := <-trap:
			log.Printf("info: signal trap(%s)", sig.String())
			switch sig {
			case syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
				cancel()
				return
			}
		}
	}
}

func init() {
	addCommand(cli.Command{
		Name: "server",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "host",
				Usage:  "bind host",
				Value:  "[0.0.0.0]",
				EnvVar: "REVPROXY_HOST",
			},
			cli.IntFlag{
				Name:   "port",
				Usage:  "bind port",
				Value:  8080,
				EnvVar: "REVPROXY_PORT",
			},
			cli.StringFlag{
				Name:   "t, template",
				Usage:  "/path/to.tpl reverse proxy router template path",
				Value:  "",
				EnvVar: "REVPROXY_TEMPLATE",
			},
			cli.StringSliceFlag{
				Name:  "header",
				Usage: "allow headers",
			},
			cli.StringSliceFlag{
				Name:  "value, v",
				Usage: "specify variables to be applied to template, format KEY=Value (e.g. IP=10.16.0.2)",
			},
		},
		Action: serverAction,
	})
}
