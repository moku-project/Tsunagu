.PHONY: dev proto proto-lint build-backend build-sandbox test lint clean

dev:
	docker compose up --build

proto:
	buf generate

proto-lint:
	buf lint
	buf breaking --against '.git#branch=main'

build-backend:
	cd backend && go build -o bin/tsunagu ./cmd/server

build-sandbox:
	cd sandbox && ./gradlew build

test:
	cd backend && go test ./...
	cd sandbox && ./gradlew test

lint:
	cd backend && golangci-lint run ./...

clean:
	rm -rf backend/bin
	cd sandbox && ./gradlew clean
