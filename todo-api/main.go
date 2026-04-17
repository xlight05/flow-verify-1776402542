package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/openchoreo/todo-api/internal/handlers"
	"github.com/openchoreo/todo-api/internal/middleware"
	"github.com/openchoreo/todo-api/internal/models"
	"github.com/openchoreo/todo-api/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	s := store.New()
	seed(s)

	mux := http.NewServeMux()
	h := handlers.New(s)
	h.Register(mux)

	handler := middleware.Recovery(middleware.Logging(mux))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("todo-api listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func seed(s *store.Store) {
	now := time.Now().UTC()
	s.Add(&models.Todo{
		ID:        uuid.NewString(),
		Title:     "Read the API docs",
		Owner:     "alice",
		Completed: false,
		CreatedAt: now,
	})
	s.Add(&models.Todo{
		ID:        uuid.NewString(),
		Title:     "Write sample todos",
		Owner:     "alice",
		Completed: false,
		CreatedAt: now,
	})
	s.Add(&models.Todo{
		ID:        uuid.NewString(),
		Title:     "Deploy the service",
		Owner:     "bob",
		Completed: false,
		CreatedAt: now,
	})
}
