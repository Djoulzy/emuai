APP_NAME := emuai
CMD_PATH := ./cmd/emuai
BIN_DIR := ./bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)

.PHONY: help build run test fmt clean

help:
	@echo "Targets disponibles:"
	@echo "  make build  - Compile le binaire"
	@echo "  make run    - Lance l'emulateur"
	@echo "  make test   - Execute les tests"
	@echo "  make fmt    - Formate le code Go"
	@echo "  make clean  - Supprime le dossier bin"

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_PATH) $(CMD_PATH)

run:
	go run $(CMD_PATH)

test:
	go test ./...

fmt:
	find . -name '*.go' -type f -print0 | xargs -0 gofmt -w

clean:
	rm -rf $(BIN_DIR)
