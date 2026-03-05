// Package models содержит доменные структуры: пользователь и короткая ссылка.
package models

import "time"

// User представляет зарегистрированного пользователя.
type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"` // никогда не сериализуется в JSON
}

// Link представляет сокращённую ссылку.
type Link struct {
	Code      string    `json:"code"`
	Original  string    `json:"original"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}
