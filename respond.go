package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errResponse struct {
		Error string `json:"error"`
	}

	responseStruct := errResponse{
		Error: msg,
	}

	fmt.Println(msg)
	respondWithJson(w, code, responseStruct)
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	if code != 200 {
		w.WriteHeader(code)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(dat)
}
