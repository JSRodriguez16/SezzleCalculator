package presentation

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"sezzlecalculator/backend/exceptions"
)

type staticFile interface {
	io.ReadSeeker
	Stat() (os.FileInfo, error)
	Close() error
}

var relativePath = filepath.Rel
var openFile = func(name string) (staticFile, error) { return os.Open(name) }

func (h *handler) openStatic(name string) (staticFile, error) {
	if !fs.ValidPath(name) || strings.Contains(name, "\\") {
		return nil, fs.ErrPermission
	}
	for _, segment := range strings.Split(name, "/") {
		if strings.HasPrefix(segment, ".") {
			return nil, fs.ErrPermission
		}
	}
	resolved, err := evalSymlinks(filepath.Join(h.staticDir, filepath.FromSlash(name)))
	if err != nil {
		return nil, err
	}
	relative, err := relativePath(h.staticDir, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, fs.ErrPermission
	}
	file, err := openFile(resolved)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		file.Close()
		if err != nil {
			return nil, err
		}
		return nil, fs.ErrNotExist
	}
	return file, nil
}

func (h *handler) serveStatic(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		name = "index.html"
	}
	file, err := h.openStatic(name)
	if err != nil && os.IsNotExist(err) && path.Ext(name) == "" {
		// Client-side routes use the SPA entry point, missing assets still 404
		file, err = h.openStatic("index.html")
	}
	if err != nil {
		writeError(w, exceptions.New(exceptions.NotFound, "La ruta solicitada no existe."))
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		writeError(w, err)
		return
	}
	if info.Name() == "index.html" {
		w.Header().Set("Cache-Control", "no-cache")
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}
