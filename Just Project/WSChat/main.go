package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"wschat/client"
	"wschat/hub"
)

// ---------- CLI-конфигурация ----------

// Config объединяет параметры запуска сервера.
type Config struct {
	Port int
}

// ParseFlags разбирает аргументы командной строки.
func ParseFlags(fs *flag.FlagSet, args []string) Config {
	var cfg Config
	fs.IntVar(&cfg.Port, "port", 8080, "HTTP/WS server port")
	fs.IntVar(&cfg.Port, "p", 8080, "HTTP/WS server port (shorthand)")
	_ = fs.Parse(args)
	return cfg
}

// ---------- Интерактивный режим ----------

func promptInt(scanner *bufio.Scanner, w io.Writer, prompt string, fallback int) int {
	fmt.Fprintf(w, "%s", prompt)
	if scanner.Scan() {
		if v, err := strconv.Atoi(strings.TrimSpace(scanner.Text())); err == nil && v > 0 {
			return v
		}
	}
	return fallback
}

// RunInteractive запрашивает порт через stdin.
func RunInteractive(r io.Reader, w io.Writer) Config {
	scanner := bufio.NewScanner(r)

	fmt.Fprintln(w, "=== WebSocket Chat (interactive mode) ===")
	fmt.Fprintln(w)

	return Config{
		Port: promptInt(scanner, w, "HTTP/WS port [8080]: ", 8080),
	}
}

// ---------- Handlers ----------

// serveHome отдаёт HTML/JS фронтенд.
func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(chatHTML))
}

// ---------- main ----------

func main() {
	var cfg Config

	if len(os.Args) < 2 {
		cfg = RunInteractive(os.Stdin, os.Stdout)
	} else {
		cfg = ParseFlags(flag.CommandLine, os.Args[1:])
	}

	// 1. Создаём хаб и запускаем в отдельной горутине.
	h := hub.New()
	go h.Run()

	// 2. Настраиваем HTTP-сервер.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", serveHome)
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		client.ServeWs(h, w, r)
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 3. Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("[server] WebSocket Chat on http://localhost%s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[server] listen error: %v", err)
		}
	}()

	<-quit
	log.Println("[server] shutting down…")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[server] shutdown error: %v", err)
	}
	log.Println("[server] stopped")
}

// ---------- Frontend ----------

const chatHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>WebSocket Chat</title>
<style>
  *, *::before, *::after { box-sizing: border-box; }
  body { margin: 0; font-family: system-ui, -apple-system, sans-serif; background: #0f172a; color: #e2e8f0; height: 100vh; display: flex; align-items: center; justify-content: center; overflow: hidden; }
  
  /* Modal Overlay */
  #overlay { position: fixed; inset: 0; background: rgba(15,23,42,.9); display: flex; align-items: center; justify-content: center; z-index: 50; }
  .modal { background: #1e293b; padding: 2rem; border-radius: 1rem; text-align: center; width: 100%; max-width: 360px; box-shadow: 0 20px 25px -5px rgba(0,0,0,.1); border: 1px solid #334155; }
  .modal h2 { margin-top: 0; margin-bottom: 1.5rem; font-size: 1.5rem; }
  .modal input { width: 100%; padding: .75rem 1rem; border-radius: .5rem; border: 1px solid #334155; background: #0f172a; color: #fff; font-size: 1rem; outline: none; transition: border-color .15s; margin-bottom: 1rem; }
  .modal input:focus { border-color: #6366f1; box-shadow: 0 0 0 3px rgba(99,102,241,.2); }
  .modal button { width: 100%; padding: .75rem; border: none; background: #6366f1; color: #fff; font-size: 1rem; font-weight: 600; border-radius: .5rem; cursor: pointer; transition: background .15s; }
  .modal button:hover { background: #4f46e5; }

  /* App Layout */
  #app { width: 100%; max-width: 800px; height: 100vh; display: flex; flex-direction: column; background: #0f172a; opacity: 0; transition: opacity .3s; }
  
  /* Header */
  .header { padding: 1rem 1.5rem; background: #1e293b; border-bottom: 1px solid #334155; display: flex; align-items: center; justify-content: space-between; flex-shrink: 0; }
  .header h1 { margin: 0; font-size: 1.25rem; }
  .status { display: flex; align-items: center; gap: .5rem; font-size: .85rem; color: #94a3b8; font-weight: 500; }
  .dot { width: .6rem; height: .6rem; border-radius: 50%; background: #ef4444; }
  .dot.online { background: #22c55e; }

  /* Chat Messages Area */
  #messages { flex: 1; overflow-y: auto; padding: 1.5rem; display: flex; flex-direction: column; gap: 1rem; scroll-behavior: smooth; }
  
  /* Message Bubble Base */
  .msg-row { display: flex; width: 100%; }
  .msg { padding: .7rem 1rem; border-radius: 1rem; max-width: 75%; position: relative; word-wrap: break-word; }
  .msg-author { font-size: .75rem; margin-bottom: .25rem; opacity: .7; font-weight: 600; display: block; }
  
  /* Theme: Others */
  .row-other { justify-content: flex-start; }
  .row-other .msg { background: #334155; color: #f8fafc; border-bottom-left-radius: .2rem; }
  
  /* Theme: Self */
  .row-self { justify-content: flex-end; }
  .row-self .msg { background: #6366f1; color: #fff; border-bottom-right-radius: .2rem; }
  .row-self .msg-author { display: none; } /* Hide 'Me' to save space */
  
  /* Theme: System */
  .row-system { justify-content: center; }
  .row-system .msg { text-align: center; background: transparent; color: #64748b; font-size: .85rem; font-style: italic; max-width: 100%; padding: .2rem 1rem; }
  .row-system .msg-author { display: none; }

  /* Input Area */
  #form { display: flex; gap: .75rem; padding: 1rem 1.5rem; background: #1e293b; border-top: 1px solid #334155; flex-shrink: 0; }
  #input { flex: 1; padding: .75rem 1rem; border-radius: 9999px; border: 1px solid #334155; background: #0f172a; color: #fff; outline: none; font-size: .95rem; }
  #input:focus { border-color: #6366f1; }
  #send { padding: .75rem 1.7rem; border: none; border-radius: 9999px; background: #6366f1; color: #fff; font-weight: 600; cursor: pointer; transition: background .15s; }
  #send:hover { background: #4f46e5; }
  
  /* Scrollbar Customization */
  ::-webkit-scrollbar { width: 8px; }
  ::-webkit-scrollbar-track { background: transparent; }
  ::-webkit-scrollbar-thumb { background: #334155; border-radius: 4px; }
  ::-webkit-scrollbar-thumb:hover { background: #475569; }
</style>
</head>
<body>

<div id="overlay">
  <div class="modal">
    <h2>Join Chat</h2>
    <form id="joinForm">
      <input type="text" id="usernameInput" placeholder="Choose a username..." autocomplete="off" autofocus required>
      <button type="submit">Enter Room</button>
    </form>
  </div>
</div>

<div id="app">
  <div class="header">
    <h1>💬 WSChat</h1>
    <div class="status">
      <span class="dot" id="dot"></span>
      <span id="statusTxt">Disconnected</span>
      <span style="border-left:1px solid #334155;margin-left:5px;padding-left:10px" id="myUsernameDisplay"></span>
    </div>
  </div>
  
  <div id="messages"></div>
  
  <form id="form">
    <input type="text" id="input" autocomplete="off" placeholder="Message..." autofocus>
    <button id="send">Send</button>
  </form>
</div>

<script>
let conn;
let myUsername = '';
const msgBox = document.getElementById("messages");
const dot = document.getElementById("dot");
const st  = document.getElementById("statusTxt");

function setStatus(online) {
  dot.className = online ? "dot online" : "dot";
  st.textContent = online ? "Online" : "Disconnected";
}

function esc(s) { return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }

function appendMsg(item) {
  const doScroll = msgBox.scrollTop > msgBox.scrollHeight - msgBox.clientHeight - 80;
  msgBox.appendChild(item);
  if (doScroll) { msgBox.scrollTop = msgBox.scrollHeight; }
}

function showSystemMsg(text) {
  const row = document.createElement("div"); row.className = "msg-row row-system";
  const m = document.createElement("div"); m.className = "msg"; m.innerHTML = esc(text);
  row.appendChild(m); appendMsg(row);
}

function showMessage(user, text) {
  if (user === "System") { showSystemMsg(text); return; }
  
  const isMe = user === myUsername;
  const row = document.createElement("div");
  row.className = "msg-row " + (isMe ? "row-self" : "row-other");
  
  const m = document.createElement("div"); m.className = "msg";
  
  const authorSpan = document.createElement("span");
  authorSpan.className = "msg-author"; authorSpan.textContent = user;
  
  const textSpan = document.createElement("span");
  textSpan.innerHTML = esc(text);
  
  m.appendChild(authorSpan); m.appendChild(textSpan);
  row.appendChild(m); appendMsg(row);
}

function initWebSocket() {
  if (!window["WebSocket"]) { showSystemMsg("Your browser does not support WebSockets."); return; }
  
  const wsURL = (document.location.protocol === "https:" ? "wss://" : "ws://") + document.location.host + "/ws?user=" + encodeURIComponent(myUsername);
  conn = new WebSocket(wsURL);
  
  conn.onclose   = () => { setStatus(false); showSystemMsg("Connection closed. Refresh to reconnect."); };
  conn.onopen    = () => { setStatus(true); };
  conn.onmessage = (e) => {
    try {
      const data = JSON.parse(e.data);
      showMessage(data.user, data.text);
    } catch { showSystemMsg("Invalid JSON received"); }
  };
}

document.getElementById("joinForm").onsubmit = function() {
  const inp = document.getElementById("usernameInput");
  myUsername = inp.value.trim();
  if (!myUsername) return false;
  
  document.getElementById("myUsernameDisplay").textContent = myUsername;
  document.getElementById("overlay").style.display = "none";
  document.getElementById("app").style.opacity = "1";
  document.getElementById("input").focus();
  
  initWebSocket();
  return false;
};

document.getElementById("form").onsubmit = function() {
  if (!conn || conn.readyState !== WebSocket.OPEN) return false;
  const input = document.getElementById("input");
  if (!input.value) return false;
  
  conn.send(input.value);
  input.value = "";
  return false;
};
</script>
</body>
</html>`
