// Package scraper реализует конкурентный сбор HTML-заголовков (<title>) по списку URL.
//
// Ключевые примитивы синхронизации:
//   - sync.WaitGroup  — счётчик активных горутин; main-горутина блокируется
//     на wg.Wait() до тех пор, пока каждый воркер не вызовет wg.Done().
//   - Буферизованный канал sem (chan struct{}) — действует как считающий семафор.
//     Размер буфера = макс. число одновременных HTTP-запросов.
//     Перед запросом горутина пишет в sem (захватывает «слот»), после — читает (освобождает).
//   - Канал results (chan Result) — каждый воркер отправляет результат, а
//     горутина-агрегатор читает из него и собирает итоговый срез.
package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// ---------- Публичные типы ----------

// Result описывает результат обработки одного URL.
type Result struct {
	URL   string // запрошенный адрес
	Title string // содержимое <title>, если удалось извлечь
	Err   error  // ошибка запроса или парсинга (nil при успехе)
}

// Config задаёт параметры скрапера.
type Config struct {
	MaxWorkers int           // макс. число одновременных HTTP-запросов (семафор)
	Timeout    time.Duration // таймаут одного HTTP-запроса
}

// DefaultConfig возвращает конфигурацию по умолчанию: 5 воркеров, 10 секунд таймаут.
func DefaultConfig() Config {
	return Config{
		MaxWorkers: 5,
		Timeout:    10 * time.Second,
	}
}

// ---------- Публичный API ----------

// Run запускает конкурентный сбор заголовков для переданных URL.
// Возвращает срез Result (по одному на каждый URL) после обработки всех адресов.
//
// Порядок результатов НЕ гарантирован — он зависит от скорости ответов серверов.
func Run(urls []string, cfg Config) []Result {
	if cfg.MaxWorkers < 1 {
		cfg.MaxWorkers = 1
	}

	// ----- Кастомный HTTP-клиент с жёстким таймаутом -----
	// Таймаут распространяется на DNS, TLS-рукопожатие, передачу тела — весь цикл.
	client := &http.Client{
		Timeout: cfg.Timeout,
	}

	// ----- Семафор: буферизованный канал -----
	// Ёмкость буфера = MaxWorkers. Горутина блокируется на записи,
	// если все слоты заняты, и продолжает только когда один из слотов освободится.
	sem := make(chan struct{}, cfg.MaxWorkers)

	// ----- Канал результатов -----
	// Небуферизованный (или маленький буфер) — воркеры пишут, агрегатор читает.
	results := make(chan Result, len(urls))

	// ----- WaitGroup -----
	// Счётчик увеличивается на 1 перед запуском каждой горутины
	// и уменьшается внутри горутины через defer wg.Done().
	var wg sync.WaitGroup

	// Запускаем по одной горутине на URL.
	for _, u := range urls {
		wg.Add(1) // +1 ДО запуска горутины — гарантирует, что Wait не завершится раньше времени.

		go func(rawURL string) {
			defer wg.Done() // при любом исходе уменьшаем счётчик

			// Захватываем слот семафора (блокирует, если все MaxWorkers слотов заняты).
			sem <- struct{}{}
			// Освобождаем слот после завершения работы.
			defer func() { <-sem }()

			title, err := fetchTitle(client, rawURL)
			results <- Result{URL: rawURL, Title: title, Err: err}
		}(u)
	}

	// ----- Горутина-«закрыватель» -----
	// Ждёт завершения всех воркеров, затем закрывает канал results,
	// чтобы агрегатор (range) корректно завершился.
	go func() {
		wg.Wait()
		close(results)
	}()

	// ----- Агрегация результатов -----
	// Читаем из канала до его закрытия. Это происходит в текущей горутине,
	// поэтому функция Run сама блокируется, пока все результаты не будут собраны.
	var collected []Result
	for r := range results {
		collected = append(collected, r)
	}

	return collected
}

// ---------- Внутренние функции ----------

// fetchTitle выполняет GET-запрос и извлекает содержимое <title> из HTML.
func fetchTitle(client *http.Client, rawURL string) (string, error) {
	// Нормализуем URL: если нет схемы — подставляем https://.
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("bad URL: %w", err)
	}
	req.Header.Set("User-Agent", "GoWebScraper/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Ограничиваем чтение 1 МБ — защищает от огромных страниц при парсинге.
	limited := io.LimitReader(resp.Body, 1<<20)
	return extractTitle(limited)
}

// extractTitle парсит HTML-поток и возвращает текст первого элемента <title>.
// Используется потоковый (SAX-подобный) парсер golang.org/x/net/html —
// он не загружает всё дерево в память.
func extractTitle(r io.Reader) (string, error) {
	tokenizer := html.NewTokenizer(r)

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			err := tokenizer.Err()
			if err == io.EOF {
				return "", fmt.Errorf("title not found")
			}
			return "", fmt.Errorf("parse error: %w", err)

		case html.StartTagToken:
			tn, _ := tokenizer.TagName()
			if string(tn) == "title" {
				// Следующий токен — текстовое содержимое <title>.
				if tokenizer.Next() == html.TextToken {
					return strings.TrimSpace(string(tokenizer.Text())), nil
				}
				return "", nil // пустой <title></title>
			}
		}
	}
}
