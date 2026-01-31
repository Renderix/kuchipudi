.PHONY: build run test clean install plugins

APP_NAME := kuchipudi
BIN_DIR := bin
PLUGIN_DIR := plugins

build:
	go build -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)

run: build
	./$(BIN_DIR)/$(APP_NAME)

test:
	go test ./... -v

test-short:
	go test ./... -v -short

clean:
	rm -rf $(BIN_DIR)
	rm -f plugins/*/$(shell basename $(PLUGIN_DIR)/*)

plugins:
	@for dir in $(PLUGIN_DIR)/*/; do \
		name=$$(basename $$dir); \
		echo "Building plugin: $$name"; \
		cd $$dir && go build -o $$name . && cd ../..; \
	done

install: build plugins
	mkdir -p ~/.$(APP_NAME)/plugins
	cp $(BIN_DIR)/$(APP_NAME) /usr/local/bin/
	cp -r $(PLUGIN_DIR)/* ~/.$(APP_NAME)/plugins/
	cp -r web ~/.$(APP_NAME)/

lint:
	golangci-lint run

fmt:
	go fmt ./...
