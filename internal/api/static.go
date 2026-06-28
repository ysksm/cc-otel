package api

import (
	"bytes"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
)

// timeZero disables Last-Modified handling in ServeContent (assets are embedded).
var timeZero time.Time

const placeholderHTML = `<!doctype html><html><head><meta charset="utf-8"><title>cc-otel</title>
<style>body{font-family:system-ui,sans-serif;max-width:40rem;margin:4rem auto;padding:0 1rem;color:#222}
code{background:#f3f3f3;padding:.1rem .3rem;border-radius:3px}</style></head>
<body><h1>cc-otel</h1>
<p>The API is running. The web UI has not been built into this binary yet.</p>
<p>Send OpenTelemetry traces to <code>POST /v1/traces</code> and query the REST API under <code>/api</code>.</p>
<p>Try <a href="/api/health">/api/health</a> or <a href="/api/projects">/api/projects</a>.</p>
</body></html>`

// handleStatic serves the embedded SPA, falling back to index.html for
// client-side routes. When the SPA has not been built, it serves a placeholder.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if s.webFS == nil {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, placeholderHTML)
			return
		}
		http.NotFound(w, r)
		return
	}

	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "" {
		name = "index.html"
	}
	data, ctype, err := readWebFile(s.webFS, name)
	if err != nil {
		// SPA fallback: serve index.html for unknown (client-routed) paths.
		data, ctype, err = readWebFile(s.webFS, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}
	w.Header().Set("Content-Type", ctype)
	http.ServeContent(w, r, name, timeZero, bytes.NewReader(data))
}

func readWebFile(fsys fs.FS, name string) (data []byte, ctype string, err error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	if st, err := f.Stat(); err == nil && st.IsDir() {
		return nil, "", fs.ErrNotExist
	}
	data, err = io.ReadAll(f)
	if err != nil {
		return nil, "", err
	}
	ctype = mime.TypeByExtension(path.Ext(name))
	if ctype == "" {
		ctype = http.DetectContentType(data)
	}
	return data, ctype, nil
}
