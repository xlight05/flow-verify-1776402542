package models

import "time"

type Todo struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Owner       string     `json:"owner"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
