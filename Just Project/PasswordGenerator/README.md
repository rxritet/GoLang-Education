# PasswordGenerator

CLI-утилита на Go для генерации криптографически стойких паролей с использованием `crypto/rand`.

## Структура проекта

```
PasswordGenerator/
├── go.mod
├── main.go                  # Точка входа, парсинг флагов, CLI-вывод
├── README.md
└── generator/
    ├── generator.go         # Логика генерации пароля
    └── generator_test.go    # Unit-тесты (table-driven)
```

## Флаги командной строки

| Флаг              | Короткий | Тип    | По умолчанию | Описание                       |
|-------------------|----------|--------|--------------|--------------------------------|
| `--length`        | `-l`     | `int`  | `12`         | Длина генерируемого пароля     |
| `--numbers`       | `-n`     | `bool` | `false`      | Включить цифры (0-9)          |
| `--symbols`       | `-s`     | `bool` | `false`      | Включить спецсимволы           |
| `--count`         | `-c`     | `int`  | `1`          | Количество паролей             |

Буквы латинского алфавита (a-z, A-Z) включены всегда.

## Интерактивный режим

Если запустить утилиту **без аргументов**, она перейдёт в интерактивный режим и по очереди спросит все параметры:

```
$ ./passgen
=== Password Generator (interactive mode) ===

Password length [12]: 20
Include digits (0-9)? [y/N]: y
Include special symbols? [y/N]: y
How many passwords? [1]: 3

G3$kLp!9qWzR@mN5xYjT
aB7&nQpZ*2wXs!Kd4RtM
Hy8#vLm@1fJz$CwN6eRq
```

## Примеры использования (флаги)

```bash
# Пароль по умолчанию (12 символов, только буквы)
go run main.go

# Короткие флаги — то же, что --length 24 --numbers
go run main.go -l 24 -n

# Пароль длиной 24 символа с цифрами
go run main.go --length 24 --numbers

# Пароль длиной 32 символа с цифрами и спецсимволами
go run main.go --length 32 --numbers --symbols

# Сгенерировать 5 паролей разом
go run main.go -l 16 -n -s -c 5

# Только спецсимволы (без цифр), длина 16
go run main.go --length 16 --symbols
```

### Примеры вывода

```
$ go run main.go --length 20 --numbers --symbols
G3$kLp!9qWzR@mN5xYjT

$ go run main.go
AbCdEfGhIjKl
```

## Сборка

```bash
cd PasswordGenerator
go build -o passgen.exe main.go   # Windows
go build -o passgen main.go       # Linux / macOS
```

После сборки:

```bash
./passgen --length 32 --numbers --symbols
```

## Запуск тестов

```bash
cd PasswordGenerator

# Все тесты
go test ./...

# С подробным выводом
go test -v ./generator/

# С покрытием
go test -cover ./generator/
```
