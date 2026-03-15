.PHONY: build run test clean migrate dev

APP_NAME := silentpass
BUILD_DIR := bin

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server

run: build
	./$(BUILD_DIR)/$(APP_NAME)

dev:
	go run ./cmd/server

test:
	go test ./... -v -race

clean:
	rm -rf $(BUILD_DIR)

migrate-up:
	psql $(DATABASE_URL) -f migrations/001_initial.sql

lint:
	golangci-lint run ./...

docker-build:
	docker build -t $(APP_NAME) .

docker-run:
	docker-compose up -d
