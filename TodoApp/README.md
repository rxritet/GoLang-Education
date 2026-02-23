# 02 — Todo CLI

> **Go Learning Path · Project 02**  
> Цель: `flag`, `os`, `encoding/json`, персистентность без БД.  
> Предыдущий проект: [01 · Books REST API](../BookManager)

CLI-менеджер задач с хранением данных в JSON-файле. Только стандартная библиотека.

---

## Быстрый старт

```bash
cd TodoApp
go run . --interactive   # интерактивный режим (рекомендуется)
```

или разовые команды:

```bash
go run . --add "Выучить горутины"
go run . --list
go run . --done 1
go run . --delete 2
```

---

## Команды

### Одиночные флаги

| Команда                         | Описание                                |
| ------------------------------- | --------------------------------------- |
| `go run . --add "текст"`        | Добавить задачу, вывести присвоенный ID |
| `go run . --list`               | Показать все задачи в виде таблицы      |
| `go run . --done <id>`          | Отметить задачу выполненной             |
| `go run . --delete <id>`        | Удалить задачу                          |
| `go run . --interactive` / `-i` | Запустить интерактивный REPL-режим      |
| `go run .` (без флагов)         | Показать справку и выйти с кодом 1      |

### Интерактивный режим (`--interactive`)

Терминал остаётся открытым — вводи команды до тех пор, пока не напишешь `exit`.

```
todo> add Написать unit-тесты
Added: [1] Написать unit-тесты
todo> list
ID    Status  Title                           Created
----  ------  ------------------------------  -------------------
1     [ ]     Написать unit-тесты             2026-02-23 19:10
todo> done 1
Done: [1] Написать unit-тесты
todo> exit
Bye!
```

Доступные команды внутри REPL:

| Команда       | Псевдонимы  | Описание             |
| ------------- | ----------- | -------------------- |
| `add <title>` | —           | Добавить задачу      |
| `list`        | `ls`        | Показать все задачи  |
| `done <id>`   | —           | Отметить выполненной |
| `delete <id>` | `del`, `rm` | Удалить задачу       |
| `help`        | `h`, `?`    | Справка              |
| `exit`        | `quit`, `q` | Выйти                |

---

## Вывод `--list`

```
ID    Status  Title                           Created
----  ------  ------------------------------  -------------------
1     [✓]     Выучить горутины               2026-02-23 10:30
2     [ ]     Написать unit-тесты            2026-02-23 09:15
```

---

## Структура проекта

```
TodoApp/
├── main.go       # Парсинг флагов, роутинг команд
├── todo.go       # Тип Todo, тип Store, методы Add/Complete/Delete/Print
├── storage.go    # load(path) и save(path, store) — JSON I/O
├── repl.go       # Интерактивный REPL-режим
├── go.mod        # module todo-cli, go 1.21
└── todos.json    # Создаётся автоматически (в .gitignore)
```

---

## Детали реализации

- **IDs** монотонно растут: `max(existing IDs) + 1` — никогда не переиспользуются
- **Первый запуск**: если `todos.json` не существует — `load` возвращает пустой `Store` без ошибки
- **Персистентность**: данные сохраняются после каждой мутирующей операции
- **Ошибки**: выводятся в `stderr`, процесс завершается с кодом `1`
- **Зависимости**: только стандартная библиотека Go (`flag`, `encoding/json`, `os`, `bufio`)

---

## Обработка ошибок

```bash
go run . --done 99     # stderr: "todo 99 not found", exit 1
go run . --add ""      # stderr: flag needs an argument, exit 1
go run .              # stderr: usage, exit 1
```

---

## Концепции Go из этого проекта

| Концепция                      | Где используется                                       |
| ------------------------------ | ------------------------------------------------------ |
| `flag`                         | Парсинг CLI-аргументов в `main.go`                     |
| `encoding/json`                | `json.MarshalIndent` / `json.Unmarshal` в `storage.go` |
| `os.ReadFile` / `os.WriteFile` | Чтение и запись `todos.json`                           |
| `os.IsNotExist`                | Graceful handling первого запуска                      |
| `bufio.Scanner`                | Построчное чтение stdin в REPL                         |
| Pointer receivers              | `(s *Store) Add`, `Complete`, `Delete`                 |
| Value receiver                 | `(s Store) Print` — только чтение                      |
