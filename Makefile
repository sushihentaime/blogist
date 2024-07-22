GOPATH:=$(shell go env GOPATH)

.PHONY: migrate-init
migrate-init:
	@go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@migrate create -ext sql -dir ./migrations -seq $(NAME)

.PHONY: test
test:
	@go test -v ./... 

.PHONY: docbuild
docbuild:
	@docker compose build

.PHONY: docup
docup:
	@docker compose up

.PHONY: docdown
docdown:
	@docker compose down
