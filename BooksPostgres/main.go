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

	"github.com/jackc/pgx/v5/pgxpool"

	"bookspostgres/handler"
	"bookspostgres/repository"
)

// ---------- CLI-конфигурация ----------

// Config объединяет настраиваемые параметры сервера.
type Config struct {
	Port int
	DSN  string
}

// ParseFlags разбирает аргументы командной строки.
func ParseFlags(fs *flag.FlagSet, args []string) Config {
	var cfg Config

	defaultDSN := getenv("DATABASE_URL", "postgres://books:books@localhost:5432/booksdb?sslmode=disable")

	fs.IntVar(&cfg.Port, "port", 8080, "HTTP server port")
	fs.IntVar(&cfg.Port, "p", 8080, "HTTP server port (shorthand)")
	fs.StringVar(&cfg.DSN, "dsn", defaultDSN, "PostgreSQL connection string")

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
	defaultDSN := getenv("DATABASE_URL", "postgres://books:books@localhost:5432/booksdb?sslmode=disable")

	fmt.Fprintln(w, "=== Books API v2 + PostgreSQL (interactive mode) ===")
	fmt.Fprintln(w)

	return Config{
		Port: promptInt(scanner, w, "HTTP port [8080]: ", 8080),
		DSN:  promptStr(scanner, w, fmt.Sprintf("PostgreSQL DSN [%s]: ", defaultDSN), defaultDSN),
	}
}

// ---------- Вспомогательная функция ----------

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

	// Контекст приложения.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Подключаемся к PostgreSQL через pgxpool.
	pool, err := pgxpool.New(ctx, cfg.DSN)
	if err != nil {
		log.Fatalf("[db] failed to create pool: %v", err)
	}
	defer pool.Close()

	// Проверяем соединение.
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("[db] ping failed: %v\nHint: run `docker-compose up -d` to start PostgreSQL", err)
	}
	log.Println("[db] connected to PostgreSQL")

	// Репозиторий и HTTP-обработчики.
	repo := repository.New(pool)
	h := handler.New(repo)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

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
