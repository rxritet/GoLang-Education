// Package handler реализует HTTP-слой для Books API v2.
//
// Маршруты:
//
//	GET    /books        — список всех книг
//	POST   /books        — добавить книгу
//	GET    /books/{id}   — получить книгу по ID
//	PUT    /books/{id}   — обновить книгу
//	DELETE /books/{id}   — удалить книгу
//	GET    /             — веб-интерфейс
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"bookspostgres/models"
	"bookspostgres/repository"
)

// ---------- Типы ----------

// createRequest — тело POST /books и PUT /books/{id}.
type createRequest struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Year   int    `json:"year"`
}

// errResponse — стандартный ответ об ошибке.
type errResponse struct {
	Error string `json:"error"`
}

// Error messages
const (
	errIDRequired   = "id required"
	errInvalidJSON  = "invalid JSON"
	errBookNotFound = "book not found"
	booksPrefix     = "/books/"
)

// ---------- Handler ----------

// Handler содержит репозиторий и реализует HTTP-обработчики.
type Handler struct {
	repo repository.Repository
}

// New создаёт Handler.
func New(repo repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes регистрирует маршруты на переданном mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", h.Dashboard)
	mux.HandleFunc("GET /books", h.ListBooks)
	mux.HandleFunc("POST /books", h.CreateBook)
	mux.HandleFunc("GET /books/", h.GetBook)
	mux.HandleFunc("PUT /books/", h.UpdateBook)
	mux.HandleFunc("DELETE /books/", h.DeleteBook)
}

// ---------- Handlers ----------

func (h *Handler) ListBooks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	books, err := h.repo.GetAll(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errResponse{Error: err.Error()})
		return
	}
	if books == nil {
		books = []*models.Book{}
	}
	writeJSON(w, http.StatusOK, books)
}

func (h *Handler) CreateBook(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errResponse{Error: errInvalidJSON})
		return
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Author) == "" || req.Year == 0 {
		writeJSON(w, http.StatusBadRequest, errResponse{Error: "title, author, year are required"})
		return
	}

	book := &models.Book{Title: req.Title, Author: req.Author, Year: req.Year}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.repo.Create(ctx, book); err != nil {
		writeJSON(w, http.StatusInternalServerError, errResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, book)
}

func (h *Handler) GetBook(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, booksPrefix)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errResponse{Error: errIDRequired})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	book, err := h.repo.GetByID(ctx, id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errResponse{Error: errBookNotFound})
		return
	}
	writeJSON(w, http.StatusOK, book)
}

func (h *Handler) UpdateBook(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, booksPrefix)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errResponse{Error: errIDRequired})
		return
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errResponse{Error: errInvalidJSON})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	book := &models.Book{ID: id, Title: req.Title, Author: req.Author, Year: req.Year}
	if err := h.repo.Update(ctx, book); err != nil {
		writeJSON(w, http.StatusNotFound, errResponse{Error: errBookNotFound})
		return
	}
	writeJSON(w, http.StatusOK, book)
}

func (h *Handler) DeleteBook(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, booksPrefix)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errResponse{Error: errIDRequired})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.repo.Delete(ctx, id); err != nil {
		writeJSON(w, http.StatusNotFound, errResponse{Error: errBookNotFound})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Dashboard ----------

func (h *Handler) Dashboard(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(dashboardHTML))
}

