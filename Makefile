APP_NAME := emuai
CMD_PATH := ./cmd/emuai
BIN_DIR := ./bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)
GO := go
TEST_PACKAGES := ./...
TEST_COVER_PROFILE := coverage.out
TEST_FLAGS := -count=1
TEST_DEEP_FLAGS := -count=1 -race -shuffle=on -covermode=atomic -coverprofile=$(TEST_COVER_PROFILE) -v -test.fullpath=true

.PHONY: help build run test test-deep fmt clean

help:
	@echo "Targets disponibles:"
	@echo "  make build  - Compile le binaire"
	@echo "  make run    - Lance l'emulateur"
	@echo "  make test   - Execute rapidement tous les tests"
	@echo "  make test-deep - Execute tous les tests en profondeur avec race et couverture"
	@echo "  make fmt    - Formate le code Go"
	@echo "  make clean  - Supprime le dossier bin"

build:
	@mkdir -p $(BIN_DIR)
ifeq ($(shell uname -s),Darwin)
ifdef VULKAN_SDK
	CGO_LDFLAGS="-L$(VULKAN_SDK)/lib" $(GO) build -tags vulkan -ldflags '-extldflags "-rpath $(VULKAN_SDK)/lib"' -o $(BIN_PATH) $(CMD_PATH)
else
	$(GO) build -tags vulkan -o $(BIN_PATH) $(CMD_PATH)
endif
else
	$(GO) build -tags vulkan -o $(BIN_PATH) $(CMD_PATH)
endif

run:
	$(GO) run $(CMD_PATH)

test:
	$(GO) test $(TEST_FLAGS) $(TEST_PACKAGES)

test-deep:
	$(GO) test $(TEST_DEEP_FLAGS) $(TEST_PACKAGES)

fmt:
	find . -name '*.go' -type f -print0 | xargs -0 gofmt -w

clean:
	rm -rf $(BIN_DIR)
