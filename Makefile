.PHONY: fmt fmt-check test vet check build

fmt:
	go fmt ./...

fmt-check:
	test -z "$$(gofmt -l $$(git ls-files '*.go'))"

test:
	go test ./...

vet:
	go vet ./...

check: fmt-check test vet

build:
	go build ./cmd/...
