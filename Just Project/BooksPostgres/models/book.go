// Package models содержит доменные структуры приложения.
package models

import "time"

// Book представляет книгу в каталоге.
type Book struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Year      int       `json:"year"`
	CreatedAt time.Time `json:"created_at"`
}
