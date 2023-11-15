package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	const PORT string = "8000"
	// mux := http.NewServeMux()
	apiCfg := apiConfig{}
	r := chi.NewRouter()

	r.Handle("/app", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("./pages")))))
	r.Handle("/app/*", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir("./pages")))))
	r.Get("/healthz", healthHandler)
	r.Get("/metrics", http.HandlerFunc(apiCfg.metricsHandler))
	r.Handle("/reset", http.HandlerFunc(apiCfg.resetHandler))

	corsMux := middlewareCors(r)

	server := http.Server{
		Addr:    "localhost:" + PORT,
		Handler: corsMux,
	}

	fmt.Printf("Booting up Server on port %v\n", PORT)
	err := server.ListenAndServe()

	if err != nil {
		log.Fatal(err)
	}

}

func healthHandler(w http.ResponseWriter, Request *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)

	w.Write([]byte("OK"))
}

func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
