package revproxy

import (
	"fmt"
	"html"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/lafikl/consistent"
	"github.com/octu0/bp"
)

const DefaultConfigTemplate string = `
{{ with $path := "/port-balance/{id:[0-9]+}" -}}
  {{- with $url := "http://{{ hostport .BASE_IP .BASE_PORT .id }}/api/{{ .id }}" -}}
  {{ HandlePrefix $path (Proxy $url) }}
  {{- end -}}
{{- end }}
{{ with $path := "/consistent-hashing/{key}" -}}
  {{- $url1 := "http://{{ .BASE_IP }}:8081/{{ .key }}/" -}}
  {{- $url2 := "http://{{ .BASE_IP }}:8082/{{ .key }}/" -}}
  {{- $url3 := "http://{{ .BASE_IP }}:8083/{{ .key }}/" -}}
  {{ HandleFunc $path (ProxyConsistent $url1 $url2 $url3) }}
{{- end }}
{{ with $path := "/ok" -}}
  {{ HandleFunc $path (Text 200 "OK") }}
{{- end }}
{{ HandleFunc "/" (Proxy "http://www.google.com/") }}
`

var (
	bufPool = bp.NewBufferPool(1000, 128)
)

type KeyValue map[string]string

type Command struct {
	Success bool
	Type    string
	Handler func(http.ResponseWriter, *http.Request)
}

var (
	CommonFuncMap = template.FuncMap{
		"hostport": func(baseIP, basePort string, value string) string {
			port, _ := strconv.Atoi(basePort)
			v, _ := strconv.Atoi(value)
			return net.JoinHostPort(baseIP, strconv.Itoa(port+v))
		},
		"add": func(a, b string) string {
			i, _ := strconv.Atoi(a)
			j, _ := strconv.Atoi(b)
			return strconv.Itoa(i + j)
		},
		"sub": func(a, b string) string {
			i, _ := strconv.Atoi(a)
			j, _ := strconv.Atoi(b)
			return strconv.Itoa(i - j)
		},
	}
)

func createHandlerFuncMap(router *mux.Router, kv KeyValue, allowHeaders []string) template.FuncMap {
	failureRequest := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`Bad Gateway`))
	}

	proxyRequest := func(t *template.Template, w http.ResponseWriter, r *http.Request) {
		reqVar := mux.Vars(r)
		vars := mergeKeyValue(reqVar, kv)

		buf := bufPool.Get()
		defer bufPool.Put(buf)

		if err := t.Execute(buf, vars); err != nil {
			log.Printf("error: url template error:%s", err.Error())
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`Internal Server Error`))
			return
		}

		u, err := url.Parse(buf.String())
		if err != nil {
			log.Printf("error: url parse error:%s", err.Error())
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`Internal Server Error`))
			return
		}
		proxy := newReverseProxy(r, u, allowHeaders)
		proxy.ServeHTTP(w, r)
	}

	return template.FuncMap{
		"HandleFunc": func(path string, cmd Command, methods ...string) string {
			status := "success"
			if cmd.Success != true {
				status = "failure"
			}

			f := router.HandleFunc(path, cmd.Handler)
			if 0 < len(methods) {
				f.Methods(methods...)
			}
			return fmt.Sprintf("install(%s) route %s = %s", status, path, cmd.Type)
		},
		"HandlePrefix": func(pattern string, cmd Command, methods ...string) string {
			status := "success"
			if cmd.Success != true {
				status = "failure"
			}

			f := router.PathPrefix(pattern).HandlerFunc(cmd.Handler)
			if 0 < len(methods) {
				f.Methods(methods...)
			}
			return fmt.Sprintf("install(%s) prefix %s = %s", status, pattern, cmd.Type)
		},
		"Text": func(statusCode int, txt string) Command {
			return Command{
				Success: true,
				Type:    "text",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/plain")
					w.WriteHeader(statusCode)
					w.Write([]byte(txt))
				},
			}
		},
		"Proxy": func(urlTemplate string) Command {
			t, err := template.New("proxy").Funcs(CommonFuncMap).Parse(urlTemplate)
			if err != nil {
				log.Printf("error: template parse error: %s", err.Error())
				return Command{
					Success: false,
					Type:    "proxy",
					Handler: failureRequest,
				}
			}

			return Command{
				Success: true,
				Type:    "proxy",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					proxyRequest(t, w, r)
				},
			}
		},
		"ProxyConsistent": func(urlTemplates ...string) Command {
			templates := make(map[string]*template.Template, len(urlTemplates))
			c := consistent.New()
			for i, t := range urlTemplates {
				name := fmt.Sprintf("proxy-consistent:%d", i)
				p, err := template.New(name).Funcs(CommonFuncMap).Parse(t)
				if err != nil {
					log.Printf("error: template parse error: %s", err.Error())
					return Command{
						Success: false,
						Type:    "proxy-consistent",
						Handler: failureRequest,
					}
				}

				templates[t] = p
				c.Add(t)
			}
			return Command{
				Success: true,
				Type:    "proxy-consistent",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					tpl, err := c.Get(r.URL.String())
					if err != nil {
						log.Printf("error: consistent Get error:%s", err.Error())
						w.Header().Set("Content-Type", "text/plain")
						w.WriteHeader(http.StatusInternalServerError)
						w.Write([]byte(`Internal Server Error`))
						return
					}

					t := templates[tpl]
					proxyRequest(t, w, r)
				},
			}
		},
	}
}

