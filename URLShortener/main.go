package main

import (
	"bufio"
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

	"urlshortener/handler"
	"urlshortener/middleware"
	"urlshortener/store"
)

// ---------- CLI-конфигурация ----------

// Config объединяет настраиваемые параметры сервера.
type Config struct {
	Port      int
	JWTSecret string
	TokenTTL  int // минуты
}

// ParseFlags разбирает аргументы командной строки.
func ParseFlags(fs *flag.FlagSet, args []string) Config {
	var cfg Config

	defaultSecret := getenv("JWT_SECRET", "change-me-in-production")

	fs.IntVar(&cfg.Port, "port", 8080, "HTTP server port")
	fs.IntVar(&cfg.Port, "p", 8080, "HTTP server port (shorthand)")
	fs.StringVar(&cfg.JWTSecret, "secret", defaultSecret, "JWT signing secret")
	fs.IntVar(&cfg.TokenTTL, "ttl", 60, "JWT token TTL in minutes")

	_ = fs.Parse(args)
	return cfg
}

// ---------- Интерактивный режим ----------

func promptStr(scanner *bufio.Scanner, w io.Writer, prompt, fallback string) string {
	fmt.Fprintf(w, "%s", prompt)
	if scanner.Scan() {
		if v := strings.TrimSpace(scanner.Text()); v != "" {
			return v
		}
	}
	return fallback
}

func promptInt(scanner *bufio.Scanner, w io.Writer, prompt string, fallback int) int {
	fmt.Fprintf(w, "%s", prompt)
	if scanner.Scan() {
		if v, err := strconv.Atoi(strings.TrimSpace(scanner.Text())); err == nil && v > 0 {
			return v
		}
	}
	return fallback
}

// RunInteractive запрашивает параметры через stdin.
func RunInteractive(r io.Reader, w io.Writer) Config {
	scanner := bufio.NewScanner(r)
	defaultSecret := getenv("JWT_SECRET", "change-me-in-production")

	fmt.Fprintln(w, "=== URL Shortener + JWT Auth (interactive mode) ===")
	fmt.Fprintln(w)

	return Config{
		Port:      promptInt(scanner, w, "HTTP port [8080]: ", 8080),
		JWTSecret: promptStr(scanner, w, fmt.Sprintf("JWT secret [%s]: ", defaultSecret), defaultSecret),
		TokenTTL:  promptInt(scanner, w, "Token TTL in minutes [60]: ", 60),
	}
}

// ---------- Вспомогательная ----------

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ---------- main ----------

func main() {
	var cfg Config

	if len(os.Args) < 2 {
		cfg = RunInteractive(os.Stdin, os.Stdout)
	} else {
		cfg = ParseFlags(flag.CommandLine, os.Args[1:])
	}

	ttl := time.Duration(cfg.TokenTTL) * time.Minute

	// Хранилище (in-memory, потокобезопасное).
	s := store.New()

	// Обработчики.
	authHandler := handler.NewAuth(s, cfg.JWTSecret, ttl)
	linksHandler := handler.NewLinks(s)

	mux := http.NewServeMux()

	// --- Публичные маршруты ---
	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)

	// --- Защищённые маршруты (требуют JWT) ---
	mux.HandleFunc("POST /shorten", middleware.Auth(cfg.JWTSecret, linksHandler.Shorten))
	mux.HandleFunc("GET /links", middleware.Auth(cfg.JWTSecret, linksHandler.ListLinks))

	// --- Catchall: редирект по коду или веб-интерфейс ---
	mux.HandleFunc("/", linksHandler.Redirect)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("[server] listening on http://localhost%s", addr)
		log.Printf("[server] JWT TTL: %v", ttl)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[server] listen error: %v", err)
		}
	}()

	<-quit
	log.Println("[server] shutting down…")
	log.Println("[server] stopped")
}
