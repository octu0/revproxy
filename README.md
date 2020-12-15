# `revproxy`

[![Apache License](https://img.shields.io/github/license/octu0/revproxy)](https://github.com/octu0/revproxy/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/octu0/revproxy?status.svg)](https://godoc.org/github.com/octu0/revproxy)
[![Go Report Card](https://goreportcard.com/badge/github.com/octu0/revproxy)](https://goreportcard.com/report/github.com/octu0/revproxy)
[![Releases](https://img.shields.io/github/v/release/octu0/revproxy)](https://github.com/octu0/revproxy/releases)

`revproxy` provides simple and configurable reverse proxy.

features:
- [httputil.ReverseProxy](https://golang.org/pkg/net/http/httputil/#ReverseProxy) base HTTP Proxy Server
- Simple Programable configure
- Port mapping Proxy
- URL based Consistent Hashing Proxy

## Download

Linux amd64 / Darwin amd64 binaries are available in [Releases](https://github.com/octu0/revproxy/releases)

## Build

Build requires Go version 1.13+ installed.

```shell
$ go version
```

Run `make pkg` to Build and package for linux, darwin.

```shell
$ git clone https://github.com/octu0/revproxy
$ make pkg
```

# Configure

## Simple Proxy

To proxy a specific Path, use HandleFunc and Proxy functions

```
{{ HandleFunc "/ok" (Text 200 "OK") }}
{{ HandleFunc "/foobar1" (Proxy "http://targethost:8080/foobar2") }}
{{ HandleFunc "/" (Proxy "http://www.google.com/") }}
```

In this configuration, proxy is performed as below:

```
- http://localhost:8080/         => proxy http://www.google.com/
- http://localhost:8080/foobar1  => proxy http://targethost:8080/foobar2
- http://localhost:8080/ok       => response "OK"
```

## Port mapping Proxy

using the built-in functions and the key/value values set with `--value` when `revproxy` is started,
proxy specify the port number.

Specify HandlePrefix to make all paths under specific path to be proxy.

```
{{ with $path := "/port-balance/{id:[0-9]+}" -}}
  {{- with $url := "http://{{ hostport .BASE_IP .BASE_PORT .id }}/api/{{ .id }}" -}}
  {{ HandlePrefix $path (Proxy $url) }}
  {{- end -}}
{{- end }}
```

In this configuration, proxy is performed as below:

```
- http://localhost:8080/port-balance/123 => proxy http://localhost:8123/api/123
- http://localhost:8080/port-balance/456 => proxy http://localhost:8456/api/456
```

### Example

Try launching it as below:

```shell
$ revproxy server --value BASE_IP=localhost --value BASE_PORT=8000 -t '{{ with $path := "/port-balance/{id:[0-9]+}" -}}
  {{- with $url := "http://{{ hostport .BASE_IP .BASE_PORT .id }}/api/{{ .id }}" -}}
  {{ HandlePrefix $path (Proxy $url) }}
  {{- end -}}
{{- end }}'
```

Destination server started as below:

```shell
# server1
$ revproxy server --port 8123 -t '{{ HandlePrefix "/api/123" (Text 200 "i am :8123/api/123") }}' &

# server2
$ revproxy server --port 8456 -t '{{ HandlePrefix "/api/456" (Text 200 "i am :8456/api/456") }}' &
```

Try to get the requests distributed.

```shell
$ curl -XGET localhost:8080/port-balance/123
i am :8123/api/123

$ curl -XGET localhost:8080/port-balance/456
i am :8456/api/456
```

## Consistent Hashing Proxy

Proxy routing by path-based consistent hashing is defined as below:

```
{{ with $path := "/consistent-hashing/{key}" -}}
  {{- $url1 := "http://{{ .BASE_IP }}:8081/{{ .key }}/" -}}
  {{- $url2 := "http://{{ .BASE_IP }}:8082/{{ .key }}/" -}}
  {{- $url3 := "http://{{ .BASE_IP }}:8083/{{ .key }}/" -}}
  {{ HandleFunc $path (ProxyConsistent $url1 $url2 $url3) }}
{{- end }}
```

This will result in the following distribution.

```
- http://localhost:8080/consistent-hashing/a => proxy http://localhost:8081/a
- http://localhost:8080/consistent-hashing/b => proxy http://localhost:8082/b
- http://localhost:8080/consistent-hashing/c => proxy http://localhost:8083/c
- http://localhost:8080/consistent-hashing/b => proxy http://localhost:8081/b
```

### Example

Try launching it as below:

```shell
### proxy server
$ revproxy server --value BASE_IP=localhost -t '{{ with $path := "/consistent-hashing/{key}" -}}
>   {{- $url1 := "http://{{ .BASE_IP }}:8081/{{ .key }}/" -}}
>   {{- $url2 := "http://{{ .BASE_IP }}:8082/{{ .key }}/" -}}
>   {{- $url3 := "http://{{ .BASE_IP }}:8083/{{ .key }}/" -}}
>   {{ HandleFunc $path (ProxyConsistent $url1 $url2 $url3) }}
> {{- end }}'
```

Destination server started as below:

```shell
# server1
$ revproxy server --port 8081 -t '{{ HandlePrefix "/" (Text 200 "i am 8081") }}' &

# server2
$ revproxy server --port 8082 -t '{{ HandlePrefix "/" (Text 200 "i am 8082") }}' &

# server2
$ revproxy server --port 8083 -t '{{ HandlePrefix "/" (Text 200 "i am 8083") }}' &
```

Try to get the requests distributed.

```shell
$ curl -XGET localhost:8080/consistent-hashing/a
i am 8081

$ curl -XGET localhost:8080/consistent-hashing/b
i am 8083

$ curl -XGET localhost:8080/consistent-hashing/c
i am 8082

$ curl -XGET localhost:8080/consistent-hashing/a
i am 8081
```

## Help

```
NAME:
   revproxy server

USAGE:
   revproxy server [command options] [arguments...]

OPTIONS:
   --host value                      bind host (default: "[0.0.0.0]") [$REVPROXY_HOST]
   --port value                      bind port (default: 8080) [$REVPROXY_PORT]
   -t value, --template value        /path/to.tpl reverse proxy router template value or template file path [$REVPROXY_TEMPLATE]
   --header value                    allow headers
   --value value, -v value           specify variables to be applied to template, format KEY=Value (e.g. IP=10.16.0.2)
   --http-read-timeout value         http server read timeout(time-unit: second) (default: 10)
   --http-write-timeout value        http server write timeout(time-unit: second) (default: 10)
   --http-idle-timeout value         http server idle timeout(time-unit: second) (default: 30)
   --http-read-header-timeout value  http server read header timeout(time-unit: second) (default: 15)
```

## License

Apache 2.0, see LICENSE file for details.
