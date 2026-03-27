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

	"sysmonitor/collector"
	"sysmonitor/handler"
)

// ---------- CLI-конфигурация ----------

// Config объединяет настраиваемые параметры сервера.
type Config struct {
	Port     int
	Interval int // интервал сбора метрик (секунды)
}

// ParseFlags разбирает аргументы через отдельный FlagSet.
func ParseFlags(fs *flag.FlagSet, args []string) Config {
	var cfg Config

	fs.IntVar(&cfg.Port, "port", 8080, "HTTP server port")
	fs.IntVar(&cfg.Port, "p", 8080, "HTTP server port (shorthand)")

	fs.IntVar(&cfg.Interval, "interval", 5, "Metrics collection interval in seconds")
	fs.IntVar(&cfg.Interval, "i", 5, "Collection interval (shorthand)")

	_ = fs.Parse(args)
	return cfg
}

// ---------- Интерактивный режим ----------

// promptInt читает число из scanner; возвращает fallback при пустом/невалидном вводе.
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

	fmt.Fprintln(w, "=== System Monitor (interactive mode) ===")
	fmt.Fprintln(w)

	cfg := Config{
		Port:     promptInt(scanner, w, "HTTP port [8080]: ", 8080),
		Interval: promptInt(scanner, w, "Collection interval in seconds [5]: ", 5),
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

	// --- Collector (фоновый сбор метрик) ---
	// Создаём контекст, который отменится при shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	coll := collector.New(time.Duration(cfg.Interval) * time.Second)

	// Запускаем фоновую горутину сбора метрик.
	// При cancel() тикер остановится и горутина завершится.
	go coll.Run(ctx)

	// --- HTTP-сервер ---
	h := handler.New(coll)
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

	// --- Graceful Shutdown ---
	// Перехватываем SIGINT (Ctrl+C) и SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем HTTP-сервер в фоне.
	go func() {
		log.Printf("[server] listening on http://localhost%s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[server] listen error: %v", err)
		}
	}()

	// Блокируемся до получения сигнала.
	sig := <-quit
	log.Printf("[server] received %v, shutting down…", sig)

	// 1. Останавливаем collector (cancel закрывает ctx.Done → тикер остановится).
	cancel()

	// 2. Даём HTTP-серверу 5 секунд на завершение текущих запросов.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[server] shutdown error: %v", err)
	}

	log.Println("[server] stopped")
}
