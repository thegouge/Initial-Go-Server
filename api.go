package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"

	"github.com/go-chi/chi/v5"
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

	respondWithJson(w, 201, respBody)
}

func (cfg *apiConfig) getAllChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetChirps()
	if err != nil {
		log.Printf("Error reading the database: %v", err)
		w.WriteHeader(500)
		return
	}

	sort.Slice(chirps, func(i, j int) bool {
		return chirps[i].Id < chirps[j].Id
	})

	respondWithJson(w, 200, chirps)
}

func (cfg *apiConfig) getChirpByID(w http.ResponseWriter, r *http.Request) {
	param := chi.URLParam(r, "chirpId")
	chirpID, err := strconv.Atoi(param)
	if err != nil {
		log.Printf("Error parsing parameter: %v", err)
		respondWithError(w, 400, "invalid chirp ID")
		return
	}

	allChirps, err := cfg.db.GetChirps()
	if err != nil {
		log.Printf("Error reading the database: %v", err)
		w.WriteHeader(500)
		return
	}

	for _, chirp := range allChirps {
		if chirp.Id == chirpID {
			respondWithJson(w, 200, chirp)
			return
		}
	}

	respondWithError(w, 404, fmt.Sprintf("Unable to find chirp with ID: %s", param))
}

type fullUser struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	params := fullUser{}

	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding User Object: %v", err)
		w.WriteHeader(500)
		return
	}

	createdUser, err := cfg.db.CreateUser(params.Email, params.Password)
	if err != nil {
		log.Printf("Error saving User to database: %v", err)
		w.WriteHeader(500)
		return
	}

	respBody := createdUser

	respondWithJson(w, 201, respBody)
}
