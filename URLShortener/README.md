# 09 — URLShortener — URL Shortener + JWT Auth

Сервис коротких ссылок с регистрацией пользователей и защитой эндпоинтов через JWT.

## Применено

- Middleware-цепочки: `middleware.Auth(secret, handler)` оборачивает любой HandlerFunc
- JWT: `golang-jwt/jwt/v5` — HMAC-SHA256, configurable TTL
- Хэширование паролей: `bcrypt` через `golang.org/x/crypto`
- In-memory store с `sync.RWMutex`
- Crypto-safe генерация коротких кодов: `crypto/rand`

## Структура

```
URLShortener/
├── main.go            # точка входа, CLI / interactive, роутинг
├── models/
│   └── models.go      # User, Link
├── store/
│   └── store.go       # потокобезопасный in-memory store
├── middleware/
│   └── auth.go        # JWT middleware + UserIDFromCtx
└── handler/
    ├── auth.go        # POST /register, POST /login
    └── links.go       # POST /shorten, GET /links, GET /{code}
```

## Быстрый старт

```bash
# Интерактивный режим
cd URLShortener && go run .

# Или с флагами
go run . --port 8080 --secret my-secret --ttl 120

# Через переменные окружения
JWT_SECRET=my-secret go run . --port 8080
```

## Эндпоинты

| Method | Path        | Auth     | Описание                        |
|--------|-------------|----------|---------------------------------|
| GET    | /           | —        | Веб-интерфейс                   |
| POST   | /register   | —        | Регистрация (username + password)|
| POST   | /login      | —        | Аутентификация → JWT            |
| POST   | /shorten    | Bearer   | Сократить URL                   |
| GET    | /links      | Bearer   | Список моих ссылок              |
| GET    | /{code}     | —        | Редирект на оригинальный URL    |

## Пример (curl)

```bash
curl -X POST http://localhost:8080/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"secret123"}'

TOKEN=$(curl -s -X POST http://localhost:8080/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"secret123"}' | jq -r .token)

curl -X POST http://localhost:8080/shorten \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://github.com/rxritet/GoLang-Education"}'
```
