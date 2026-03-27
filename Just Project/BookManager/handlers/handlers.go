package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"thirdproject/models"
)

// Константы для повторяющихся сообщений об ошибках
const (
	errBadID    = "некорректный ID"
	errNotFound = "книга не найдена"
)

// Handler хранит зависимости для всех HTTP-обработчиков
type Handler struct {
	store *models.Store
}

// New создаёт новый Handler с переданным хранилищем
func New(store *models.Store) *Handler {
	return &Handler{store: store}
}

// ---------- вспомогательные функции ----------

// writeJSON отправляет JSON-ответ с указанным статус-кодом
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError отправляет JSON-ошибку
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// parseID извлекает числовой ID из последнего сегмента URL (/api/books/42 → 42)
func parseID(r *http.Request) (int, error) {
	parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
	return strconv.Atoi(parts[len(parts)-1])
}

// ---------- маршрутизатор ----------

// BooksRouter направляет запросы к /api/books и /api/books/{id}
func (h *Handler) BooksRouter(w http.ResponseWriter, r *http.Request) {
	// Включаем CORS для удобства разработки
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// /api/books  → коллекция
	// /api/books/ → тоже коллекция
	// /api/books/42 → конкретная книга
	path := strings.TrimRight(r.URL.Path, "/")
	isCollection := path == "/api/books"

	if isCollection {
		switch r.Method {
		case http.MethodGet:
			h.GetAllBooks(w, r)
		case http.MethodPost:
			h.CreateBook(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "метод не поддерживается")
		}
		return
	}

	// Работа с конкретной книгой
	switch r.Method {
	case http.MethodGet:
		h.GetBook(w, r)
	case http.MethodPut:
		h.UpdateBook(w, r)
	case http.MethodDelete:
		h.DeleteBook(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "метод не поддерживается")
	}
}

// ---------- CRUD-обработчики ----------

// GetAllBooks   GET /api/books
// Возвращает список всех книг
func (h *Handler) GetAllBooks(w http.ResponseWriter, r *http.Request) {
	books := h.store.GetAll()
	writeJSON(w, http.StatusOK, books)
}

// GetBook   GET /api/books/{id}
// Возвращает книгу по ID
func (h *Handler) GetBook(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, errBadID)
		return
	}

	book, ok := h.store.GetByID(id)
	if !ok {
		writeError(w, http.StatusNotFound, errNotFound)
		return
	}

	writeJSON(w, http.StatusOK, book)
}

// CreateBook   POST /api/books
// Создаёт новую книгу из тела запроса (JSON)
func (h *Handler) CreateBook(w http.ResponseWriter, r *http.Request) {
	var book models.Book
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		writeError(w, http.StatusBadRequest, "неверный формат JSON")
		return
	}
	if book.Title == "" || book.Author == "" {
		writeError(w, http.StatusBadRequest, "поля title и author обязательны")
		return
	}

	created := h.store.Create(book)
	writeJSON(w, http.StatusCreated, created)
}

// UpdateBook   PUT /api/books/{id}
// Полностью заменяет книгу по ID
func (h *Handler) UpdateBook(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, errBadID)
		return
	}

	var book models.Book
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		writeError(w, http.StatusBadRequest, "неверный формат JSON")
		return
	}
	if book.Title == "" || book.Author == "" {
		writeError(w, http.StatusBadRequest, "поля title и author обязательны")
		return
	}

	updated, ok := h.store.Update(id, book)
	if !ok {
		writeError(w, http.StatusNotFound, errNotFound)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// DeleteBook   DELETE /api/books/{id}
// Удаляет книгу по ID
func (h *Handler) DeleteBook(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, errBadID)
		return
	}

	if !h.store.Delete(id) {
		writeError(w, http.StatusNotFound, errNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "книга удалена"})
}
