package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/openchoreo/todo-api/internal/models"
)

var ErrNotFound = errors.New("todo not found")

type Store struct {
	mu       sync.RWMutex
	path     string
	todos    []models.Todo
	indexMap map[string]int
}

func New(path string) (*Store, error) {
	s := &Store{
		path:     path,
		todos:    []models.Todo{},
		indexMap: map[string]int{},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read data file: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	var todos []models.Todo
	if err := json.Unmarshal(data, &todos); err != nil {
		return fmt.Errorf("parse data file: %w", err)
	}
	if todos == nil {
		todos = []models.Todo{}
	}
	s.todos = todos
	s.rebuildIndex()
	return nil
}

func (s *Store) rebuildIndex() {
	s.indexMap = make(map[string]int, len(s.todos))
	for i, t := range s.todos {
		s.indexMap[t.ID] = i
	}
}

func (s *Store) persistLocked() error {
	data, err := json.MarshalIndent(s.todos, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal todos: %w", err)
	}
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, "todos-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

func (s *Store) List() []models.Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.Todo, len(s.todos))
	copy(out, s.todos)
	return out
}

func (s *Store) Add(t models.Todo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.todos = append(s.todos, t)
	s.indexMap[t.ID] = len(s.todos) - 1
	if err := s.persistLocked(); err != nil {
		s.todos = s.todos[:len(s.todos)-1]
		delete(s.indexMap, t.ID)
		return err
	}
	return nil
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, ok := s.indexMap[id]
	if !ok {
		return ErrNotFound
	}
	removed := s.todos[idx]
	s.todos = append(s.todos[:idx], s.todos[idx+1:]...)
	s.rebuildIndex()
	if err := s.persistLocked(); err != nil {
		s.todos = append(s.todos, models.Todo{})
		copy(s.todos[idx+1:], s.todos[idx:])
		s.todos[idx] = removed
		s.rebuildIndex()
		return err
	}
	return nil
}
