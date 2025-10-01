package main

import (
	"go-practice2/internal/handlers"
	"go-practice2/internal/middleware"

	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.Handle("/user", middleware.AuthMiddleware(http.HandlerFunc(handlers.UserHandler)))

	log.Println("Listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
