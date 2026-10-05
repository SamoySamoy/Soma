package httpx

import (
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// SPA serves a single-page app from fsys. Existing files are served as-is;
// any other path gets index.html so client-side routes work on reload.
// Files under assets/ have content-hashed names and are cached for a year.
func SPA(fsys fs.FS) http.Handler {
	files := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			WriteProblem(w, r, Problem{Status: http.StatusMethodNotAllowed, Code: "request.method_not_allowed"})
			return
		}

		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" && name != "index.html" {
			if st, err := fs.Stat(fsys, name); err == nil && !st.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					w.Header().Set("Cache-Control", "no-cache")
				}
				files.ServeHTTP(w, r)
				return
			}
		}

		index, err := fs.ReadFile(fsys, "index.html")
		if errors.Is(err, fs.ErrNotExist) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("The web app has not been built. Run: go tool task web:build\n"))
			return
		}
		if err != nil {
			WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(index)
	})
}
