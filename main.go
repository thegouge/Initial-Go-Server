package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	const PORT string = "8000"
	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir("./pages")))

	corsMux := middlewareCors(mux)

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
