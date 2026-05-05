BINARY    := searchcore
BUILD_DIR := bin
CMD_DIR   := cmd/server

.PHONY: build test proto seed docker-up docker-down lint clean run

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./$(CMD_DIR)

test:
	go test -v -race ./...

test-short:
	go test -short ./...

proto:
	protoc --go_out=. --go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		proto/searchcore.proto

seed:
	go run ./scripts/seed.go

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-reset:
	docker-compose down -v

run: build
	./$(BUILD_DIR)/$(BINARY)

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)
