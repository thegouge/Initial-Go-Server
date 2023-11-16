package main

import (
	"fmt"
	"net/http"
)

type apiConfig struct {
	fileserverHits int
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, Request *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(200)

	w.Write([]byte(fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits)))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, Request *http.Request) {
	cfg.fileserverHits = 0

	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)

	w.Write([]byte("Metrics have been reset"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++

		next.ServeHTTP(w, r)
	})
}
