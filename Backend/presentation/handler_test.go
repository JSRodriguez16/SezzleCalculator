package presentation

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sezzlecalculator/backend/business"
	"sezzlecalculator/backend/exceptions"
)

func newHandler(t *testing.T) http.Handler {
	t.Helper()
	handler, err := NewHandler(business.Service{}, "")
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func request(handler http.Handler, method, path, body, contentType string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func TestOperations(t *testing.T) {
	tests := []struct {
		operation, body string
		want            float64
	}{
		{"add", `{"a":2,"b":3}`, 5},
		{"subtract", `{"a":2,"b":3}`, -1},
		{"multiply", `{"a":2,"b":3}`, 6},
		{"divide", `{"a":7,"b":2}`, 3.5},
		{"power", `{"a":2,"b":3}`, 8},
		{"sqrt", `{"a":81}`, 9},
		{"percentage", `{"a":200,"b":15}`, 30},
		{"add", `{"a":0,"b":0}`, 0},
		{"sqrt", `{"a":0}`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.operation+tt.body, func(t *testing.T) {
			t.Parallel()
			w := request(newHandler(t), http.MethodPost, "/api/v1/"+tt.operation, tt.body, "application/json; charset=utf-8")
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}
			var response struct {
				Operation string  `json:"operation"`
				Result    float64 `json:"result"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Operation != tt.operation || response.Result != tt.want {
				t.Fatalf("response = %+v, want %s = %v", response, tt.operation, tt.want)
			}
			if w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
				t.Fatalf("unexpected content type: %s", w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestRequestErrors(t *testing.T) {
	tests := []struct {
		name, method, path, body, contentType string
		status                                int
		code                                  exceptions.Code
	}{
		{"missing content type", "POST", "/api/v1/add", `{"a":1,"b":2}`, "", 415, exceptions.UnsupportedMediaType},
		{"wrong content type", "POST", "/api/v1/add", `{"a":1,"b":2}`, "text/plain", 415, exceptions.UnsupportedMediaType},
		{"malformed content type", "POST", "/api/v1/add", `{"a":1,"b":2}`, "application/json; bad", 415, exceptions.UnsupportedMediaType},
		{"empty body", "POST", "/api/v1/add", "", "application/json", 400, exceptions.InvalidJSON},
		{"malformed JSON", "POST", "/api/v1/add", `{"a":1`, "application/json", 400, exceptions.InvalidJSON},
		{"string operand", "POST", "/api/v1/add", `{"a":"1","b":2}`, "application/json", 400, exceptions.InvalidJSON},
		{"boolean operand", "POST", "/api/v1/add", `{"a":true,"b":2}`, "application/json", 400, exceptions.InvalidJSON},
		{"array body", "POST", "/api/v1/add", `[1,2]`, "application/json", 400, exceptions.InvalidJSON},
		{"unknown field", "POST", "/api/v1/add", `{"a":1,"b":2,"extra":3}`, "application/json", 400, exceptions.InvalidJSON},
		{"extra operand for root", "POST", "/api/v1/sqrt", `{"a":4,"b":2}`, "application/json", 400, exceptions.InvalidJSON},
		{"missing first operand", "POST", "/api/v1/add", `{"b":2}`, "application/json", 400, exceptions.InvalidInput},
		{"missing second operand", "POST", "/api/v1/add", `{"a":2}`, "application/json", 400, exceptions.InvalidInput},
		{"missing root operand", "POST", "/api/v1/sqrt", `{}`, "application/json", 400, exceptions.InvalidInput},
		{"null operand", "POST", "/api/v1/add", `{"a":null,"b":2}`, "application/json", 400, exceptions.InvalidInput},
		{"null body", "POST", "/api/v1/add", `null`, "application/json", 400, exceptions.InvalidInput},
		{"trailing JSON", "POST", "/api/v1/add", `{"a":1,"b":2} {}`, "application/json", 400, exceptions.InvalidJSON},
		{"trailing junk", "POST", "/api/v1/add", `{"a":1,"b":2} junk`, "application/json", 400, exceptions.InvalidJSON},
		{"out of range operand", "POST", "/api/v1/add", `{"a":1e400,"b":2}`, "application/json", 400, exceptions.InvalidJSON},
		{"divide by zero", "POST", "/api/v1/divide", `{"a":1,"b":0}`, "application/json", 422, exceptions.DivisionByZero},
		{"negative root", "POST", "/api/v1/sqrt", `{"a":-1}`, "application/json", 422, exceptions.InvalidSquareRoot},
		{"nonreal power", "POST", "/api/v1/power", `{"a":-1,"b":0.5}`, "application/json", 422, exceptions.InvalidExponent},
		{"overflow", "POST", "/api/v1/power", `{"a":10,"b":400}`, "application/json", 422, exceptions.NonFiniteResult},
		{"unknown API route", "GET", "/api/v1/unknown", "", "", 404, exceptions.NotFound},
		{"API prefix", "GET", "/api", "", "", 404, exceptions.NotFound},
		{"wrong operation method", "GET", "/api/v1/add", "", "", 405, exceptions.MethodNotAllowed},
		{"wrong health method", "POST", "/api/health", "", "", 405, exceptions.MethodNotAllowed},
		{"API-only root", "GET", "/", "", "", 404, exceptions.NotFound},
		{"oversized body", "POST", "/api/v1/add", strings.Repeat(" ", int(MaxBodyBytes)+1), "application/json", 413, exceptions.PayloadTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := request(newHandler(t), tt.method, tt.path, tt.body, tt.contentType)
			assertError(t, w, tt.status, tt.code)
			if w.Code == 405 && w.Header().Get("Allow") == "" {
				t.Error("405 response must include Allow")
			}
		})
	}
}

func assertError(t *testing.T, w *httptest.ResponseRecorder, status int, code exceptions.Code) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, status, w.Body.String())
	}
	var response struct {
		Error exceptions.Error `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("response must be JSON: %v; body = %s", err, w.Body.String())
	}
	if response.Error.Code != code || response.Error.Message == "" {
		t.Fatalf("error = %+v, want code %s and nonempty message", response.Error, code)
	}
	if w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Error("errors must include the JSON content type")
	}
}

func TestChunkedAndTrailingOversizedBodies(t *testing.T) {
	for _, body := range []string{
		strings.Repeat(" ", int(MaxBodyBytes)+1),
		`{"a":1,"b":2}` + strings.Repeat(" ", int(MaxBodyBytes)),
	} {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/add", strings.NewReader(body))
		r.ContentLength = -1 // Chunked requests cannot rely on Content-Length.
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		newHandler(t).ServeHTTP(w, r)
		assertError(t, w, http.StatusRequestEntityTooLarge, exceptions.PayloadTooLarge)
	}
}

type fakeCalculator func(business.Operation, float64, float64) (float64, error)

func (f fakeCalculator) Calculate(operation business.Operation, a, b float64) (float64, error) {
	return f(operation, a, b)
}

func TestInternalFailuresReturnSafeJSON(t *testing.T) {
	tests := []struct {
		name    string
		service fakeCalculator
	}{
		{"unexpected error", func(business.Operation, float64, float64) (float64, error) {
			return 0, errors.New("private implementation details")
		}},
		{"panic", func(business.Operation, float64, float64) (float64, error) {
			panic("private implementation details")
		}},
		{"unserializable result", func(business.Operation, float64, float64) (float64, error) {
			return math.NaN(), nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewHandler(tt.service, "")
			if err != nil {
				t.Fatal(err)
			}
			w := request(handler, "POST", "/api/v1/add", `{"a":1,"b":2}`, "application/json")
			assertError(t, w, http.StatusInternalServerError, exceptions.InternalError)
			if strings.Contains(w.Body.String(), "private") {
				t.Fatal("internal implementation details leaked")
			}
		})
	}
}

func TestHealthOverHTTP(t *testing.T) {
	server := httptest.NewServer(newHandler(t))
	defer server.Close()
	response, err := server.Client().Get(server.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || strings.TrimSpace(string(body)) != `{"status":"ok"}` {
		t.Fatalf("health = %d %s", response.StatusCode, body)
	}
}

func TestStaticFrontend(t *testing.T) {
	staticDir := t.TempDir()
	writeFile(t, filepath.Join(staticDir, "index.html"), "<!doctype html><h1>Calculator</h1>")
	writeFile(t, filepath.Join(staticDir, "app.js"), "console.log('calculator')")
	writeFile(t, filepath.Join(staticDir, ".env"), "PRIVATE=secret")
	handler, err := NewHandler(business.Service{}, staticDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"/", "/index.html", "/history", "/history/recent"} {
		w := request(handler, "GET", route, "", "")
		if w.Code != 200 || !strings.Contains(w.Body.String(), "Calculator") {
			t.Errorf("SPA route %s = %d %s", route, w.Code, w.Body.String())
		}
	}
	w := request(handler, "GET", "/app.js", "", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "console.log") {
		t.Fatalf("static asset = %d %s", w.Code, w.Body.String())
	}
	w = request(handler, "HEAD", "/", "", "")
	if w.Code != 200 || w.Body.Len() != 0 {
		t.Fatalf("HEAD = %d with %d body bytes", w.Code, w.Body.Len())
	}
	for _, route := range []string{"/missing.js", "/api/unknown", "/api", "/../outside", "/%2e%2e/outside", "/.env", "/nested/../app.js", "/nested%5c..%5capp.js"} {
		t.Run(route, func(t *testing.T) {
			assertError(t, request(handler, "GET", route, "", ""), 404, exceptions.NotFound)
		})
	}
	w = request(handler, "POST", "/", "", "")
	assertError(t, w, 405, exceptions.MethodNotAllowed)
	if w.Header().Get("Allow") != "GET, HEAD" {
		t.Errorf("static Allow = %q", w.Header().Get("Allow"))
	}
}

func TestStaticSymlinksCannotEscapeBuild(t *testing.T) {
	staticDir := t.TempDir()
	writeFile(t, filepath.Join(staticDir, "index.html"), "Calculator")
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "private.txt")
	writeFile(t, outsidePath, "private")
	if err := os.Symlink(outsidePath, filepath.Join(staticDir, "leak.txt")); err != nil {
		t.Skipf("symlinks unavailable on this system: %v", err)
	}
	handler, err := NewHandler(business.Service{}, staticDir)
	if err != nil {
		t.Fatal(err)
	}
	assertError(t, request(handler, "GET", "/leak.txt", "", ""), 404, exceptions.NotFound)
}

func TestStaticConfigurationFailsEarly(t *testing.T) {
	for _, directory := range []string{filepath.Join(t.TempDir(), "missing"), t.TempDir()} {
		if _, err := NewHandler(business.Service{}, directory); err == nil {
			t.Fatalf("NewHandler(%q) should reject a missing build", directory)
		}
	}
	staticDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(staticDir, "index.html"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := NewHandler(business.Service{}, staticDir); err == nil {
		t.Fatal("index.html must be a regular file")
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

type coverageFile struct {
	reader  *strings.Reader
	infos   []os.FileInfo
	statErr error
}

func (f *coverageFile) Read(p []byte) (int, error) { return f.reader.Read(p) }
func (f *coverageFile) Seek(offset int64, whence int) (int64, error) {
	return f.reader.Seek(offset, whence)
}
func (f *coverageFile) Stat() (os.FileInfo, error) {
	if f.statErr != nil {
		return nil, f.statErr
	}
	info := f.infos[0]
	if len(f.infos) > 1 {
		f.infos = f.infos[1:]
	}
	return info, nil
}
func (f *coverageFile) Close() error { return nil }

func TestNewHandlerReturnsAbsolutePathError(t *testing.T) {
	previous := absolutePath
	defer func() { absolutePath = previous }()
	absolutePath = func(string) (string, error) { return "", errors.New("absolute path failure") }
	if _, err := NewHandler(nil, "static"); err == nil {
		t.Fatal("NewHandler should return absolute path error")
	}
}

func TestOpenStaticReturnsRelativePathError(t *testing.T) {
	staticDir := t.TempDir()
	previousEval, previousRel := evalSymlinks, relativePath
	defer func() { evalSymlinks, relativePath = previousEval, previousRel }()
	evalSymlinks = func(name string) (string, error) { return name, nil }
	relativePath = func(string, string) (string, error) { return "", errors.New("relative path failure") }
	if _, err := (&handler{staticDir: staticDir}).openStatic("index.html"); err == nil {
		t.Fatal("openStatic should return relative path error")
	}
}

func TestOpenStaticReturnsOpenError(t *testing.T) {
	previousEval, previousOpen := evalSymlinks, openFile
	defer func() { evalSymlinks, openFile = previousEval, previousOpen }()
	evalSymlinks = func(name string) (string, error) { return name, nil }
	openFile = func(string) (staticFile, error) { return nil, errors.New("open failure") }
	if _, err := (&handler{staticDir: t.TempDir()}).openStatic("index.html"); err == nil {
		t.Fatal("openStatic should return open error")
	}
}

func TestOpenStaticReturnsStatErrors(t *testing.T) {
	staticDir := t.TempDir()
	previousEval, previousOpen := evalSymlinks, openFile
	defer func() { evalSymlinks, openFile = previousEval, previousOpen }()
	evalSymlinks = func(name string) (string, error) { return name, nil }
	openFile = func(string) (staticFile, error) {
		return &coverageFile{reader: strings.NewReader(""), statErr: errors.New("stat failure")}, nil
	}
	if _, err := (&handler{staticDir: staticDir}).openStatic("index.html"); err == nil {
		t.Fatal("openStatic should return stat error")
	}

	directory, err := os.Open(staticDir)
	if err != nil {
		t.Fatal(err)
	}
	info, err := directory.Stat()
	directory.Close()
	if err != nil {
		t.Fatal(err)
	}
	openFile = func(string) (staticFile, error) {
		return &coverageFile{reader: strings.NewReader(""), infos: []os.FileInfo{info}}, nil
	}
	if _, err := (&handler{staticDir: staticDir}).openStatic("index.html"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("openStatic error = %v, want not exist", err)
	}
}

func TestServeStaticReturnsStatError(t *testing.T) {
	staticDir := t.TempDir()
	previousEval, previousOpen := evalSymlinks, openFile
	defer func() { evalSymlinks, openFile = previousEval, previousOpen }()
	evalSymlinks = func(name string) (string, error) { return name, nil }
	staticPath := filepath.Join(staticDir, "index.html")
	if err := os.WriteFile(staticPath, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(staticPath)
	if err != nil {
		t.Fatal(err)
	}
	openFile = func(string) (staticFile, error) {
		return &coverageFile{reader: strings.NewReader("content"), infos: []os.FileInfo{info, info}}, nil
	}
	file := openFile
	openFile = func(name string) (staticFile, error) {
		value, err := file(name)
		if err != nil {
			return value, err
		}
		return &serveStatErrorFile{staticFile: value}, nil
	}
	response := httptest.NewRecorder()
	(&handler{staticDir: staticDir}).serveStatic(response, httptest.NewRequest(http.MethodGet, "/index.html", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

type serveStatErrorFile struct {
	staticFile
	statCalls int
}

func (f *serveStatErrorFile) Stat() (os.FileInfo, error) {
	info, err := f.staticFile.Stat()
	if err != nil {
		return nil, err
	}
	if f.statCalls == 0 {
		f.statCalls++
		return info, nil
	}
	return nil, errors.New("serve stat failure")
}
