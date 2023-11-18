package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/thegouge/Initial-Go-Server/internal/database"
)

type apiConfig struct {
	fileserverHits int
	db             *database.DB
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

func (cfg *apiConfig) chirpValidationHandler(w http.ResponseWriter, r *http.Request) {
	type validationParams struct {
		Body string `json:"body"`
	}
	type validResponse struct {
		Id   int    `json:"id"`
		Body string `json:"body"`
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

	createdChirp, err := cfg.db.CreateChirp(cleanString(params.Body))
	if err != nil {
		log.Printf("Error saving Chirp to database: %v", err)
		w.WriteHeader(500)
		return
	}

	respBody := createdChirp

	respondWithJson(w, 200, respBody)
}

func (cfg *apiConfig) getAllChirps(w http.ResponseWriter, r *http.Request) {

}
