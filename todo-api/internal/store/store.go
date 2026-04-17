package store

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/openchoreo/todo-api/internal/models"
)

var (
	ErrNotFound  = errors.New("todo not found")
	ErrForbidden = errors.New("forbidden")
)

type Store struct {
	mu    sync.RWMutex
	todos map[string]*models.Todo
}

func New() *Store {
	return &Store{todos: make(map[string]*models.Todo)}
}

func (s *Store) Add(t *models.Todo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := cloneTodo(t)
	s.todos[cp.ID] = cp
}

func (s *Store) Get(id, owner string) (*models.Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.todos[id]
	if !ok {
		return nil, ErrNotFound
	}
	if t.Owner != owner {
		return nil, ErrForbidden
	}
	return cloneTodo(t), nil
}

func (s *Store) ListByOwner(owner string) []*models.Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.Todo, 0)
	for _, t := range s.todos {
		if t.Owner == owner {
			out = append(out, cloneTodo(t))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

func (s *Store) Delete(id, owner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.todos[id]
	if !ok {
		return ErrNotFound
	}
	if t.Owner != owner {
		return ErrForbidden
	}
	delete(s.todos, id)
	return nil
}

func (s *Store) SetCompleted(id, owner string, completed bool, now time.Time) (*models.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.todos[id]
	if !ok {
		return nil, ErrNotFound
	}
	if t.Owner != owner {
		return nil, ErrForbidden
	}
	if completed {
		if !t.Completed {
			t.Completed = true
			ts := now
			t.CompletedAt = &ts
		}
	} else {
		t.Completed = false
		t.CompletedAt = nil
	}
	return cloneTodo(t), nil
}

func cloneTodo(t *models.Todo) *models.Todo {
	cp := *t
	if t.CompletedAt != nil {
		ts := *t.CompletedAt
		cp.CompletedAt = &ts
	}
	return &cp
}
