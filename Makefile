APP_NAME := emuai
CMD_PATH := ./cmd/emuai
BIN_DIR := ./bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)
GO := go
UNAME_S := $(shell uname -s)
DARWIN_VULKAN_LIB_CANDIDATES := $(if $(VULKAN_SDK),$(VULKAN_SDK)/lib,) $(HOME)/VulkanSDK/latest/macOS/lib /usr/local/lib /opt/homebrew/lib
DARWIN_VULKAN_LIB_DIRS := $(strip $(foreach dir,$(DARWIN_VULKAN_LIB_CANDIDATES),$(if $(wildcard $(dir)/libMoltenVK.dylib),$(dir),)))
DARWIN_VULKAN_CGO_LDFLAGS := $(strip $(foreach dir,$(DARWIN_VULKAN_LIB_DIRS),-L$(dir)))
TEST_PACKAGES := ./...
TEST_COVER_PROFILE := coverage.out
TEST_FLAGS := -count=1
TEST_DEEP_FLAGS := -count=1 -race -shuffle=on -covermode=atomic -coverprofile=$(TEST_COVER_PROFILE) -v -test.fullpath=true

.PHONY: help build build-vulkan run test test-deep fmt clean

help:
	@echo "Targets disponibles:"
	@echo "  make build  - Compile le binaire standard"
	@echo "  make build-vulkan - Compile le binaire avec le backend Vulkan"
	@echo "  make run    - Lance l'emulateur"
	@echo "  make test   - Execute rapidement tous les tests"
	@echo "  make test-deep - Execute tous les tests en profondeur avec race et couverture"
	@echo "  make fmt    - Formate le code Go"
	@echo "  make clean  - Supprime le dossier bin"

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_PATH) $(CMD_PATH)

build-vulkan:
	@mkdir -p $(BIN_DIR)
ifeq ($(UNAME_S),Darwin)
	@set -e; \
	if [ -n "$(DARWIN_VULKAN_CGO_LDFLAGS)" ]; then \
		CGO_LDFLAGS="$(DARWIN_VULKAN_CGO_LDFLAGS)" $(GO) build -tags vulkan -o $(BIN_PATH) $(CMD_PATH); \
	else \
		$(GO) build -tags vulkan -o $(BIN_PATH) $(CMD_PATH); \
		echo "warning: libMoltenVK.dylib not found in VULKAN_SDK, ~/VulkanSDK/latest/macOS/lib, /usr/local/lib, or /opt/homebrew/lib" >&2; \
	fi; \
	for dir in $(DARWIN_VULKAN_LIB_DIRS); do \
		install_name_tool -add_rpath "$$dir" $(BIN_PATH) 2>/dev/null || true; \
	done
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
