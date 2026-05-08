GOLANGCI_LINT_VERSION ?= v2.12.2

.PHONY: lint test

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run

test:
	go test ./...
