package main

import (
	"log"
	"net/http"
	"os"
	"time"

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
	deadline := time.Now().Add(timeout)
	url := baseURL + "/api/v1/sessions"
	client := &http.Client{Timeout: 2 * time.Second}
	log.Printf("[devo] Waiting for server readiness (timeout: %v)...", timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				log.Printf("[devo] Server is ready.")
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	log.Printf("[devo] Server did not become ready within %v", timeout)
	log.Fatalf("server did not become ready within %v", timeout)
}
