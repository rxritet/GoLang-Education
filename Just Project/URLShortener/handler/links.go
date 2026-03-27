// Package handler реализует HTTP-слой: сокращение ссылок и редиректы.
//
// Маршруты:
//
//	POST /shorten    — создать короткую ссылку (требует JWT)
//	GET  /links      — список ссылок текущего пользователя (требует JWT)
//	GET  /{code}     — редирект на оригинальный URL (публичный)
//	GET  /           — веб-интерфейс
package handler

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"urlshortener/middleware"
	"urlshortener/models"
	"urlshortener/store"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const codeLen = 6

// LinksHandler обрабатывает операции со ссылками.
type LinksHandler struct {
	Store *store.Store
}

// NewLinks создаёт LinksHandler.
func NewLinks(s *store.Store) *LinksHandler {
	return &LinksHandler{Store: s}
}

// ---------- POST /shorten ----------

type shortenRequest struct {
	URL string `json:"url"`
}

// Shorten создаёт короткий код для переданного URL.
func (h *LinksHandler) Shorten(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("invalid JSON"))
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, errResp("url is required"))
		return
	}
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		writeJSON(w, http.StatusBadRequest, errResp("url must start with http:// or https://"))
		return
	}

	ownerID := middleware.UserIDFromCtx(r.Context())

	// Генерируем уникальный 6-символьный код.
	code, err := generateCode(h.Store)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errResp("could not generate code"))
		return
	}

	link := &models.Link{
		Code:      code,
		Original:  req.URL,
		OwnerID:   ownerID,
		CreatedAt: time.Now(),
	}

	if err := h.Store.SaveLink(link); err != nil {
		writeJSON(w, http.StatusInternalServerError, errResp(err.Error()))
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"code":     link.Code,
		"short":    "/" + link.Code,
		"original": link.Original,
	})
}

// ---------- GET /links ----------

// ListLinks возвращает ссылки текущего пользователя.
func (h *LinksHandler) ListLinks(w http.ResponseWriter, r *http.Request) {
	ownerID := middleware.UserIDFromCtx(r.Context())
	links := h.Store.ListLinksByOwner(ownerID)
	if links == nil {
		links = []*models.Link{}
	}
	writeJSON(w, http.StatusOK, links)
}

// ---------- GET /{code} ----------

// Redirect выполняет 302-редирект на оригинальный URL.
func (h *LinksHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/")
	if code == "" {
		h.Dashboard(w, r)
		return
	}

	link, err := h.Store.GetLink(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, link.Original, http.StatusFound)
}

// ---------- Dashboard ----------

// Dashboard отдаёт HTML веб-интерфейс.
func (h *LinksHandler) Dashboard(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(dashboardHTML))
}

// ---------- Вспомогательные ----------

// generateCode создаёт случайный 6-символьный код, уникальный в хранилище.
func generateCode(s *store.Store) (string, error) {
	for range 10 {
		code := make([]byte, codeLen)
		for i := range code {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
			if err != nil {
				return "", err
			}
			code[i] = alphabet[n.Int64()]
		}
		c := string(code)
		if _, err := s.GetLink(c); err == store.ErrLinkNotFound {
			return c, nil
		}
	}
	return uuid.NewString()[:8], nil
}

