package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/thegouge/go-chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits int
	db             *database.DB
	secret         string
	polkaKey       string
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

	auth := r.Header.Get("Authorization")

	if auth == "" {
		respondWithError(w, 401, "You need to be logged in to chirp!")
	}

	bearerlessToken := strings.Split(auth, " ")[1]

	decoder := json.NewDecoder(r.Body)
	params := validationParams{}

	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error decoding Chirp: %v", err))
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	id, err := cfg.db.VerifyAccessToken(bearerlessToken, cfg.secret)

	if id == -1 || err != nil {
		respondWithError(w, 401, "Something went wrong authenticating user")
	}

	createdChirp, err := cfg.db.CreateChirp(cleanString(params.Body), id)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error saving Chirp to database: %v", err))
		return
	}

	respBody := createdChirp

	respondWithJson(w, 201, respBody)
}

func (cfg *apiConfig) getAllChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetChirps()
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error reading the database: %v", err))
		return
	}

	sortDir := r.URL.Query().Get("sort")

	if sortDir == "desc" {
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].Id > chirps[j].Id
		})
	} else {
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].Id < chirps[j].Id
		})
	}

	stringAuthor := r.URL.Query().Get("author_id")

	if stringAuthor != "" {
		authorId, err := strconv.Atoi(r.URL.Query().Get("author_id"))

		if err != nil {
			respondWithJson(w, 400, "invalid author id")
		}

		filtered := make([]database.Chirp, 0)
		for _, val := range chirps {
			if val.AuthorId == authorId {
				filtered = append(filtered, val)
			}
		}
		chirps = filtered
	}

	respondWithJson(w, 200, chirps)
}

func (cfg *apiConfig) getChirpByID(w http.ResponseWriter, r *http.Request) {
	param := chi.URLParam(r, "chirpId")
	chirpID, err := strconv.Atoi(param)
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Error parsing parameter: %v", err))
		return
	}

	allChirps, err := cfg.db.GetChirps()
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error reading the database: %v", err))
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
	Email            string `json:"email"`
	Password         string `json:"password"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	params := fullUser{}

	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error decoding User Object: %v", err))
		return
	}

	_, exists, err := cfg.db.GetUserByEmail(params.Email)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error Checking if User Exists: %v", err))
		return
	}

	if exists {
		respondWithError(w, 400, "A user already exists with that Email")
		return
	}

	createdUser, err := cfg.db.CreateUser(params.Email, params.Password)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error saving User to database: %v", err))
		return
	}

	respBody := createdUser

	respondWithJson(w, 201, respBody)
}

type UserWithToken struct {
	Email        string `json:"email"`
	Id           int    `json:"id"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	IsChirpyRed  bool   `json:"is_chirpy_red"`
}

func (cfg *apiConfig) logInUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	params := fullUser{}

	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error decoding User Object: %v", err))
		return
	}

	response, authUser, err := cfg.db.AuthenticateUser(params.Email, params.Password, cfg.secret)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error authenticating user: %v\n", err))
		return
	}

	if !response {
		respondWithError(w, 401, "Invalid request")
		return
	}

	respBody := UserWithToken{
		Email:        params.Email,
		Id:           authUser.Id,
		Token:        authUser.Token,
		RefreshToken: authUser.RefreshToken,
		IsChirpyRed:  authUser.IsChirpyRed,
	}

	respondWithJson(w, 200, respBody)
}

type editedUserResponse struct {
	Email string `json:"email"`
	Id    int    `json:"id"`
}

func (cfg *apiConfig) updateUser(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")

	if auth == "" {
		respondWithError(w, 401, "You need to be logged in to edit a user!")
	}

	bearerlessToken := strings.Split(auth, " ")[1]

	authorized, err := cfg.db.VerifyAccessToken(bearerlessToken, cfg.secret)

	if err != nil {
		respondWithError(w, 401, "Invalid request")
		return
	}

	if authorized != -1 {
		decoder := json.NewDecoder(r.Body)
		params := database.EditingUser{}
		err := decoder.Decode(&params)

		if err != nil {
			respondWithError(w, 500, "something went wrong decoding the edit")
			return
		}

		editedUser, err := cfg.db.EditUser(authorized, params)

		if err != nil {
			respondWithError(w, 500, "Something went wrong editing the user")
			return
		}

		respondWithJson(w, 200, editedUserResponse{
			Email: editedUser.Email,
			Id:    authorized,
		})

	} else {
		respondWithError(w, 401, "invalid access token")
	}

}

type tokenResponse struct {
	Token string `json:"token"`
}

func (cfg *apiConfig) refreshUserToken(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	bearerlessToken := strings.Split(auth, " ")[1]

	newAccessToken, err := cfg.db.VerifyRefreshToken(bearerlessToken, cfg.secret)

	if err != nil {
		respondWithError(w, 401, "Refresh Token invalid")
		return
	}

	respBody := tokenResponse{
		Token: newAccessToken,
	}

	respondWithJson(w, 200, respBody)

}

func (cfg *apiConfig) revokeUserToken(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	bearerlessToken := strings.Split(auth, " ")[1]

	err := cfg.db.RevokeRefreshToken(bearerlessToken)
	if err != nil {
		respondWithError(w, 500, "Something has gone wrong revoking token")
	} else {
		respondWithJson(w, 200, nil)
	}
}

func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")

	if auth == "" {
		respondWithError(w, 401, "You need to be logged in to delete a chirp!")
		return
	}

	bearerlessToken := strings.Split(auth, " ")[1]
	param := chi.URLParam(r, "chirpId")
	chirpID, err := strconv.Atoi(param)

	if err != nil {
		respondWithError(w, 400, "You need to put in a chirp id!")
		return
	}

	err = cfg.db.DeleteChirp(bearerlessToken, cfg.secret, chirpID)

	if err != nil {
		respondWithError(w, 403, "You are not authorized to delete that chirp")
		return
	}

	respondWithJson(w, 200, nil)
}

type polkaEvent struct {
	Event string `json:"event"`
	Data  struct {
		UserId int `json:"user_id"`
	} `json:"data"`
}

func (cfg *apiConfig) handlePayment(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")

	if auth == "" {
		respondWithError(w, 401, "You need to have proper authorization to pay for chirpy red")
		return
	}

	rawToken := strings.Split(auth, " ")[1]

	if rawToken != cfg.polkaKey {
		respondWithError(w, 401, "Invalid Polka Token")
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := polkaEvent{}

	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 400, "Something went wrong parsing request body")
		return
	}

	if params.Event != "user.upgraded" {
		respondWithJson(w, 200, nil)
		return
	}

	err = cfg.db.UpgradeUser(params.Data.UserId)

	if err != nil {
		respondWithError(w, 404, "could not find user to upgrade")
		return
	}

	respondWithJson(w, 200, nil)
}