// ---------- Утилита ----------

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Books API v2 — PostgreSQL</title>
<style>
  *, *::before, *::after { box-sizing: border-box; }
  body { margin: 0; font-family: system-ui, -apple-system, sans-serif; background: #0f172a; color: #e2e8f0; }
  .container { max-width: 900px; margin: 0 auto; padding: 2rem 1rem; }
  h1 { font-size: 1.8rem; margin-bottom: .25rem; }
  p.sub { color: #94a3b8; margin-top: 0; }
  .card { background: #1e293b; border-radius: .75rem; padding: 1.5rem; margin-bottom: 1.5rem; }
  .grid { display: grid; grid-template-columns: 1fr 1fr; gap: .75rem; }
  label { display: block; font-weight: 600; margin-bottom: .3rem; font-size: .9rem; color: #94a3b8; }
  input { width: 100%; padding: .5rem .75rem; border-radius: .5rem; border: 1px solid #334155; background: #0f172a; color: #e2e8f0; font-size: .95rem; }
  input:focus { outline: none; border-color: #6366f1; box-shadow: 0 0 0 3px rgba(99,102,241,.25); }
  button { padding: .5rem 1.2rem; border: none; border-radius: .5rem; font-size: .9rem; font-weight: 600; cursor: pointer; transition: background .15s; }
  .btn-primary { background: #6366f1; color: #fff; }
  .btn-primary:hover { background: #4f46e5; }
  .btn-danger  { background: #ef4444; color: #fff; padding: .3rem .75rem; font-size: .8rem; }
  .btn-danger:hover { background: #dc2626; }
  .btn-secondary { background: #334155; color: #e2e8f0; }
  .btn-secondary:hover { background: #475569; }
  .actions { display: flex; gap: .6rem; margin-top: 1rem; align-items: center; }
  table { width: 100%; border-collapse: collapse; }
  th, td { text-align: left; padding: .6rem .75rem; border-bottom: 1px solid #334155; }
  th { color: #94a3b8; font-size: .8rem; text-transform: uppercase; }
  td.mono { font-family: monospace; font-size: .8rem; color: #64748b; }
  .empty { color: #64748b; text-align: center; padding: 2rem 0; }
  .toast { position: fixed; bottom: 1.5rem; right: 1.5rem; background: #22c55e; color: #fff; padding: .6rem 1.2rem; border-radius: .5rem; font-weight: 600; opacity: 0; transition: opacity .3s; pointer-events: none; }
  .toast.show { opacity: 1; }
  .toast.error { background: #ef4444; }
  .badge-pg { display: inline-flex; align-items: center; gap: .35rem; background: #1e3a5f; color: #60a5fa; border-radius: 9999px; padding: .2rem .7rem; font-size: .8rem; font-weight: 600; }
</style>
</head>
<body>
<div class="container">
  <h1>📚 Books API <span class="badge-pg">🐘 PostgreSQL</span></h1>
  <p class="sub">REST API v2 — database/sql + pgx driver. Full CRUD, connection pooling, transactions.</p>

  <div class="card">
    <div class="grid">
      <div><label for="title">Title</label><input id="title" placeholder="The Go Programming Language"></div>
      <div><label for="author">Author</label><input id="author" placeholder="Donovan & Kernighan"></div>
      <div><label for="year">Year</label><input id="year" type="number" placeholder="2015" min="1"></div>
    </div>
    <div class="actions">
      <button class="btn-primary" onclick="createBook()">Add Book</button>
      <button class="btn-secondary" onclick="loadBooks()">↻ Refresh</button>
      <span id="count" style="color:#64748b;font-size:.85rem;margin-left:auto"></span>
    </div>
  </div>

  <div class="card">
    <div id="books"><p class="empty">Loading…</p></div>
  </div>
</div>
<div class="toast" id="toast"></div>

<script>
function toast(msg, err) {
  const t = document.getElementById('toast');
  t.textContent = msg;
  t.className = 'toast show' + (err ? ' error' : '');
  setTimeout(() => t.className = 'toast', 2800);
}

async function loadBooks() {
  const res = await fetch('/books');
  const books = await res.json();
  const el = document.getElementById('books');
  document.getElementById('count').textContent = books.length + ' book(s)';
  if (!books.length) { el.innerHTML = '<p class="empty">No books yet. Add one above!</p>'; return; }
  let h = '<table><thead><tr><th>Title</th><th>Author</th><th>Year</th><th>ID</th><th></th></tr></thead><tbody>';
  for (const b of books) {
    h += '<tr>'
      + '<td>' + esc(b.title) + '</td>'
      + '<td>' + esc(b.author) + '</td>'
      + '<td>' + b.year + '</td>'
      + '<td class="mono">' + b.id.slice(0,8) + '…</td>'
      + '<td><button class="btn-danger" onclick="deleteBook(\'' + b.id + '\')">Delete</button></td>'
      + '</tr>';
  }
  h += '</tbody></table>';
  el.innerHTML = h;
}

async function createBook() {
  const title  = document.getElementById('title').value.trim();
  const author = document.getElementById('author').value.trim();
  const year   = parseInt(document.getElementById('year').value, 10);
  if (!title || !author || !year) { toast('All fields are required', true); return; }
  const res = await fetch('/books', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({title,author,year}) });
  const data = await res.json();
  if (!res.ok) { toast(data.error || 'Error', true); return; }
  toast('Book added: ' + data.title);
  document.getElementById('title').value = '';
  document.getElementById('author').value = '';
  document.getElementById('year').value = '';
  loadBooks();
}

async function deleteBook(id) {
  if (!confirm('Delete this book?')) return;
  const res = await fetch('/books/' + id, { method: 'DELETE' });
  if (res.status === 204) { toast('Deleted'); loadBooks(); }
  else { toast('Error deleting', true); }
}

function esc(s) { return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }

loadBooks();
setInterval(loadBooks, 10000);
</script>
</body>
</html>`