func errResp(msg string) map[string]string { return map[string]string{"error": msg} }

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
<title>URL Shortener + JWT</title>
<style>
  *, *::before, *::after { box-sizing: border-box; }
  body { margin: 0; font-family: system-ui, -apple-system, sans-serif; background: #0f172a; color: #e2e8f0; min-height: 100vh; }
  .page { max-width: 860px; margin: 0 auto; padding: 2rem 1rem; }
  h1 { font-size: 1.9rem; margin-bottom: .2rem; }
  .sub { color: #94a3b8; margin-top: 0; margin-bottom: 2rem; }
  .tabs { display: flex; gap: .5rem; margin-bottom: 1.5rem; }
  .tab { padding: .45rem 1.1rem; border-radius: .5rem; border: none; font-weight: 600; cursor: pointer; background: #1e293b; color: #94a3b8; transition: all .15s; }
  .tab.active { background: #6366f1; color: #fff; }
  .card { background: #1e293b; border-radius: .75rem; padding: 1.5rem; margin-bottom: 1.25rem; }
  label { display: block; font-weight: 600; font-size: .88rem; color: #94a3b8; margin-bottom: .3rem; }
  input { width: 100%; padding: .5rem .75rem; border-radius: .5rem; border: 1px solid #334155; background: #0f172a; color: #e2e8f0; font-size: .95rem; margin-bottom: .75rem; }
  input:focus { outline: none; border-color: #6366f1; box-shadow: 0 0 0 3px rgba(99,102,241,.2); }
  button { padding: .5rem 1.2rem; border: none; border-radius: .5rem; font-size: .9rem; font-weight: 600; cursor: pointer; transition: background .15s; }
  .btn-primary { background: #6366f1; color: #fff; }
  .btn-primary:hover { background: #4f46e5; }
  .btn-sm { padding: .3rem .75rem; font-size: .8rem; background: #334155; color: #e2e8f0; }
  .btn-sm:hover { background: #475569; }
  .notice { font-size: .85rem; margin-top: .5rem; color: #94a3b8; }
  table { width: 100%; border-collapse: collapse; margin-top: .5rem; }
  th, td { text-align: left; padding: .55rem .75rem; border-bottom: 1px solid #334155; }
  th { font-size: .8rem; text-transform: uppercase; color: #64748b; }
  .mono { font-family: monospace; font-size: .85rem; }
  .link-code { color: #818cf8; font-weight: 700; cursor:pointer; text-decoration:underline dotted; }
  .empty { color: #475569; text-align: center; padding: 1.5rem 0; }
  .badge-token { background: #134e4a; color: #34d399; border-radius: 9999px; padding: .2rem .75rem; font-size: .78rem; font-family: monospace; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 300px; display: inline-block; vertical-align: middle; }
  .status-bar { display: flex; align-items: center; gap: .75rem; padding: .6rem 1rem; background: #1e293b; border-radius: .5rem; margin-bottom: 1.5rem; font-size: .88rem; }
  .dot { width: .6rem; height: .6rem; border-radius: 50%; }
  .dot.green { background: #22c55e; }  .dot.red { background: #ef4444; }
  .toast { position: fixed; bottom: 1.5rem; right: 1.5rem; background: #22c55e; color: #fff; padding: .6rem 1.2rem; border-radius: .5rem; font-weight: 600; opacity: 0; transition: opacity .3s; }
  .toast.show { opacity: 1; }  .toast.error { background: #ef4444; }
  .pane { display: none; } .pane.active { display: block; }
</style>
</head>
<body>
<div class="page">
  <h1>🔗 URL Shortener</h1>
  <p class="sub">JWT Auth · bcrypt · In-memory store · Middleware chains</p>

  <div id="statusBar" class="status-bar">
    <span class="dot red" id="statusDot"></span>
    <span id="statusText">Not logged in</span>
    <button class="btn-sm" id="logoutBtn" onclick="logout()" style="display:none;margin-left:auto">Log out</button>
  </div>

  <div class="tabs">
    <button class="tab active" onclick="showTab('auth')">🔐 Auth</button>
    <button class="tab" onclick="showTab('shorten')">✂️ Shorten</button>
    <button class="tab" onclick="showTab('links')">📋 My Links</button>
  </div>

  <!-- Auth Tab -->
  <div id="pane-auth" class="pane active">
    <div class="card">
      <label for="u">Username</label><input id="u" placeholder="alice">
      <label for="p">Password</label><input id="p" type="password" placeholder="min 6 chars">
      <div style="display:flex;gap:.6rem">
        <button class="btn-primary" onclick="register()">Register</button>
        <button class="btn-primary" style="background:#0f766e" onclick="login()">Login</button>
      </div>
    </div>
  </div>

  <!-- Shorten Tab -->
  <div id="pane-shorten" class="pane">
    <div class="card">
      <label for="url">Long URL</label>
      <input id="url" placeholder="https://example.com/very/long/path">
      <button class="btn-primary" onclick="shorten()">Shorten</button>
      <div id="result"></div>
    </div>
  </div>

  <!-- Links Tab -->
  <div id="pane-links" class="pane">
    <div class="card">
      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:.75rem">
        <b>My Links</b>
        <button class="btn-sm" onclick="loadLinks()">↻ Refresh</button>
      </div>
      <div id="linksTable"><p class="empty">Load your links above.</p></div>
    </div>
  </div>
</div>
<div class="toast" id="toast"></div>

<script>
let token = localStorage.getItem('jwt') || '';
updateStatus();

function showTab(id) {
  document.querySelectorAll('.pane').forEach(p => p.classList.remove('active'));
  document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
  document.getElementById('pane-' + id).classList.add('active');
  event.target.classList.add('active');
  if (id === 'links') loadLinks();
}

function toast(msg, err) {
  const t = document.getElementById('toast');
  t.textContent = msg; t.className = 'toast show' + (err ? ' error' : '');
  setTimeout(() => t.className = 'toast', 2800);
}

function updateStatus() {
  const dot = document.getElementById('statusDot');
  const txt = document.getElementById('statusText');
  const btn = document.getElementById('logoutBtn');
  if (token) {
    try {
      const payload = JSON.parse(atob(token.split('.')[1]));
      dot.className = 'dot green'; txt.textContent = 'Logged in · sub: ' + payload.sub.slice(0,8) + '…';
      btn.style.display = 'inline-block';
    } catch { dot.className = 'dot red'; txt.textContent = 'Invalid token'; }
  } else {
    dot.className = 'dot red'; txt.textContent = 'Not logged in'; btn.style.display = 'none';
  }
}

function logout() { token = ''; localStorage.removeItem('jwt'); updateStatus(); toast('Logged out'); }

async function register() {
  const username = document.getElementById('u').value.trim();
  const password = document.getElementById('p').value;
  const res = await fetch('/register', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({username, password}) });
  const data = await res.json();
  if (!res.ok) { toast(data.error || 'Error', true); return; }
  toast('Registered as ' + data.username + ' — now log in!');
}

async function login() {
  const username = document.getElementById('u').value.trim();
  const password = document.getElementById('p').value;
  const res = await fetch('/login', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({username, password}) });
  const data = await res.json();
  if (!res.ok) { toast(data.error || 'Error', true); return; }
  token = data.token; localStorage.setItem('jwt', token);
  updateStatus(); toast('Welcome back, ' + username + '!');
}

async function shorten() {
  if (!token) { toast('Please login first', true); return; }
  const url = document.getElementById('url').value.trim();
  const res = await fetch('/shorten', { method:'POST', headers:{'Content-Type':'application/json','Authorization':'Bearer '+token}, body: JSON.stringify({url}) });
  const data = await res.json();
  if (!res.ok) { toast(data.error || 'Error', true); return; }
  const shortURL = window.location.origin + data.short;
  document.getElementById('result').innerHTML = '<p class="notice">Short URL: <a href="' + shortURL + '" target="_blank" style="color:#818cf8">' + shortURL + '</a> &nbsp;<button class="btn-sm" onclick="navigator.clipboard.writeText(\'' + shortURL + '\').then(()=>toast(\'Copied!\'))">Copy</button></p>';
  toast('Link created!');
}

async function loadLinks() {
  if (!token) { toast('Please login first', true); return; }
  const res = await fetch('/links', { headers: {'Authorization':'Bearer '+token} });
  const links = await res.json();
  const el = document.getElementById('linksTable');
  if (!res.ok) { el.innerHTML = '<p class="empty">Error loading links</p>'; return; }
  if (!links.length) { el.innerHTML = '<p class="empty">No links yet. Shorten some URLs!</p>'; return; }
  let h = '<table><thead><tr><th>Code</th><th>Original URL</th><th>Created</th></tr></thead><tbody>';
  for (const l of links) {
    h += '<tr>'
      + '<td class="mono"><span class="link-code" onclick="window.open(\'/\'+\'' + l.code + '\',\'_blank\')">' + l.code + '</span></td>'
      + '<td style="max-width:360px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap" title="' + esc(l.original) + '">' + esc(l.original) + '</td>'
      + '<td class="mono">' + new Date(l.created_at).toLocaleString() + '</td>'
      + '</tr>';
  }
  h += '</tbody></table>';
  el.innerHTML = h;
}

function esc(s) { return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }
</script>
</body>
</html>`
