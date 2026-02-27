// Package handler реализует HTTP-слой (эндпоинты) для Job Queue сервера.
//
// Маршруты:
//
//	POST /jobs      — создать задачу, вернуть ID
//	GET  /jobs/{id} — получить статус задачи по ID
//	GET  /jobs      — список всех задач
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"jobqueue/store"
	"jobqueue/worker"
)

// ---------- Типы запросов / ответов ----------

// CreateJobRequest — тело JSON для POST /jobs.
type CreateJobRequest struct {
	Task string `json:"task"`
}

// CreateJobResponse — ответ на успешное создание задачи.
type CreateJobResponse struct {
	ID     string       `json:"id"`
	Status store.Status `json:"status"`
}

// ErrorResponse — стандартный ответ об ошибке.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ---------- Handler ----------

// Handler содержит зависимости (store, pool) и предоставляет ServeHTTP.
type Handler struct {
	Store *store.MemoryStore
	Pool  *worker.Pool
}

// New создаёт Handler с переданными зависимостями.
func New(s *store.MemoryStore, p *worker.Pool) *Handler {
	return &Handler{Store: s, Pool: p}
}

// RegisterRoutes регистрирует маршруты на переданном mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", h.Dashboard) // корневая страница — веб-панель
	mux.HandleFunc("POST /jobs", h.CreateJob)
	mux.HandleFunc("GET /jobs/", h.GetJob) // Go 1.22+ поддержит wildcard; здесь парсим руками
	mux.HandleFunc("GET /jobs", h.ListJobs)
}

// ---------- POST /jobs ----------

// CreateJob принимает JSON {"task":"..."}, создаёт Job и ставит в очередь.
func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON: " + err.Error()})
		return
	}
	if strings.TrimSpace(req.Task) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "field 'task' is required"})
		return
	}

	// Создаём задачу со статусом «queued».
	job := &store.Job{
		ID:        uuid.NewString(),
		Task:      req.Task,
		Status:    store.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Сохраняем в хранилище (потокобезопасно через Lock).
	h.Store.Save(job)

	// Помещаем в канал воркер-пула (неблокирующий select внутри Submit).
	if !h.Pool.Submit(job.ID) {
		// Очередь переполнена — откатываем статус.
		_ = h.Store.UpdateStatus(job.ID, store.StatusFailed, "queue is full")
		writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: "job queue is full, try later"})
		return
	}

	writeJSON(w, http.StatusAccepted, CreateJobResponse{
		ID:     job.ID,
		Status: job.Status,
	})
}

// ---------- GET /jobs/{id} ----------

// GetJob возвращает текущее состояние задачи по ID.
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из пути: /jobs/{id}
	id := strings.TrimPrefix(r.URL.Path, "/jobs/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "job ID is required"})
		return
	}

	job, err := h.Store.Get(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: fmt.Sprintf("job %q not found", id)})
		return
	}

	writeJSON(w, http.StatusOK, job)
}

// ---------- GET /jobs ----------

// ListJobs возвращает все задачи.
func (h *Handler) ListJobs(w http.ResponseWriter, _ *http.Request) {
	jobs := h.Store.List()
	writeJSON(w, http.StatusOK, jobs)
}

// ---------- Утилита ----------

// writeJSON сериализует payload и отправляет с правильным Content-Type.
func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

// ---------- GET / (Dashboard) ----------

