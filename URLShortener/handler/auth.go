// Package handler реализует HTTP-слой для URL Shortener.
//
// Auth-маршруты:
//
//	POST /register — регистрация (username + password)
//	POST /login    — аутентификация, возвращает JWT
package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"urlshortener/models"
	"urlshortener/store"
)

// AuthHandler обрабатывает регистрацию и вход.
type AuthHandler struct {
	Store     *store.Store
	JWTSecret string
	TokenTTL  time.Duration
}

// NewAuth создаёт AuthHandler.
func NewAuth(s *store.Store, secret string, ttl time.Duration) *AuthHandler {
	return &AuthHandler{Store: s, JWTSecret: secret, TokenTTL: ttl}
}

// ---------- POST /register ----------

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register создаёт нового пользователя.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("invalid JSON"))
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || len(req.Password) < 6 {
		writeJSON(w, http.StatusBadRequest, errResp("username required; password min 6 chars"))
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errResp("hash error"))
		return
	}

	user := &models.User{
		ID:           uuid.NewString(),
		Username:     req.Username,
		PasswordHash: string(hash),
	}

	if err := h.Store.SaveUser(user); err != nil {
		if err == store.ErrUserExists {
			writeJSON(w, http.StatusConflict, errResp("username already taken"))
			return
		}
		writeJSON(w, http.StatusInternalServerError, errResp(err.Error()))
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"id":       user.ID,
		"username": user.Username,
	})
}

// ---------- POST /login ----------

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login проверяет пароль и возвращает JWT.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("invalid JSON"))
		return
	}

	user, err := h.Store.GetUserByUsername(req.Username)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errResp("invalid credentials"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, errResp("invalid credentials"))
		return
	}

	// Создаём JWT.
	claims := jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(h.TokenTTL).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(h.JWTSecret))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errResp("could not sign token"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token":      signed,
		"expires_in": int(h.TokenTTL.Seconds()),
	})
}
