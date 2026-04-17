package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/openchoreo/todo-api/internal/middleware"
	"github.com/openchoreo/todo-api/internal/models"
	"github.com/openchoreo/todo-api/internal/store"
)

const (
	contentTypeJSON        = "application/json; charset=utf-8"
	todoItemAllowedMethods = "GET, PATCH, DELETE"
	todoCollectionMethods  = "GET, POST"
)

type Handler struct {
	store *store.Store
	now   func() time.Time
}

func New(s *store.Store) *Handler {
	return &Handler{store: s, now: func() time.Time { return time.Now().UTC() }}
}

func (h *Handler) SetClock(now func() time.Time) {
	h.now = now
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.health)
	mux.Handle("/todos", middleware.Auth(http.HandlerFunc(h.todosCollection)))
	mux.Handle("/todos/", middleware.Auth(http.HandlerFunc(h.todoItem)))
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)
	if body == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, models.ErrorResponse{Code: code, Message: message})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
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
		w.Header().Set("Allow", todoCollectionMethods)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (h *Handler) listTodos(w http.ResponseWriter, r *http.Request) {
	subject := middleware.Subject(r)
	todos := h.store.ListByOwner(subject)
	writeJSON(w, http.StatusOK, todos)
}

func (h *Handler) createTodo(w http.ResponseWriter, r *http.Request) {
	subject := middleware.Subject(r)

	var body struct {
		Title string `json:"title"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	if dec.More() {
		writeError(w, http.StatusBadRequest, "invalid_body", "unexpected trailing content")
		return
	}

	title := strings.TrimSpace(body.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "invalid_title", "title is required")
		return
	}
	if len(title) > 255 {
		writeError(w, http.StatusBadRequest, "invalid_title", "title must be at most 255 characters")
		return
	}

	t := &models.Todo{
		ID:        uuid.NewString(),
		Title:     title,
		Owner:     subject,
		Completed: false,
		CreatedAt: h.now(),
	}
	h.store.Add(t)
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) todoItem(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/todos/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid todo id")
		return
	}
	subject := middleware.Subject(r)

	switch r.Method {
	case http.MethodGet:
		h.getTodo(w, id, subject)
	case http.MethodPatch:
		h.patchTodo(w, r, id, subject)
	case http.MethodDelete:
		h.deleteTodo(w, id, subject)
	default:
		w.Header().Set("Allow", todoItemAllowedMethods)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (h *Handler) getTodo(w http.ResponseWriter, id, subject string) {
	t, err := h.store.Get(id, subject)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteTodo(w http.ResponseWriter, id, subject string) {
	if err := h.store.Delete(id, subject); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) patchTodo(w http.ResponseWriter, r *http.Request, id, subject string) {
	completed := true

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "failed to read request body")
		return
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 {
		var body struct {
			Completed *bool `json:"completed"`
		}
		dec := json.NewDecoder(bytes.NewReader(trimmed))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
			return
		}
		if dec.More() {
			writeError(w, http.StatusBadRequest, "invalid_body", "unexpected trailing content")
			return
		}
		if body.Completed != nil {
			completed = *body.Completed
		}
	}

	updated, err := h.store.SetCompleted(id, subject, completed, h.now())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "todo not found")
	case errors.Is(err, store.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "not authorized to access this todo")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
