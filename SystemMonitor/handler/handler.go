// Package handler реализует HTTP-слой для System Monitor API.
//
// Маршруты:
//
//	GET /          — веб-дашборд с автообновлением метрик
//	GET /metrics   — JSON-снимок последних метрик
//	GET /health    — простой health-check {status: "ok"}
package handler

import (
	"encoding/json"
	"net/http"

	"sysmonitor/collector"
)

// Handler содержит зависимость от Collector.
type Handler struct {
	Collector *collector.Collector
}

// New создаёт Handler.
func New(c *collector.Collector) *Handler {
	return &Handler{Collector: c}
}

// RegisterRoutes регистрирует маршруты на переданном mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", h.Dashboard)
	mux.HandleFunc("GET /metrics", h.GetMetrics)
	mux.HandleFunc("GET /health", h.Health)
}

// ---------- GET /metrics ----------

// GetMetrics возвращает последний снимок метрик в формате JSON.
func (h *Handler) GetMetrics(w http.ResponseWriter, _ *http.Request) {
	snapshot := h.Collector.Snapshot()
	writeJSON(w, http.StatusOK, snapshot)
}

// ---------- GET /health ----------

// Health — минимальный health-check.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------- GET / ----------

// Dashboard отдаёт HTML-страницу с визуализацией метрик.
func (h *Handler) Dashboard(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(dashboardHTML))
}

// ---------- Утилиты ----------

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
<title>System Monitor</title>
<style>
  *,*::before,*::after{box-sizing:border-box}
  body{margin:0;font-family:system-ui,-apple-system,sans-serif;background:#0f172a;color:#e2e8f0}
  .container{max-width:900px;margin:0 auto;padding:2rem 1rem}
  h1{font-size:1.8rem;margin-bottom:.3rem}
  .sub{color:#94a3b8;margin-top:0}
  .grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(200px,1fr));gap:1rem;margin-bottom:1.5rem}
  .card{background:#1e293b;border-radius:.75rem;padding:1.2rem}
  .card .label{font-size:.78rem;color:#94a3b8;text-transform:uppercase;font-weight:600;margin-bottom:.3rem}
  .card .value{font-size:1.6rem;font-weight:700}
  .badge{display:inline-block;padding:.15rem .55rem;border-radius:9999px;font-size:.75rem;font-weight:600;background:#22c55e;color:#fff;margin-left:.5rem}
  .meta{background:#1e293b;border-radius:.75rem;padding:1.2rem;margin-top:1rem}
  .meta table{width:100%;border-collapse:collapse}
  .meta td{padding:.4rem .6rem;border-bottom:1px solid #334155}
  .meta td:first-child{color:#94a3b8;font-weight:600;width:40%}
  .mono{font-family:ui-monospace,monospace;font-size:.9rem}
  .dot{display:inline-block;width:8px;height:8px;border-radius:50%;background:#22c55e;margin-right:.4rem;animation:pulse 2s infinite}
  @keyframes pulse{0%,100%{opacity:1}50%{opacity:.4}}
</style>
</head>
<body>
<div class="container">
  <h1><span class="dot"></span> System Monitor</h1>
  <p class="sub">Live runtime metrics — auto-refreshes every 3 seconds</p>

  <div class="grid" id="cards"></div>

  <div class="meta">
    <table id="meta"></table>
  </div>
</div>

<script>
function fmt(bytes){
  if(bytes<1024)return bytes+' B';
  if(bytes<1048576)return (bytes/1024).toFixed(1)+' KB';
  return (bytes/1048576).toFixed(2)+' MB';
}

function card(label,value){
  return '<div class="card"><div class="label">'+label+'</div><div class="value">'+value+'</div></div>';
}

function row(k,v){
  return '<tr><td>'+k+'</td><td class="mono">'+v+'</td></tr>';
}

async function refresh(){
  try{
    const r=await fetch('/metrics');
    const m=await r.json();
    document.getElementById('cards').innerHTML=
      card('Alloc Memory',fmt(m.alloc_bytes))
      +card('Heap Objects',m.heap_objects.toLocaleString())
      +card('Goroutines',m.num_goroutines)
      +card('GC Cycles',m.num_gc)
      +card('GC Pause',((m.gc_pause_ns||0)/1e6).toFixed(2)+' ms')
      +card('Sys Memory',fmt(m.sys_bytes));

    document.getElementById('meta').innerHTML=
      row('Go Version',m.go_version)
      +row('OS / Arch',m.goos+' / '+m.goarch)
      +row('CPUs',m.num_cpu)
      +row('Total Alloc',fmt(m.total_alloc_bytes))
      +row('Heap Sys',fmt(m.heap_sys_bytes))
      +row('GC CPU %',m.gc_cpu_percent.toFixed(4)+'%')
      +row('Uptime',m.uptime)
      +row('Snapshot',new Date(m.timestamp).toLocaleTimeString());
  }catch(e){console.error(e)}
}

refresh();
setInterval(refresh,3000);
</script>
</body>
</html>`
