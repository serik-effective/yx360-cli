BINARY := yx360
PKG := ./...

.PHONY: build test vet fmt lint

build:
	go build -o bin/$(BINARY) ./cmd/yx360

test:
	go test $(PKG)

vet:
	go vet $(PKG)

fmt:
	gofmt -w .

lint: vet
