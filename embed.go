package main

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed frontend/build/*
var frontendFS embed.FS

func frontendFileServer() http.Handler {
	dist, _ := fs.Sub(frontendFS, "frontend/build")
	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		f, err := dist.Open(path)
		if err != nil {
			data, _ := frontendFS.ReadFile("frontend/build/index.html")
			w.Header().Set("Content-Type", "text/html")
			w.Write(data)
			return
		}
		f.Close()

		fileServer.ServeHTTP(w, r)
	})
}
