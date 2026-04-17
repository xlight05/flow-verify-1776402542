package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/openchoreo/todo-api/internal/middleware"
	"github.com/openchoreo/todo-api/internal/models"
	"github.com/openchoreo/todo-api/internal/store"
)

func newTestServer(t *testing.T, nowFn func() time.Time) (*httptest.Server, *store.Store) {
	t.Helper()
	s := store.New()
	h := New(s)
	if nowFn != nil {
		h.SetClock(nowFn)
	}
	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(middleware.Recovery(mux))
	t.Cleanup(srv.Close)
	return srv, s
}

func addTodo(s *store.Store, id, owner, title string, completed bool) {
	todo := &models.Todo{
		ID:        id,
		Title:     title,
		Owner:     owner,
		Completed: completed,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if completed {
		ts := todo.CreatedAt
		todo.CompletedAt = &ts
	}
	s.Add(todo)
}

func do(t *testing.T, method, url, token string, body any) *http.Response {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			reader = bytes.NewReader([]byte(v))
		case []byte:
			reader = bytes.NewReader(v)
		default:
			buf, err := json.Marshal(body)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			reader = bytes.NewReader(buf)
		}
	}
	var req *http.Request
	var err error
	if reader != nil {
		req, err = http.NewRequest(method, url, reader)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	return resp
}

func decodeTodo(t *testing.T, resp *http.Response) models.Todo {
	t.Helper()
	var todo models.Todo
	if err := json.NewDecoder(resp.Body).Decode(&todo); err != nil {
		t.Fatalf("decode todo: %v", err)
	}
	return todo
}

