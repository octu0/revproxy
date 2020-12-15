package main

import (
	"context"
	"io/ioutil"
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
	rto := c.Int("http-read-timeout")
	wto := c.Int("http-write-timeout")
	ito := c.Int("http-idle-timeout")
	rhto := c.Int("http-read-header-timeout")

	cfg, err := loadConfigTemplate(c.String("template"))
	if err != nil {
		log.Printf("error: failed to load template('%s') err:%s", c.String("template"), err)
		return err
	}

	kv := make(revproxy.KeyValue)
	for _, vv := range kvs {
		keyvalue := strings.Split(vv, "=")
		if len(keyvalue) < 1 {
			continue
		}

		key, value := keyvalue[0], keyvalue[1]
		kv[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if len(kv) < 1 {
		log.Printf("info: empty kv parameter")
	}

	sctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go watchSignal(cancel)

	svr := revproxy.NewServer(
		revproxy.ListenAddr(net.JoinHostPort(host, strconv.Itoa(port))),
		revproxy.AllowHeaders(headers),
		revproxy.ReadTimeout(time.Duration(rto)*time.Second),
		revproxy.WriteTimeout(time.Duration(wto)*time.Second),
		revproxy.IdleTimeout(time.Duration(ito)*time.Second),
		revproxy.ReadHeaderTimeout(time.Duration(rhto)*time.Second),
	)

	log.Printf("info: server starting...")

	go svr.Start(cfg, kv)

	<-sctx.Done() // wait stop

	log.Printf("info: server stopping...")

	t, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	svr.Stop(t)
	log.Printf("info: server stop")
	return nil
}

func loadConfigTemplate(path string) (string, error) {
	if path == "" {
		return revproxy.DefaultConfigTemplate, nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path, nil
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
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
				Usage:  "/path/to.tpl reverse proxy router template value or template file path",
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
			cli.IntFlag{
				Name:  "http-read-timeout",
				Usage: "http server read timeout(time-unit: second)",
				Value: 10,
			},
			cli.IntFlag{
				Name:  "http-write-timeout",
				Usage: "http server write timeout(time-unit: second)",
				Value: 10,
			},
			cli.IntFlag{
				Name:  "http-idle-timeout",
				Usage: "http server idle timeout(time-unit: second)",
				Value: 30,
			},
			cli.IntFlag{
				Name:  "http-read-header-timeout",
				Usage: "http server read header timeout(time-unit: second)",
				Value: 15,
			},
		},
		Action: serverAction,
	})
}
