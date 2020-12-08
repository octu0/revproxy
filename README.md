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

**Work In Progress**

## Installation

```bash
go get github.com/octu0/revproxy
```


## Configure

### Simple Proxy

```
{{ HandleFunc "/ok" (Text 200 "OK") }}
{{ HandleFunc "/" (Proxy "http://www.google.com/") }}
```

- http://localhost:8080/    => proxy www.google.com
- http://localhost:8080/ok  => response "OK"

### Port mapping Proxy

```
{{ with $path := "/port-balance/{id:[0-9]+}" }}
  {{- with $url := "http://{{ hostport .BASE_IP .BASE_PORT .id }}/api/{{ .id }}" -}}
  {{ HandlePrefix $path (Proxy $url) }}
  {{- end -}}
{{ end }}
```

- http://localhost:8080/port-balance/123 => proxy http://localhost:8123/api/123
- http://localhost:8080/port-balance/456 => proxy http://localhost:8456/api/456

### Consistent Hashing Proxy

```
{{ with $path := "/consistent-hashing/{key}" }}
  {{- with $url1 := "http://{{ .BASE_IP }}:8081/{{ .key }}/" -}}
  {{- with $url2 := "http://{{ .BASE_IP }}:8082/{{ .key }}/" -}}
  {{- with $url3 := "http://{{ .BASE_IP }}:8083/{{ .key }}/" -}}
  {{ HandleFunc $path (ProxyConsistent $url1 $url2 $url3) }}
  {{- end -}}
{{ end }}
```

- http://localhost/consistent-hashing/foo1 => proxy http://localhost:8081/foo1
- http://localhost/consistent-hashing/foo2 => proxy http://localhost:8082/foo2
- http://localhost/consistent-hashing/foo3 => proxy http://localhost:8083/foo3
- http://localhost/consistent-hashing/foo4 => proxy http://localhost:8083/foo3

## License

Apache 2.0, see LICENSE file for details.