func TestHealth(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestPatchSuccessDefaultTrue(t *testing.T) {
	fixed := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	srv, s := newTestServer(t, func() time.Time { return fixed })
	addTodo(s, "t1", "alice", "Do it", false)

	resp := do(t, http.MethodPatch, srv.URL+"/todos/t1", "alice", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	todo := decodeTodo(t, resp)
	if !todo.Completed {
		t.Fatalf("expected completed=true")
	}
	if todo.CompletedAt == nil || !todo.CompletedAt.Equal(fixed) {
		t.Fatalf("expected completed_at=%v, got %v", fixed, todo.CompletedAt)
	}
}

func TestPatchExplicitTrue(t *testing.T) {
	fixed := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	srv, s := newTestServer(t, func() time.Time { return fixed })
	addTodo(s, "t1", "alice", "Do it", false)

	resp := do(t, http.MethodPatch, srv.URL+"/todos/t1", "alice", map[string]any{"completed": true})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	todo := decodeTodo(t, resp)
	if !todo.Completed || todo.CompletedAt == nil {
		t.Fatalf("expected completed=true with timestamp")
	}
}

func TestPatchExplicitFalseClearsTimestamp(t *testing.T) {
	fixed := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	srv, s := newTestServer(t, func() time.Time { return fixed })
	addTodo(s, "t1", "alice", "Do it", true)

	resp := do(t, http.MethodPatch, srv.URL+"/todos/t1", "alice", map[string]any{"completed": false})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	todo := decodeTodo(t, resp)
	if todo.Completed {
		t.Fatalf("expected completed=false")
	}
	if todo.CompletedAt != nil {
		t.Fatalf("expected completed_at=nil, got %v", todo.CompletedAt)
	}
}

func TestPatchIdempotency(t *testing.T) {
	first := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	second := time.Date(2026, 4, 17, 13, 0, 0, 0, time.UTC)
	times := []time.Time{first, second}
	idx := 0
	srv, s := newTestServer(t, func() time.Time {
		t := times[idx]
		if idx < len(times)-1 {
			idx++
		}
		return t
	})
	addTodo(s, "t1", "alice", "Do it", false)

	// First patch - sets timestamp to first
	r1 := do(t, http.MethodPatch, srv.URL+"/todos/t1", "alice", nil)
	defer r1.Body.Close()
	if r1.StatusCode != http.StatusOK {
		t.Fatalf("first patch status %d", r1.StatusCode)
	}
	t1 := decodeTodo(t, r1)

	// Second patch - should NOT change timestamp
	r2 := do(t, http.MethodPatch, srv.URL+"/todos/t1", "alice", nil)
	defer r2.Body.Close()
	if r2.StatusCode != http.StatusOK {
		t.Fatalf("second patch status %d", r2.StatusCode)
	}
	t2 := decodeTodo(t, r2)

	if t1.CompletedAt == nil || t2.CompletedAt == nil {
		t.Fatalf("expected completed_at set")
	}
	if !t1.CompletedAt.Equal(*t2.CompletedAt) {
		t.Fatalf("timestamps changed across idempotent patches: %v != %v", t1.CompletedAt, t2.CompletedAt)
	}
	if !t1.CompletedAt.Equal(first) {
		t.Fatalf("first timestamp unexpected: %v", t1.CompletedAt)
	}
}

func TestPatchNotFound(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	resp := do(t, http.MethodPatch, srv.URL+"/todos/missing", "alice", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestPatchUnauthorized(t *testing.T) {
	srv, s := newTestServer(t, nil)
	addTodo(s, "t1", "alice", "Do it", false)
	resp := do(t, http.MethodPatch, srv.URL+"/todos/t1", "", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestPatchForbiddenOtherOwner(t *testing.T) {
	srv, s := newTestServer(t, nil)
	addTodo(s, "t1", "alice", "Do it", false)
	resp := do(t, http.MethodPatch, srv.URL+"/todos/t1", "bob", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestPatchMalformedBody(t *testing.T) {
	srv, s := newTestServer(t, nil)
	addTodo(s, "t1", "alice", "Do it", false)
	resp := do(t, http.MethodPatch, srv.URL+"/todos/t1", "alice", "{not json")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestPatchUnknownField(t *testing.T) {
	srv, s := newTestServer(t, nil)
	addTodo(s, "t1", "alice", "Do it", false)
	resp := do(t, http.MethodPatch, srv.URL+"/todos/t1", "alice", map[string]any{"nope": 1})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestMethodNotAllowedOnTodoItem(t *testing.T) {
	srv, s := newTestServer(t, nil)
	addTodo(s, "t1", "alice", "Do it", false)
	resp := do(t, http.MethodPut, srv.URL+"/todos/t1", "alice", map[string]any{"title": "x"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
	allow := resp.Header.Get("Allow")
	for _, m := range []string{"GET", "PATCH", "DELETE"} {
		if !strings.Contains(allow, m) {
			t.Fatalf("Allow header missing %s: %q", m, allow)
		}
	}
}

func TestCreateAndListIsolation(t *testing.T) {
	srv, _ := newTestServer(t, nil)

	r1 := do(t, http.MethodPost, srv.URL+"/todos", "alice", map[string]any{"title": "first"})
	defer r1.Body.Close()
	if r1.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", r1.StatusCode)
	}
	alice1 := decodeTodo(t, r1)
	if alice1.Owner != "alice" {
		t.Fatalf("owner = %q, want alice", alice1.Owner)
	}

	r2 := do(t, http.MethodPost, srv.URL+"/todos", "bob", map[string]any{"title": "second"})
	defer r2.Body.Close()
	if r2.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", r2.StatusCode)
	}

	r3 := do(t, http.MethodGet, srv.URL+"/todos", "alice", nil)
	defer r3.Body.Close()
	var todos []models.Todo
	if err := json.NewDecoder(r3.Body).Decode(&todos); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(todos) != 1 || todos[0].Owner != "alice" {
		t.Fatalf("expected only alice's todo, got %+v", todos)
	}
}

func TestGetForbiddenForOtherOwner(t *testing.T) {
	srv, s := newTestServer(t, nil)
	addTodo(s, "t1", "alice", "Do it", false)
	resp := do(t, http.MethodGet, srv.URL+"/todos/t1", "bob", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestDeleteSuccess(t *testing.T) {
	srv, s := newTestServer(t, nil)
	addTodo(s, "t1", "alice", "Do it", false)
	resp := do(t, http.MethodDelete, srv.URL+"/todos/t1", "alice", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}

func TestCreateRequiresTitle(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	resp := do(t, http.MethodPost, srv.URL+"/todos", "alice", map[string]any{"title": ""})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestInvalidIdReturns400(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	resp := do(t, http.MethodPatch, srv.URL+"/todos/", "alice", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
