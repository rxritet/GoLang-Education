# 🐹 Go Learning Path — 10 Projects

> Мой путь изучения Go через практику. Каждый проект добавляет новые концепции языка поверх предыдущего — от базового REST API до worker pool и системного мониторинга.
>
> Полученные здесь навыки применены в production-проектах — **[Specto](https://github.com/rxritet/Specto)** и **[HabitDuel](https://github.com/rxritet/HabitDuel)**.

---

## 🗺️ Roadmap

| # | Проект | Концепции | Статус |
|---|--------|-----------|--------|
| 01 | [BookManager](#01--bookmanager--rest-api) | `net/http`, CRUD, handlers/models, in-memory + `sync.RWMutex`, CORS | ✅ Готово |
| 02 | [TodoApp](#02--todoapp--cli-repl) | `bufio.Scanner`, REPL-лоп, JSON-персистентность, файловый I/O | ✅ Готово |
| 03 | [WeatherApp](#03--weatherapp--external-api) | HTTP client, внешние API, Makefile, `cmd/internal` структура | ✅ Готово |
| 04 | [PasswordGenerator](#04--passwordgenerator--crypto) | `crypto/rand`, `strings.Builder`, unit-тесты | ✅ Готово |
| 05 | [WebScraper](#05--webscraper--goroutines) | горутины, каналы, `sync.WaitGroup`, HTML-парсинг | ✅ Готово |
| 06 | [JobQueue](#06--jobqueue--worker-pool) | worker pool, буф. каналы, `context` с таймаутом | ✅ Готово |
| 07 | [SystemMonitor](#07--systemmonitor--ticker--metrics) | `time.Ticker`, фоновые горутины, `sync.Mutex`, `runtime` | ✅ Готово |
| 08 | [Books API v2 + PostgreSQL](#08--books-api-v2--postgresql) | `database/sql`, `pgx`, SQL-миграции, транзакции | ⏳ Планируется |
| 09 | [URL Shortener + JWT](#09--url-shortener--jwt-auth) | middleware, JWT, `bcrypt`, цепочки | ⏳ Планируется |
| 10 | [WebSocket Chat](#10--websocket-chat) | `gorilla/websocket`, broadcast, real-time состояние | ⏳ Планируется |

---

## 💪 Применено в production

Навыки из этих проектов напрямую использованы в настоящих проектах:

| Проект | Что применено |
|--------|-------------------|
| **[Specto](https://github.com/rxritet/Specto)** | чистый `net/http` (как BookManager), двойная стратегия БД BoltDB/PostgreSQL (как Books v2), tx-in-context, Cobra CLI, Mage, OpenTelemetry |
| **[HabitDuel](https://github.com/rxritet/HabitDuel)** | WebSocket real-time связь с WebSocket Chat, JWT-аутентификация (как URL Shortener), cron и фоновые горутины (как SystemMonitor) |

---

## 01 — BookManager — REST API

**Первый проект.** Полноценный REST API для управления коллекцией книг без внешних зависимостей.

**Структура:** `handlers/` — HTTP-обработчики, `models/` — доменные структуры, `static/` — веб-интерфейс.

- Полный CRUD через `net/http`
- Потокобезопасное in-memory хранилище (`sync.RWMutex`)
- Веб-интерфейс на чистом HTML/JS
- CORS для локальной разработки

```bash
cd BookManager && go run .
# http://localhost:8080
```

---

## 02 — TodoApp — CLI REPL

Менеджер задач с REPL-интерфейсом. Задачи сохраняются в JSON-файл и живут между перезапусками.

**Файлы:** `main.go` — точка входа, `repl.go` — интерактивный цикл, `todo.go` — модель, `storage.go` — персистентность.

- `bufio.Scanner` для чтения stdin
- `encoding/json` + `os` для персистентности
- Чистое разделение ответственностей (модель / I/O / UI)

```bash
cd TodoApp && go run .
# add <текст> | list | done <id> | delete <id> | quit
```

---

## 03 — WeatherApp — External API

Консольное приложение для получения погоды через OpenWeatherMap API. Структура по `cmd/internal` — стандартная для реальных Go-проектов.

- Внешний HTTP-клиент с таймаутом
- Парсинг вложенного JSON
- Makefile: `make run CITY=Almaty`, `make build`
- Грамотная обработка ошибок сети

```bash
cd WeatherApp
make run CITY=Almaty
# или: go run ./cmd/weather --city Almaty
```

---

## 04 — PasswordGenerator — Crypto

CLI-утилита для генерации криптографически стойких паролей. Бизнес-логика вынесена в пакет `generator/`.

- `crypto/rand` вместо `math/rand` — безопасная генерация
- `strings.Builder` для эффективной сборки строки
- Флаги: `--length`, `--symbols`, `--numbers`, `--count`
- Unit-тесты: `go test ./...`

```bash
cd PasswordGenerator
go run . --length 16 --symbols --count 3
go test ./...
```

---

## 05 — WebScraper — Goroutines

Параллельный скрапер: принимает список URL из файла и одновременно собирает заголовки всех страниц.

**Структура:** `scraper/` — парсер и клиент, `main.go` — оркестрация, `urls.txt` — входной файл.

- Каждый URL обрабатывается в отдельной горутине
- `sync.WaitGroup` для синхронизации
- Результаты собираются через канал (`chan Result`)
- HTML-парсинг: `golang.org/x/net/html`

```bash
cd WebScraper && go run . --file urls.txt
```

---

## 06 — JobQueue — Worker Pool

HTTP-сервер принимает задачи и выполняет их в фоне через пул воркеров.

**Структура:** `handler/` — HTTP, `worker/` — пул воркеров, `store/` — хранилище задач.

- Worker pool pattern: N воркеров, буферизованный канал задач
- `context.WithTimeout` для отмены зависших задач
- Статусы: `pending` → `running` → `done/failed`
- REST эндпоинты: `POST /jobs`, `GET /jobs`, `GET /jobs/:id`

```bash
cd JobQueue && go run .
curl -X POST http://localhost:8080/jobs -d '{"type":"email","payload":"test"}'
curl http://localhost:8080/jobs
```

---

## 07 — SystemMonitor — Ticker & Metrics

Мониторинг системных ресурсов: каждые N секунд собираются метрики и отдаются в JSON.

**Структура:** `collector/` — сбор метрик, `handler/` — HTTP-отдача.

- `time.Ticker` в фоновой горутине
- `sync.Mutex` для безопасного доступа к последнему снепшоту
- Пакет `runtime`: память, горутины, GC-циклы
- История снепшотов: `GET /metrics/history`

```bash
cd SystemMonitor && go run .
curl http://localhost:8080/metrics
curl http://localhost:8080/metrics/history
```

---

## 08 — Books API v2 + PostgreSQL

Рефакторинг BookManager: замена in-memory store на PostgreSQL.

- `database/sql` + `pgx` драйвер
- SQL-миграции
- Транзакции и пул соединений

```bash
# cd BooksPostgres
# docker-compose up -d && go run .
```

> ⏳ В плане. На практике эта концепция уже проработана в [Specto](https://github.com/rxritet/Specto) (двойная стратегия BoltDB/PostgreSQL с единым интерфейсом репозитория).

---

## 09 — URL Shortener + JWT Auth

Сервис коротких ссылок с регистрацией и защитой эндпоинтов через JWT.

- Middleware-цепочки
- JWT: `golang-jwt/jwt`
- Хэширование паролей: `bcrypt`

```bash
# cd URLShortener
# go run . → POST /register, POST /login, POST /shorten
```

> ⏳ В плане. JWT-аутентификация уже применена в [HabitDuel](https://github.com/rxritet/HabitDuel) (Dart Shelf сервер, middleware-слой).

---

## 10 — WebSocket Chat

Чат в реальном времени: сервер получает сообщения и рассылает всем подключённым.

- `gorilla/websocket`
- Broadcast через каналы
- Управление состоянием подключённых клиентов

```bash
# cd WSChat && go run .
# http://localhost:8080
```

> ⏳ В плане. WebSocket уже реализован в [HabitDuel](https://github.com/rxritet/HabitDuel) (хаб реального времени дуэлей на Dart Shelf).

---

## ⚙️ Требования

- Go 1.21+
- Docker (для проектов с PostgreSQL)

---

## 📈 Прогресс

![Projects](https://img.shields.io/badge/Выполнено-7%2F10-00ADD8?style=flat-square&logo=go&logoColor=white)
![Language](https://img.shields.io/badge/Language-Go-00ADD8?style=flat-square&logo=go)
![Production](https://img.shields.io/badge/Production—ready-Specto%20%7C%20HabitDuel-success?style=flat-square)

---

*Обновляется по мере прохождения каждого проекта.*
