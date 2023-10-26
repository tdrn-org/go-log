MAKEFLAGS += --no-print-directory

GOBIN ?= $(shell go env GOPATH)/bin

.DEFAULT_GOAL := check

.PHONY: deps
deps:
	go mod download -x

.PHONY: testdeps
testdeps: deps
	go install honnef.co/go/tools/cmd/staticcheck@2023.1.6

.PHONY: tidy
tidy:
	go mod verify
	go mod tidy

.PHONY: vet
vet: testdeps
	go vet ./...

.PHONY: staticcheck
staticcheck: testdeps
	$(GOBIN)/staticcheck ./...

.PHONY: lint
lint: vet staticcheck

.PHONY: test
test: testdeps
	go test -v -coverpkg=./... -covermode=atomic -coverprofile=coverage.out ./...

.PHONY: check
check: test lint

.PHONY: clean
clean:
	go clean ./...
