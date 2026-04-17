package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/openchoreo/todo-api/internal/models"
	"github.com/openchoreo/todo-api/internal/store"
)

const contentTypeJSON = "application/json; charset=utf-8"

type Handler struct {
	store *store.Store
}

func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/todos", h.todosCollection)
	mux.HandleFunc("/todos/", h.todoItem)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)
	if body == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, models.ErrorResponse{Message: message})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) todosCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listTodos(w, r)
	case http.MethodPost:
		h.createTodo(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) listTodos(w http.ResponseWriter, _ *http.Request) {
	todos := h.store.List()
	if todos == nil {
		todos = []models.Todo{}
	}
	writeJSON(w, http.StatusOK, todos)
}

func (h *Handler) createTodo(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTodoRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if len(title) > 255 {
		writeError(w, http.StatusBadRequest, "title must be at most 255 characters")
		return
	}

	completed := false
	if req.Completed != nil {
		completed = *req.Completed
	}

	todo := models.Todo{
		ID:        uuid.NewString(),
		Title:     title,
		Completed: completed,
		CreatedAt: time.Now().UTC(),
	}

	if err := h.store.Add(todo); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to persist todo")
		return
	}

	writeJSON(w, http.StatusCreated, todo)
}

func (h *Handler) todoItem(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/todos/")
	id = strings.Trim(id, "/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}

	switch r.Method {
	case http.MethodDelete:
		if err := h.store.Delete(id); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "todo not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to delete todo")
			return
		}
		w.Header().Set("Content-Type", contentTypeJSON)
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
