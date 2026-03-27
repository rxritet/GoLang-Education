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

	"jobqueue/handler"
	"jobqueue/store"
	"jobqueue/worker"
)

// ---------- CLI-конфигурация ----------

// Config объединяет все настраиваемые параметры сервера.
type Config struct {
	Port       int
	Workers    int
	QueueSize  int
	JobTimeout int // секунды
}

// ParseFlags разбирает аргументы через отдельный FlagSet.
func ParseFlags(fs *flag.FlagSet, args []string) Config {
	var cfg Config

	fs.IntVar(&cfg.Port, "port", 8080, "HTTP server port")
	fs.IntVar(&cfg.Port, "p", 8080, "HTTP server port (shorthand)")

	fs.IntVar(&cfg.Workers, "workers", 3, "Number of worker goroutines")
	fs.IntVar(&cfg.Workers, "w", 3, "Number of workers (shorthand)")

	fs.IntVar(&cfg.QueueSize, "queue", 100, "Job queue buffer size")
	fs.IntVar(&cfg.QueueSize, "q", 100, "Queue buffer size (shorthand)")

	fs.IntVar(&cfg.JobTimeout, "timeout", 30, "Job execution timeout in seconds")
	fs.IntVar(&cfg.JobTimeout, "t", 30, "Job timeout (shorthand)")

	_ = fs.Parse(args)
	return cfg
}

// ---------- Интерактивный режим ----------

// promptInt читает строку из scanner и возвращает число, либо fallback при ошибке/пустом вводе.
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

	fmt.Fprintln(w, "=== Job Queue Server (interactive mode) ===")
	fmt.Fprintln(w)

	cfg := Config{
		Port:       promptInt(scanner, w, "HTTP port [8080]: ", 8080),
		Workers:    promptInt(scanner, w, "Number of workers [3]: ", 3),
		QueueSize:  promptInt(scanner, w, "Queue buffer size [100]: ", 100),
		JobTimeout: promptInt(scanner, w, "Job timeout in seconds [30]: ", 30),
	}

	fmt.Fprintln(w)
	return cfg
}

// ---------- main ----------

func main() {
	var cfg Config

	if len(os.Args) < 2 {
		cfg = RunInteractive(os.Stdin, os.Stdout)
	} else {
		cfg = ParseFlags(flag.CommandLine, os.Args[1:])
	}

	// Слой хранения.
	jobStore := store.New()

	// Слой бизнес-логики: Worker Pool.
	pool := worker.NewPool(jobStore, worker.Config{
		NumWorkers: cfg.Workers,
		QueueSize:  cfg.QueueSize,
		JobTimeout: time.Duration(cfg.JobTimeout) * time.Second,
	})

	// Слой хендлеров.
	h := handler.New(jobStore, pool)
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

	// Graceful shutdown: перехватываем SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("[server] listening on http://localhost%s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[server] listen error: %v", err)
		}
	}()

	<-quit // блокируемся до сигнала
	log.Println("[server] shutting down…")

	pool.Stop() // ждём завершения воркеров
	log.Println("[server] stopped")
}
