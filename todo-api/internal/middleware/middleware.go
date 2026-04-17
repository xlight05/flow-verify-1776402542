package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/openchoreo/todo-api/internal/models"
)

type contextKey string

const subjectKey contextKey = "subject"

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.status = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.status = http.StatusOK
		r.wroteHeader = true
	}
	return r.ResponseWriter.Write(b)
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start))
	})
}

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered at %s %s: %v", r.Method, r.URL.Path, rec)
				writeJSONError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "authorization header required")
			return
		}
		if !strings.HasPrefix(header, "Bearer ") && !strings.HasPrefix(header, "bearer ") {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "bearer token required")
			return
		}
		token := strings.TrimSpace(header[len("Bearer "):])
		if token == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "empty bearer token")
			return
		}
		ctx := context.WithValue(r.Context(), subjectKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Subject(r *http.Request) string {
	v, _ := r.Context().Value(subjectKey).(string)
	return v
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(models.ErrorResponse{Code: code, Message: message})
}
