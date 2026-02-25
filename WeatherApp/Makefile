.PHONY: build run test clean

BINARY_NAME=weather
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/weather

run:
	go run ./cmd/weather -city="Almaty"

test:
	go test -v -count=1 ./internal/weather/...

clean:
	rm -rf $(BUILD_DIR)
