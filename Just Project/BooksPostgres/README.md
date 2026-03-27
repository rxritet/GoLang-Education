# 08 — BooksPostgres — Books API v2 + PostgreSQL

Рефакторинг `BookManager`: замена потокобезопасного in-memory хранилища на PostgreSQL.

## Применено

- `database/sql`-паттерн через `pgx/v5` pgxpool
- Интерфейс `Repository` — единый контракт для хранилища
- SQL-миграция через `docker-entrypoint-initdb.d/`
- Транзакционно-безопасный пул соединений
- Graceful shutdown сервера

## Структура

```
BooksPostgres/
├── main.go                  # точка входа, CLI / interactive, DB init
├── docker-compose.yml       # postgres:16-alpine
├── migrations/
│   └── 001_init.sql         # CREATE TABLE books
├── models/
│   └── book.go              # Book struct
├── repository/
│   └── repository.go        # Repository interface + PostgresRepo (pgx)
└── handler/
    └── handler.go           # HTTP CRUD + веб-интерфейс
```

## Быстрый старт

```bash
# 1. Поднять PostgreSQL
docker-compose up -d

# 2. Запустить сервер (интерактивный режим)
cd BooksPostgres && go run .

# Или с флагами
go run . --port 8080 --dsn "postgres://books:books@localhost:5432/booksdb?sslmode=disable"

# Через переменную окружения
DATABASE_URL=postgres://books:books@localhost:5432/booksdb?sslmode=disable go run .
```

## Эндпоинты

| Method | Path          | Описание             |
|--------|---------------|----------------------|
| GET    | /             | Веб-интерфейс        |
| GET    | /books        | Список всех книг     |
| POST   | /books        | Добавить книгу       |
| GET    | /books/{id}   | Получить по UUID     |
| PUT    | /books/{id}   | Обновить книгу       |
| DELETE | /books/{id}   | Удалить книгу        |

## Пример

```bash
curl -X POST http://localhost:8080/books \
  -H 'Content-Type: application/json' \
  -d '{"title":"The Go Programming Language","author":"Donovan","year":2015}'

curl http://localhost:8080/books
```
