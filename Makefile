.PHONY: all build test test-short coverage benchmark lint fuzz fuzz-long \
        calibrate build-reproducible build-pgo build-all clean

GO ?= go
BIN := tau
PKG := ./cmd/tau

all: lint test build

build:
	$(GO) build -trimpath -buildvcs=true -o $(BIN) $(PKG)

build-reproducible:
	$(GO) build -trimpath -buildvcs=true \
		-ldflags="-buildid= -X main.buildTimestamp=1778889600" \
		-o $(BIN) $(PKG)

build-pgo:
	$(GO) build -trimpath -pgo=default.pgo -o $(BIN) $(PKG)

build-all:
	GOOS=linux   GOARCH=amd64 $(GO) build -trimpath -o dist/tau-linux-amd64   $(PKG)
	GOOS=linux   GOARCH=arm64 $(GO) build -trimpath -o dist/tau-linux-arm64   $(PKG)
	GOOS=darwin  GOARCH=amd64 $(GO) build -trimpath -o dist/tau-darwin-amd64  $(PKG)
	GOOS=darwin  GOARCH=arm64 $(GO) build -trimpath -o dist/tau-darwin-arm64  $(PKG)
	GOOS=windows GOARCH=amd64 $(GO) build -trimpath -o dist/tau-windows-amd64.exe $(PKG)

test:
	$(GO) test -v -race -cover ./...

test-short:
	$(GO) test -v -short ./...

coverage:
	$(GO) test -coverprofile=coverage.txt -covermode=atomic ./...
	$(GO) tool cover -html=coverage.txt -o coverage.html

benchmark:
	$(GO) test -bench=. -benchmem -run=^$$ ./internal/tau/...

lint:
	golangci-lint run ./...

fuzz:
	$(GO) test -fuzz=. -fuzztime=30s ./internal/tau/invariants/

fuzz-long:
	$(GO) test -fuzz=. -fuzztime=24h ./internal/tau/invariants/

calibrate:
	$(GO) run $(PKG) calibrate $(ARGS)

clean:
	rm -f $(BIN) coverage.txt coverage.html
	rm -rf dist/
