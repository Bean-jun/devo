package cli

import (
	"context"
	"net/http"
	"os"
	"time"

	"devo/internal/pkg/logging"
	webembed "devo/web"
)

func serveWebUI(mux *http.ServeMux) {
	webFS, err := webembed.StaticFS()
	if err != nil {
		webFS = os.DirFS("web/dist")
	}

	fileServer := http.FileServer(http.FS(webFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			fileServer.ServeHTTP(w, r)
			return
		}
		path := r.URL.Path
		f, err := webFS.Open(path[1:])
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

func waitForReady(baseURL string, timeout time.Duration) {
	ctx := context.Background()
	deadline := time.Now().Add(timeout)
	url := baseURL + "/api/v1/sessions"
	client := &http.Client{Timeout: 2 * time.Second}
	logging.Info(ctx, "waiting for server readiness",
		"timeout", timeout.String(),
	)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	logging.Error(ctx, "server not ready within timeout",
		"timeout", timeout.String(),
	)
}
