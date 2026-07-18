.PHONY: fmt fmt-check test test-race vet shell-check check build

fmt:
	go fmt ./...

fmt-check:
	test -z "$$(gofmt -l $$(git ls-files --cached --others --exclude-standard '*.go'))"

test:
	go test ./...

vet:
	go vet ./...

shell-check:
	for file in $$(git ls-files --cached --others --exclude-standard '*.sh'); do sh -n "$$file"; done

test-race:
	go test -race ./...

check: fmt-check test vet shell-check

build:
	go build ./cmd/...
