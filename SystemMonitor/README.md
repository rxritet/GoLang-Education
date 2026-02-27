# System Monitor API

HTTP-сервис на Go, который непрерывно собирает метрики приложения и системы в фоновом режиме и отдаёт их по HTTP в формате JSON, а также отображает на веб-дашборде.

## Архитектура

```
main.go                   CLI + Graceful Shutdown
  ├── collector/           Фоновый сбор метрик (Ticker + RWMutex)
  │   └── collector.go     Collector, Metrics, Run(ctx), Snapshot()
  └── handler/             HTTP-слой
      └── handler.go       GET /  GET /metrics  GET /health
```

### Ключевые паттерны

| Паттерн | Где используется |
|---------|-----------------|
| `time.Ticker` | `collector.Run()` — периодический сбор метрик |
| `sync.RWMutex` | `Collector` — множество читателей, один писатель |
| `runtime.ReadMemStats` | `collector.collect()` — память, GC, горутины |
| `context.WithCancel` | `main.go` — остановка collector при shutdown |
| Graceful Shutdown | `os.Signal` + `srv.Shutdown` с таймаутом 5 с |
| Interactive Mode | Без аргументов → интерактивный ввод параметров |

### Потокобезопасность

```
Фоновая горутина (Ticker)       HTTP-хендлер GET /metrics
────────────────────────        ──────────────────────────
     mu.Lock()                       mu.RLock()
     обновляет snapshot              читает snapshot (копию)
     mu.Unlock()                     mu.RUnlock()
```

`RWMutex` позволяет множеству HTTP-читателей работать параллельно, блокируя только на время короткой записи (~µs).

## API

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/` | HTML-дашборд с автообновлением (3 с) |
| GET | `/metrics` | JSON-снимок метрик |
| GET | `/health` | `{"status": "ok"}` |

### Пример ответа `/metrics`

```json
{
  "alloc_bytes": 1234567,
  "total_alloc_bytes": 9876543,
  "sys_bytes": 12345678,
  "heap_alloc_bytes": 1234567,
  "heap_sys_bytes": 8765432,
  "heap_objects": 4567,
  "num_gc": 3,
  "gc_pause_ns": 123456,
  "gc_cpu_percent": 0.0012,
  "num_goroutines": 5,
  "go_version": "go1.25.0",
  "goos": "windows",
  "goarch": "amd64",
  "num_cpu": 8,
  "uptime": "2m35s",
  "timestamp": "2025-01-15T12:00:00Z"
}
```

## Запуск

### Интерактивный режим (без аргументов)

```bash
go run .
# === System Monitor (interactive mode) ===
# HTTP port [8080]: 9090
# Collection interval in seconds [5]: 3
```

### CLI-режим

```bash
# Значения по умолчанию (порт 8080, интервал 5 с)
go run .

# Указать порт и интервал
go run . --port 9090 --interval 3

# Короткие флаги
go run . -p 9090 -i 3
```

### Флаги

| Флаг | Короткий | По умолчанию | Описание |
|------|----------|-------------|----------|
| `--port` | `-p` | 8080 | Порт HTTP-сервера |
| `--interval` | `-i` | 5 | Интервал сбора метрик (секунды) |

## Тестирование

```bash
go test -v ./...
```

9 тестов: 6 в `collector/`, 3 в `handler/`.

## Graceful Shutdown

При получении `SIGINT` (Ctrl+C) или `SIGTERM`:

1. Отменяется контекст → collector останавливает `Ticker` и завершает горутину
2. HTTP-сервер вызывает `Shutdown` с таймаутом 5 секунд для завершения текущих запросов
3. Программа завершается

## Структура проекта

```
SystemMonitor/
├── go.mod
├── main.go                 CLI, interactive mode, graceful shutdown
├── README.md
├── collector/
│   ├── collector.go        Collector + Metrics
│   └── collector_test.go   6 тестов
└── handler/
    ├── handler.go          HTTP-хендлеры + HTML-дашборд
    └── handler_test.go     3 теста
```
