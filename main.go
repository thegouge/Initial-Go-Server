package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func main() {
	const PORT string = "8000"

	db, dbErr := database.NewDB("database.json")
	if dbErr != nil {
		log.Fatal(dbErr)
	}

	apiCfg := apiConfig{
		db: db,
	}

	r := chi.NewRouter()
	api := chi.NewRouter()
	admin := chi.NewRouter()

	r.Handle("/app", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("./pages")))))
	r.Handle("/app/*", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir("./pages")))))

	api.Get("/healthz", healthHandler)
	api.Handle("/reset", http.HandlerFunc(apiCfg.resetHandler))
	api.Post("/chirps", http.HandlerFunc(apiCfg.chirpValidationHandler))
	api.Get("/chirps", http.HandlerFunc(apiCfg.getAllChirps))

	admin.Get("/metrics", http.HandlerFunc(apiCfg.metricsHandler))

	r.Mount("/api", api)
	r.Mount("/admin", admin)

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

func chirpValidationHandler(w http.ResponseWriter, r *http.Request) {
	type validationParams struct {
		Body string `json:"body"`
	}
	type validResponse struct {
		CleanedBody string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := validationParams{}

	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding Chirp: %v", err)
		w.WriteHeader(500)
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	respBody := validResponse{
		CleanedBody: cleanString(params.Body),
	}

	respondWithJson(w, 200, respBody)
}

func cleanString(input string) string {
	CURSE_WORDS := [4]string{
		"kerfuffle",
		"sharbert",
		"fornax",
		"fuck",
	}

	words := strings.Split(input, " ")

	for i, word := range words {
		for _, curse := range CURSE_WORDS {
			if curse == strings.ToLower(word) {
				words[i] = "****"
			}
		}
	}

	return strings.Join(words, " ")
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
