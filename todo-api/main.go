package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/openchoreo/todo-api/internal/handlers"
	"github.com/openchoreo/todo-api/internal/middleware"
	"github.com/openchoreo/todo-api/internal/store"
)

func main() {
	dataFile := os.Getenv("DATA_FILE")
	if dataFile == "" {
		dataFile = "/data/todos.json"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	s, err := store.New(dataFile)
	if err != nil {
		log.Fatalf("failed to initialize store: %v", err)
	}

	mux := http.NewServeMux()
	h := handlers.New(s)
	h.Register(mux)

	handler := middleware.Recovery(middleware.Logging(mux))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("todo-api listening on :%s (data file: %s)", port, dataFile)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
