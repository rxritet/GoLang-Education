# JobQueue

HTTP-сервер на Go, реализующий паттерн **Job Queue + Worker Pool** с полным контролем конкурентности через горутины, каналы, `sync.RWMutex` и `context`.

## Структура проекта

```
JobQueue/
├── go.mod
├── main.go                    # Точка входа, CLI-флаги, интерактивный режим
├── README.md
├── store/
│   ├── store.go               # In-memory хранилище (sync.RWMutex)
│   └── store_test.go          # Тесты хранилища
├── worker/
│   ├── pool.go                # Worker Pool + буферизованный канал-семафор
│   └── pool_test.go           # Тесты пула (включая timeout)
└── handler/
    ├── handler.go             # HTTP-хендлеры (POST/GET /jobs)
    └── handler_test.go        # Тесты хендлеров (httptest)
```

## Архитектура

```
  HTTP client                          Worker Pool
  ──────────                           ───────────
  POST /jobs ──▶ handler ──▶ store.Save()
                    │
                    ▼
              pool.Submit(id)
                    │
                    ▼
           ┌──────────────┐
           │ buffered chan │  ← QueueSize (не блокирует HTTP)
           └──────┬───────┘
                  │ fan-out
          ┌───────┼───────┐
          ▼       ▼       ▼
       worker1 worker2 worker3
          │       │       │
          └───────┼───────┘
                  ▼
          store.UpdateStatus()   ← потокобезопасно (Lock)

  GET /jobs/{id} ──▶ handler ──▶ store.Get() ← RLock (параллельное чтение)
```

### Примитивы синхронизации

| Примитив | Где | Зачем |
|---|---|---|
| `sync.RWMutex` | `store/` | Потокобезопасный доступ к map задач |
| `chan string` (buffered) | `worker/` | Очередь задач; размер буфера = QueueSize |
| `sync.WaitGroup` | `worker/` | Ожидание завершения воркеров при shutdown |
| `context.WithTimeout` | `worker/` | Дедлайн на выполнение одной задачи |

## API

### `POST /jobs`

Создаёт задачу и ставит в очередь.

```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{"task":"send_email"}'
```

Ответ (`202 Accepted`):
```json
{"id": "550e8400-e29b-41d4-a716-446655440000", "status": "queued"}
```

### `GET /jobs/{id}`

Возвращает текущее состояние задачи.

```bash
curl http://localhost:8080/jobs/550e8400-e29b-41d4-a716-446655440000
```

Ответ:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "task": "send_email",
  "status": "completed",
  "created_at": "2026-02-27T23:00:00Z",
  "updated_at": "2026-02-27T23:00:03Z"
}
```

### `GET /jobs`

Список всех задач.

```bash
curl http://localhost:8080/jobs
```

### Статусы задач

| Статус | Описание |
|--------|----------|
| `queued` | В очереди, ждёт воркера |
| `running` | Воркер выполняет задачу |
| `completed` | Успешно завершена |
| `failed` | Завершилась с ошибкой |
| `cancelled` | Отменена по таймауту контекста |

## Флаги командной строки

| Флаг | Короткий | По умолчанию | Описание |
|------|----------|--------------|----------|
| `--port` | `-p` | `8080` | Порт HTTP-сервера |
| `--workers` | `-w` | `3` | Число воркеров |
| `--queue` | `-q` | `100` | Размер буфера очереди |
| `--timeout` | `-t` | `30` | Таймаут задачи (секунды) |

## Примеры запуска

### Флаги

```bash
go run main.go -p 8080 -w 5 -q 200 -t 60
```

### Интерактивный режим

```
$ go run main.go
=== Job Queue Server (interactive mode) ===

HTTP port [8080]: 9000
Number of workers [3]: 5
Queue buffer size [100]:
Job timeout in seconds [30]: 15
```

### Полный сценарий

```bash
# Терминал 1 — запуск сервера
go run main.go -w 3

# Терминал 2 — отправка задач
curl -X POST localhost:8080/jobs -d '{"task":"send_email"}'
curl -X POST localhost:8080/jobs -d '{"task":"resize_image"}'
curl -X POST localhost:8080/jobs -d '{"task":"generate_report"}'

# Проверка статуса
curl localhost:8080/jobs/<id>

# Список всех
curl localhost:8080/jobs
```

## Сборка

```bash
cd JobQueue
go build -o jobqueue.exe main.go   # Windows
go build -o jobqueue main.go       # Linux / macOS
```

## Запуск тестов

```bash
cd JobQueue

# Все тесты
go test ./...

# С подробным выводом
go test -v ./store/ ./worker/ ./handler/

# С покрытием
go test -cover ./...
```
