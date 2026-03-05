// Package store реализует потокобезопасное in-memory хранилище пользователей и ссылок.
package store

import (
	"errors"
	"sync"

	"urlshortener/models"
)

// Sentinel-ошибки для чистой обработки в хендлерах.
var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("username already taken")
	ErrLinkNotFound = errors.New("link not found")
	ErrCodeExists   = errors.New("short code already exists")
)

// Store хранит пользователей и ссылки в памяти.
type Store struct {
	mu         sync.RWMutex
	users      map[string]*models.User // id → User
	byUsername map[string]*models.User // username → User
	links      map[string]*models.Link // code → Link
}

// New создаёт пустой Store.
func New() *Store {
	return &Store{
		users:      make(map[string]*models.User),
		byUsername: make(map[string]*models.User),
		links:      make(map[string]*models.Link),
	}
}

// ---------- Users ----------

// SaveUser добавляет нового пользователя. Возвращает ErrUserExists, если имя занято.
func (s *Store) SaveUser(u *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.byUsername[u.Username]; ok {
		return ErrUserExists
	}
	s.users[u.ID] = u
	s.byUsername[u.Username] = u
	return nil
}

// GetUserByUsername возвращает пользователя по имени.
func (s *Store) GetUserByUsername(username string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.byUsername[username]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

// GetUserByID возвращает пользователя по ID.
func (s *Store) GetUserByID(id string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

// ---------- Links ----------

// SaveLink сохраняет ссылку. Возвращает ErrCodeExists, если код занят.
func (s *Store) SaveLink(l *models.Link) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.links[l.Code]; ok {
		return ErrCodeExists
	}
	s.links[l.Code] = l
	return nil
}

// GetLink возвращает ссылку по коду.
func (s *Store) GetLink(code string) (*models.Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	l, ok := s.links[code]
	if !ok {
		return nil, ErrLinkNotFound
	}
	return l, nil
}

// ListLinksByOwner возвращает все ссылки указанного пользователя.
func (s *Store) ListLinksByOwner(ownerID string) []*models.Link {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []*models.Link
	for _, l := range s.links {
		if l.OwnerID == ownerID {
			out = append(out, l)
		}
	}
	return out
}
