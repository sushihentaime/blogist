GOPATH:=$(shell go env GOPATH)

.PHONY: migrate-init
migrate-init:
	@go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@migrate create -ext sql -dir ./migrations -seq $(NAME)

.PHONY: test
test:
	@go test -v ./... -cover  

.PHONY: build
build:
	@docker compose build

.PHONY: up
up:
	@docker compose up

.PHONY: down
down:
	@docker compose down

# Quality Control
.PHONY: audit
audit:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

.PHONY: vendor
vendor:
	@go mod tidy
	@go mod verify
	@go mod vendor
