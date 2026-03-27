# WebScraper

Конкурентный Web Scraper на Go — собирает HTML-заголовки (`<title>`) из списка URL, используя горутины, семафор и каналы.

## Структура проекта

```
WebScraper/
├── go.mod
├── main.go              # CLI-точка входа, интерактивный режим
├── urls.txt             # Пример файла с URL
├── README.md
└── scraper/
    ├── scraper.go       # Ядро: горутины, семафор, каналы, парсинг
    └── scraper_test.go  # Unit-тесты (httptest + table-driven)
```

## Как это работает

```
┌──────────┐     ┌───────┐     ┌──────────────┐     ┌────────────┐
│ URL file │────▶│ main  │────▶│  scraper.Run │────▶│  Результат │
└──────────┘     └───────┘     └──────┬───────┘     └────────────┘
                                      │
                        ┌─────────────┼─────────────┐
                        ▼             ▼             ▼
                   goroutine 1   goroutine 2   goroutine N
                        │             │             │
                   ┌────┴────┐   ┌────┴────┐   ┌────┴────┐
                   │ sem <── │   │ sem <── │   │ sem <── │  ← семафор
                   │ fetch   │   │ fetch   │   │ fetch   │
                   │ sem ──> │   │ sem ──> │   │ sem ──> │
                   └────┬────┘   └────┬────┘   └────┬────┘
                        │             │             │
                        └──────┐      │      ┌──────┘
                               ▼      ▼      ▼
                          ┌────────────────────┐
                          │  results channel   │
                          └────────┬───────────┘
                                   ▼
                            aggregator goroutine
```

**Примитивы синхронизации:**

| Примитив | Роль |
|----------|------|
| `sync.WaitGroup` | Ожидание завершения всех горутин-воркеров |
| `chan struct{}` (buffered) | Семафор — ограничивает число одновременных HTTP-запросов |
| `chan Result` | Передача результатов от воркеров к агрегатору |

## Флаги командной строки

| Флаг | Короткий | Тип | По умолчанию | Описание |
|------|----------|-----|--------------|----------|
| `--file` | `-f` | `string` | — | Путь к файлу с URL (обязательный) |
| `--workers` | `-w` | `int` | `5` | Макс. одновременных запросов |
| `--timeout` | `-t` | `int` | `10` | Таймаут HTTP-запроса (секунды) |

## Примеры использования

### Режим флагов

```bash
# Базовый запуск
go run main.go -f urls.txt

# 10 воркеров, таймаут 5 секунд
go run main.go --file urls.txt --workers 10 --timeout 5

# Короткие флаги
go run main.go -f urls.txt -w 8 -t 3
```

### Интерактивный режим

Запуск без аргументов переключает в диалоговый режим:

```
$ go run main.go
=== Web Scraper (interactive mode) ===

Path to URL file: urls.txt
Max concurrent workers [5]: 3
HTTP timeout in seconds [10]: 5

Scraping 5 URLs (workers=3, timeout=5s)…

────────────────────────────────────────────────────────────
  URL                                       TITLE / ERROR
────────────────────────────────────────────────────────────
  https://go.dev                            Go Programming Language
  https://github.com                        GitHub
  https://en.wikipedia.org                  Wikipedia
  https://www.rust-lang.org                 Rust Programming Language
  https://news.ycombinator.com              Hacker News
────────────────────────────────────────────────────────────
  Done: 5 success, 0 failed, 5 total
```

### Формат файла URL

```text
# Комментарии начинаются с #
https://go.dev
https://github.com
example.com        # схема https:// подставится автоматически
```

## Сборка

```bash
cd WebScraper
go build -o webscraper.exe main.go   # Windows
go build -o webscraper main.go       # Linux / macOS
```

## Запуск тестов

```bash
cd WebScraper

# Все тесты
go test ./...

# Подробный вывод
go test -v ./scraper/

# С покрытием
go test -cover ./scraper/
```
