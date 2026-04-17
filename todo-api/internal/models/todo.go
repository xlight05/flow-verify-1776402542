package models

import "time"

type Todo struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTodoRequest struct {
	Title     string `json:"title"`
	Completed *bool  `json:"completed,omitempty"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}