// Dashboard отдаёт HTML-страницу с интерфейсом для создания задач и просмотра статусов.
func (h *Handler) Dashboard(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Job Queue Dashboard</title>
<style>
  *, *::before, *::after { box-sizing: border-box; }
  body { margin: 0; font-family: system-ui, -apple-system, sans-serif; background: #0f172a; color: #e2e8f0; }
  .container { max-width: 800px; margin: 0 auto; padding: 2rem 1rem; }
  h1 { font-size: 1.8rem; margin-bottom: .5rem; }
  p.sub { color: #94a3b8; margin-top: 0; }
  .card { background: #1e293b; border-radius: .75rem; padding: 1.5rem; margin-bottom: 1.5rem; }
  label { display: block; font-weight: 600; margin-bottom: .4rem; }
  input[type=text] { width: 100%; padding: .55rem .75rem; border-radius: .5rem; border: 1px solid #334155; background: #0f172a; color: #e2e8f0; font-size: 1rem; }
  input[type=text]:focus { outline: none; border-color: #3b82f6; box-shadow: 0 0 0 3px rgba(59,130,246,.3); }
  button { padding: .55rem 1.25rem; border: none; border-radius: .5rem; font-size: .95rem; font-weight: 600; cursor: pointer; transition: background .15s; }
  .btn-primary { background: #3b82f6; color: #fff; }
  .btn-primary:hover { background: #2563eb; }
  .btn-secondary { background: #334155; color: #e2e8f0; }
  .btn-secondary:hover { background: #475569; }
  .actions { display: flex; gap: .75rem; margin-top: 1rem; }
  table { width: 100%; border-collapse: collapse; margin-top: 1rem; }
  th, td { text-align: left; padding: .6rem .75rem; border-bottom: 1px solid #334155; }
  th { color: #94a3b8; font-weight: 600; font-size: .85rem; text-transform: uppercase; }
  .badge { display: inline-block; padding: .15rem .55rem; border-radius: 9999px; font-size: .8rem; font-weight: 600; }
  .badge-queued    { background: #fbbf24; color: #1e293b; }
  .badge-running   { background: #3b82f6; color: #fff; }
  .badge-completed { background: #22c55e; color: #fff; }
  .badge-failed    { background: #ef4444; color: #fff; }
  .badge-cancelled { background: #a855f7; color: #fff; }
  .toast { position: fixed; bottom: 1.5rem; right: 1.5rem; background: #22c55e; color: #fff; padding: .6rem 1.2rem; border-radius: .5rem; font-weight: 600; opacity: 0; transition: opacity .3s; }
  .toast.show { opacity: 1; }
  .toast.error { background: #ef4444; }
  .empty { color: #64748b; text-align: center; padding: 2rem 0; }
  .mono { font-family: ui-monospace, monospace; font-size: .85rem; }
</style>
</head>
<body>
<div class="container">
  <h1>&#9881; Job Queue Dashboard</h1>
  <p class="sub">Create tasks and monitor worker pool status in real time.</p>

  <div class="card">
    <label for="task">Task name</label>
    <input type="text" id="task" placeholder='e.g. send_email, resize_image, generate_report'>
    <div class="actions">
      <button class="btn-primary" onclick="createJob()">Create Job</button>
      <button class="btn-secondary" onclick="loadJobs()">&#x21bb; Refresh</button>
    </div>
  </div>

  <div class="card">
    <label>Jobs</label>
    <div id="jobs"><p class="empty">No jobs yet. Create one above!</p></div>
  </div>
</div>

<div class="toast" id="toast"></div>

<script>
function showToast(msg, isError) {
  const t = document.getElementById('toast');
  t.textContent = msg;
  t.className = 'toast show' + (isError ? ' error' : '');
  setTimeout(() => t.className = 'toast', 2500);
}

async function createJob() {
  const task = document.getElementById('task').value.trim();
  if (!task) { showToast('Enter a task name', true); return; }
  try {
    const res = await fetch('/jobs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ task })
    });
    const data = await res.json();
    if (!res.ok) { showToast(data.error || 'Error', true); return; }
    showToast('Job created: ' + data.id.slice(0, 8) + '…');
    document.getElementById('task').value = '';
    loadJobs();
  } catch (e) { showToast('Network error', true); }
}

function badgeClass(status) {
  return 'badge badge-' + status;
}

async function loadJobs() {
  try {
    const res = await fetch('/jobs');
    const jobs = await res.json();
    const el = document.getElementById('jobs');
    if (!jobs || jobs.length === 0) {
      el.innerHTML = '<p class="empty">No jobs yet. Create one above!</p>';
      return;
    }
    jobs.sort((a, b) => new Date(b.created_at) - new Date(a.created_at));
    let html = '<table><thead><tr><th>ID</th><th>Task</th><th>Status</th><th>Error</th><th>Updated</th></tr></thead><tbody>';
    for (const j of jobs) {
      const updated = new Date(j.updated_at).toLocaleTimeString();
      html += '<tr>'
        + '<td class="mono">' + j.id.slice(0, 8) + '…</td>'
        + '<td>' + j.task + '</td>'
        + '<td><span class="' + badgeClass(j.status) + '">' + j.status + '</span></td>'
        + '<td>' + (j.error || '—') + '</td>'
        + '<td>' + updated + '</td>'
        + '</tr>';
    }
    html += '</tbody></table>';
    el.innerHTML = html;
  } catch (e) { console.error(e); }
}

// Enter key submits.
document.getElementById('task').addEventListener('keydown', e => { if (e.key === 'Enter') createJob(); });

// Auto-refresh every 2s.
loadJobs();
setInterval(loadJobs, 2000);
</script>
</body>
</html>`
