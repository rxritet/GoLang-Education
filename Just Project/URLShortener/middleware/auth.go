// Package middleware реализует JWT-аутентификацию для HTTP-обработчиков.
//
// Использование:
//
//	mux.HandleFunc("POST /shorten", middleware.Auth(secret, handler.Shorten))
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ctxKey — тип для ключей контекста (избегаем коллизий).
type ctxKey string

const UserIDKey ctxKey = "user_id"

// Auth возвращает middleware, проверяющий JWT в заголовке Authorization.
// При успехе — инжектирует user_id в контекст запроса.
func Auth(secret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing Authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, `{"error":"invalid Authorization format"}`, http.StatusUnauthorized)
			return
		}

		tokenStr := parts[1]
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, `{"error":"invalid claims"}`, http.StatusUnauthorized)
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			http.Error(w, `{"error":"invalid subject claim"}`, http.StatusUnauthorized)
			return
		}

		// Передаём user_id дальше через контекст.
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next(w, r.WithContext(ctx))
	}
}

// UserIDFromCtx извлекает user_id из контекста.
func UserIDFromCtx(ctx context.Context) string {
	id, _ := ctx.Value(UserIDKey).(string)
	return id
}