func mergeKeyValue(src map[string]string, kv KeyValue) map[string]string {
	vars := make(map[string]string, len(src)+len(kv))
	for key, value := range src {
		vars[key] = value
	}
	for key, value := range kv {
		vars[key] = value
	}
	return vars
}

func newReverseProxy(originReq *http.Request, u *url.URL, allowHeaders []string) *httputil.ReverseProxy {
	targetQuery := u.RawQuery
	dst := httputil.NewSingleHostReverseProxy(u)
	dst.Director = func(req *http.Request) {
		req.URL.Scheme = u.Scheme
		req.URL.Host = u.Host
		req.URL.Path = u.Path

		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		for _, h := range allowHeaders {
			if 0 < len(originReq.Header[h]) {
				req.Header.Set(h, strings.Join(originReq.Header[h], ","))
			}
		}
	}
	dst.BufferPool = &proxyBufferPool{proxyPool}
	return dst
}

func applyTemplate(tpl string, kv KeyValue, router *mux.Router, allowHeaders []string) error {
	buf := bufPool.Get()
	defer bufPool.Put(buf)

	handlerFuncMap := createHandlerFuncMap(router, kv, allowHeaders)
	for name, fn := range CommonFuncMap {
		handlerFuncMap[name] = fn
	}
	for name, _ := range handlerFuncMap {
		log.Printf("debug: use func %s", name)
	}

	t := strings.TrimSpace(tpl)
	tp := template.Must(template.New("base").Funcs(handlerFuncMap).Parse(t))
	if err := tp.Execute(buf, kv); err != nil {
		return err
	}

	err := router.Walk(func(r *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := r.GetPathTemplate()
		if err == nil {
			log.Printf("debug: ROUTE:%s", pathTemplate)
		}
		pathRegexp, err := r.GetPathRegexp()
		if err == nil {
			log.Printf("debug: Path regexp:%s", pathRegexp)
		}
		queriesTemplates, err := r.GetQueriesTemplates()
		if err == nil {
			log.Printf("debug: Queries templates:%s", strings.Join(queriesTemplates, ","))
		}
		queriesRegexps, err := r.GetQueriesRegexp()
		if err == nil {
			log.Printf("debug: Queries regexps:%s", strings.Join(queriesRegexps, ","))
		}
		methods, err := r.GetMethods()
		if err == nil {
			log.Printf("debug: Methods:%s", strings.Join(methods, ","))
		}
		return nil
	})
	if err != nil {
		return err
	}

	log.Printf("trace: load config template: %s", t)
	log.Printf("info: \n%s\n", html.UnescapeString(buf.String()))
	return nil
}

var (
	proxyPool = bp.NewBytePool(1000, 4*1024)
)

// compile check
var (
	_ httputil.BufferPool = (*proxyBufferPool)(nil)
)

type proxyBufferPool struct {
	pool *bp.BytePool
}

func (p *proxyBufferPool) Get() []byte {
	return p.pool.Get()
}
func (p *proxyBufferPool) Put(d []byte) {
	p.pool.Put(d)
}
