BINARY     := gobalancer
BUILD_DIR  := bin
CMD_DIR    := ./cmd/gobalancer

GOLANGCI   := golangci-lint
GOTEST     := go test
GOBUILD    := go build

.PHONY: all build test lint race clean docker run

all: lint test build

build:
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY) $(CMD_DIR)

test:
	$(GOTEST) ./... -v -timeout 60s

race:
	$(GOTEST) -race ./... -timeout 60s

lint:
	$(GOLANGCI) run ./...

bench:
	$(GOTEST) -bench=. -benchmem ./...

clean:
	@rm -rf $(BUILD_DIR)

docker:
	docker build -t $(BINARY):latest .

run: build
	./$(BUILD_DIR)/$(BINARY) -config config/config.yaml

.DEFAULT_GOAL := all
