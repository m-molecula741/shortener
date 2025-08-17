run:
	DATABASE_DSN="postgres://postgres:741852963@localhost:5432/template" go run ./cmd/shortener

build:
	go build -o shortener ./cmd/shortener

run-build:
	./shortener

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
