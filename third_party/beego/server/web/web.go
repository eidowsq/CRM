package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type config struct {
	mu   sync.RWMutex
	data map[string]string
}

func (c *config) DefaultString(key, def string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.data[key]; ok && v != "" {
		return v
	}
	return def
}

var AppConfig = &config{data: map[string]string{}}

func init() {
	loadAppConfig()
}

func loadAppConfig() {
	AppConfig.mu.Lock()
	defer AppConfig.mu.Unlock()
	for _, file := range configCandidates() {
		if b, err := os.ReadFile(file); err == nil {
			parseAppConfig(string(b), AppConfig.data)
			return
		}
	}
}

func configCandidates() []string {
	files := []string{"conf/app.conf", "app.conf"}
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		files = append(files,
			filepath.Join(base, "conf", "app.conf"),
			filepath.Join(base, "app.conf"),
		)
	}
	return files
}

func parseAppConfig(text string, dst map[string]string) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.Index(line, "="); i > 0 {
			key := strings.TrimSpace(line[:i])
			val := strings.TrimSpace(line[i+1:])
			dst[key] = expandEnv(val)
		}
	}
}

func expandEnv(v string) string {
	if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
		body := strings.TrimSuffix(strings.TrimPrefix(v, "${"), "}")
		parts := strings.SplitN(body, "||", 2)
		if len(parts) == 2 {
			if env := os.Getenv(parts[0]); env != "" {
				return env
			}
			return parts[1]
		}
		if env := os.Getenv(body); env != "" {
			return env
		}
	}
	return v
}

type Controller struct {
	Ctx  *Context
	Data map[string]any
}

func (c *Controller) ServeJSON() {
	if c.Data == nil {
		c.Data = map[string]any{}
	}
	c.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(c.Ctx.ResponseWriter).Encode(c.Data["json"])
}

func (c *Controller) GetString(key string) string {
	if c.Ctx == nil || c.Ctx.Request == nil {
		return ""
	}
	if v := c.Ctx.Request.URL.Query().Get(key); v != "" {
		return v
	}
	return c.Ctx.Request.FormValue(key)
}

func (c *Controller) GetInt(key string) (int, error) {
	return strconv.Atoi(c.GetString(key))
}

type ResponseWriter struct{ http.ResponseWriter }

type Input struct {
	Request     *http.Request
	RequestBody []byte
	params      map[string]string
}

func (i *Input) Param(key string) string {
	return i.params[key]
}

type Context struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	Input          *Input
}

type route struct {
	pattern string
	methods map[string]string
	ctrl    any
}

var (
	mu          sync.RWMutex
	routes      []route
	staticMount string
	staticDir   string
)

func Router(pattern string, c any, mapping string) {
	methods := map[string]string{}
	for _, item := range strings.Split(mapping, ";") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		kv := strings.SplitN(item, ":", 2)
		if len(kv) == 2 {
			methods[strings.ToUpper(strings.TrimSpace(kv[0]))] = strings.TrimSpace(kv[1])
		}
	}
	mu.Lock()
	defer mu.Unlock()
	routes = append(routes, route{pattern: pattern, methods: methods, ctrl: c})
}

func SetStaticPath(prefix, dir string) {
	staticMount = prefix
	staticDir = resolveAppPath(dir)
}

func Run() {
	port := AppConfig.DefaultString("httpport", "8080")
	mux := http.NewServeMux()
	mux.HandleFunc("/", serveHTTP)
	_ = http.ListenAndServe(":"+port, mux)
}

func serveHTTP(w http.ResponseWriter, r *http.Request) {
	if handleRoute(w, r) {
		return
	}
	if serveStatic(w, r) {
		return
	}
	http.NotFound(w, r)
}

func handleRoute(w http.ResponseWriter, r *http.Request) bool {
	mu.RLock()
	defer mu.RUnlock()
	for _, rt := range routes {
		params, ok := matchRoute(rt.pattern, r.URL.Path)
		if !ok {
			continue
		}
		method := rt.methods[strings.ToUpper(r.Method)]
		if method == "" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return true
		}
		var body []byte
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			body, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
		ctrlType := reflect.TypeOf(rt.ctrl)
		ctrlValue := reflect.New(ctrlType.Elem())
		invoke := ctrlValue.MethodByName(method)
		if !invoke.IsValid() {
			http.Error(w, "handler not found", http.StatusNotFound)
			return true
		}
		injectContext(ctrlValue.Elem(), &Context{Request: r, ResponseWriter: w, Input: &Input{Request: r, RequestBody: body, params: params}})
		invoke.Call(nil)
		return true
	}
	return false
}

func injectContext(v reflect.Value, ctx *Context) {
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return
	}
	if f := v.FieldByName("Controller"); f.IsValid() && f.CanSet() {
		if c, ok := f.Addr().Interface().(*Controller); ok {
			c.Ctx = ctx
			c.Data = map[string]any{}
			return
		}
	}
	if f := v.FieldByName("APIController"); f.IsValid() && f.CanSet() {
		injectContext(f, ctx)
	}
}

func matchRoute(pattern, actual string) (map[string]string, bool) {
	pp := strings.Split(strings.Trim(pattern, "/"), "/")
	ap := strings.Split(strings.Trim(actual, "/"), "/")
	if len(pp) != len(ap) {
		if pattern == "/" && actual == "/" {
			return map[string]string{}, true
		}
		return nil, false
	}
	params := map[string]string{}
	for i := range pp {
		if strings.HasPrefix(pp[i], ":") {
			params[pp[i]] = ap[i]
			continue
		}
		if pp[i] != ap[i] {
			return nil, false
		}
	}
	return params, true
}

func serveStatic(w http.ResponseWriter, r *http.Request) bool {
	if staticDir == "" {
		return false
	}
	if r.URL.Path == "/" && staticMount == "/" {
		http.ServeFile(w, r, path.Join(staticDir, "index.html"))
		return true
	}
	fp := path.Clean(path.Join(staticDir, strings.TrimPrefix(r.URL.Path, "/")))
	info, err := os.Stat(fp)
	if err != nil || info.IsDir() {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, path.Join(staticDir, "index.html"))
			return true
		}
		return false
	}
	http.ServeFile(w, r, fp)
	return true
}

func resolveAppPath(value string) string {
	if value == "" {
		return value
	}
	if filepath.IsAbs(value) {
		return value
	}
	if _, err := os.Stat(value); err == nil {
		return value
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), value)
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate
		}
		return candidate
	}
	return value
}
