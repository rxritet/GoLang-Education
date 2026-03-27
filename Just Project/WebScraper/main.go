package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"webscraper/scraper"
)

// ---------- CLI-конфигурация ----------

// Config содержит параметры, полученные из флагов или интерактивного ввода.
type Config struct {
	FilePath   string        // путь к файлу с URL
	MaxWorkers int           // максимум одновременных запросов
	Timeout    time.Duration // таймаут HTTP-запроса
}

// ParseFlags разбирает аргументы командной строки через отдельный FlagSet
// (удобно для тестирования — не затрагивает глобальный flag.CommandLine).
func ParseFlags(fs *flag.FlagSet, args []string) Config {
	var cfg Config
	fs.StringVar(&cfg.FilePath, "file", "", "Path to text file with URLs (one per line)")
	fs.StringVar(&cfg.FilePath, "f", "", "Path to text file with URLs (shorthand)")
	fs.IntVar(&cfg.MaxWorkers, "workers", 5, "Max concurrent HTTP requests")
	fs.IntVar(&cfg.MaxWorkers, "w", 5, "Max concurrent requests (shorthand)")

	var timeoutSec int
	fs.IntVar(&timeoutSec, "timeout", 10, "HTTP request timeout in seconds")
	fs.IntVar(&timeoutSec, "t", 10, "HTTP timeout in seconds (shorthand)")

	_ = fs.Parse(args)

	cfg.Timeout = time.Duration(timeoutSec) * time.Second
	return cfg
}

// ---------- Интерактивный режим ----------

// RunInteractive запрашивает параметры через stdin.
func RunInteractive(r io.Reader, w io.Writer) Config {
	scanner := bufio.NewScanner(r)
	cfg := Config{MaxWorkers: 5, Timeout: 10 * time.Second}

	fmt.Fprintln(w, "=== Web Scraper (interactive mode) ===")
	fmt.Fprintln(w)

	// Файл с URL
	fmt.Fprintf(w, "Path to URL file: ")
	if scanner.Scan() {
		cfg.FilePath = strings.TrimSpace(scanner.Text())
	}

	// Workers
	fmt.Fprintf(w, "Max concurrent workers [5]: ")
	if scanner.Scan() {
		if v, err := strconv.Atoi(strings.TrimSpace(scanner.Text())); err == nil && v > 0 {
			cfg.MaxWorkers = v
		}
	}

	// Timeout
	fmt.Fprintf(w, "HTTP timeout in seconds [10]: ")
	if scanner.Scan() {
		if v, err := strconv.Atoi(strings.TrimSpace(scanner.Text())); err == nil && v > 0 {
			cfg.Timeout = time.Duration(v) * time.Second
		}
	}

	fmt.Fprintln(w)
	return cfg
}

// ---------- Загрузка URL из файла ----------

// LoadURLs читает текстовый файл и возвращает непустые строки (по одной URL на строку).
func LoadURLs(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer f.Close()

	return ReadURLs(f)
}

// ReadURLs читает URL из произвольного io.Reader (удобно для тестов).
func ReadURLs(r io.Reader) ([]string, error) {
	var urls []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") { // пропускаем пустые и комментарии
			urls = append(urls, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("file contains no URLs")
	}
	return urls, nil
}

// ---------- Вывод результатов ----------

// PrintResults форматирует и печатает результаты скрапинга.
func PrintResults(w io.Writer, results []scraper.Result) {
	fmt.Fprintln(w, strings.Repeat("─", 60))
	fmt.Fprintf(w, "  %-40s  %s\n", "URL", "TITLE / ERROR")
	fmt.Fprintln(w, strings.Repeat("─", 60))

	var ok, fail int
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(w, "  %-40s  [ERROR] %v\n", truncate(r.URL, 40), r.Err)
			fail++
		} else {
			fmt.Fprintf(w, "  %-40s  %s\n", truncate(r.URL, 40), r.Title)
			ok++
		}
	}

	fmt.Fprintln(w, strings.Repeat("─", 60))
	fmt.Fprintf(w, "  Done: %d success, %d failed, %d total\n", ok, fail, ok+fail)
}

// truncate обрезает строку до maxLen символов, добавляя "…" при обрезке.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

// ---------- main ----------

func main() {
	var cfg Config

	// Если аргументов нет — интерактивный режим, иначе — флаги.
	if len(os.Args) < 2 {
		cfg = RunInteractive(os.Stdin, os.Stdout)
	} else {
		cfg = ParseFlags(flag.CommandLine, os.Args[1:])
	}

	if cfg.FilePath == "" {
		fmt.Fprintln(os.Stderr, "error: URL file path is required (--file / -f)")
		os.Exit(1)
	}

	urls, err := LoadURLs(cfg.FilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Scraping %d URLs (workers=%d, timeout=%s)…\n\n",
		len(urls), cfg.MaxWorkers, cfg.Timeout)

	results := scraper.Run(urls, scraper.Config{
		MaxWorkers: cfg.MaxWorkers,
		Timeout:    cfg.Timeout,
	})

	PrintResults(os.Stdout, results)
}
