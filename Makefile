.PHONY: run test fmt vet compose-up compose-down

run:
	cd backend && go run ./cmd/server

test:
	cd backend && go test ./...

fmt:
	cd backend && gofmt -w .

vet:
	cd backend && go vet ./...

compose-up:
	docker compose up --build

compose-down:
	docker compose down
