.PHONY: all build build-web build-go dev dev-web run-web run-tui test test-web test-e2e clean lint vsix

APP_NAME := devo
GO_ENTRY := cmd/devo/
WEB_DIR  := web
BUILD_DIR := build
VSIX_DIR := vscode-extension
DB_PATH  := .env/devo.db

# ========== Version ==========
VERSION := $(shell type VERSION)
GIT_HASH := $(shell git rev-parse --short HEAD)
GIT_DIRTY := $(shell git status --porcelain)
ifneq ($(GIT_DIRTY),)
  DIRTY_SUFFIX := -dirty
else
  DIRTY_SUFFIX :=
endif
FULL_VERSION := $(VERSION)-$(GIT_HASH)$(DIRTY_SUFFIX)

# ========== Build ==========
all: build

build: vsix
	@echo [OK] Build complete: $(BUILD_DIR)/$(APP_NAME).exe

build-web:
	@echo [BUILD] Frontend (version: $(FULL_VERSION))...
	cd $(WEB_DIR) && npm install && set VITE_APP_VERSION=$(FULL_VERSION) && npm run build

build-go: build-web
	@echo [BUILD] Backend (version: $(FULL_VERSION))...
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	go build -ldflags="-s -w -X main.Version=$(FULL_VERSION)" -o $(BUILD_DIR)/$(APP_NAME).exe $(GO_ENTRY)

# ========== VS Code Extension ==========
vsix: build-go
	@echo [VSIX] Syncing version $(FULL_VERSION) to extension...
	cd $(VSIX_DIR) && node -e "var p=require('./package.json');p.version='$(FULL_VERSION)';require('fs').writeFileSync('package.json',JSON.stringify(p,null,2)+'\n')"
	@echo [VSIX] Copying binary to extension...
	@if not exist $(VSIX_DIR)\bin mkdir $(VSIX_DIR)\bin
	upx -9 $(BUILD_DIR)\$(APP_NAME).exe
	copy $(BUILD_DIR)\$(APP_NAME).exe $(VSIX_DIR)\bin\$(APP_NAME).exe
	@echo [VSIX] Packaging extension...
	cd $(VSIX_DIR) && npm run vsix
	cmd /c "move $(VSIX_DIR)\$(APP_NAME)-*.vsix $(BUILD_DIR)\"
	@echo [OK] VSIX package: $(BUILD_DIR)\$(APP_NAME)-*.vsix

# ========== Development ==========
dev:
	@echo [DEV] Starting frontend + backend...
	start cmd /c "cd $(WEB_DIR) && npm run dev"
	go run $(GO_ENTRY) --web --port 8080

dev-web:
	@echo [DEV] Frontend dev server...
	cd $(WEB_DIR) && npm run dev

# ========== Run ==========
run-web: build
	@echo [RUN] Web mode...
	$(BUILD_DIR)/$(APP_NAME).exe --web --port 8080

run-tui: build
	@echo [RUN] TUI mode...
	$(BUILD_DIR)/$(APP_NAME).exe --tui

# ========== Test ==========
test: test-web test-go

test-web:
	@echo [TEST] Frontend unit tests...
	cd $(WEB_DIR) && npm test

test-e2e:
	@echo [TEST] E2E tests...
	cd $(WEB_DIR) && npm run test:e2e

test-go:
	@echo [TEST] Backend tests...
	go test ./...

# ========== Lint ==========
lint:
	@echo [LINT] Frontend...
	cd $(WEB_DIR) && npm run lint
	@echo [LINT] Backend...
	go vet ./...

# ========== Clean ==========
clean:
	@echo [CLEAN] Removing build artifacts...
	rm -rf $(BUILD_DIR)
	rm -rf $(WEB_DIR)/dist
	rm -rf $(VSIX_DIR)\bin
	rm -f $(VSIX_DIR)\*.vsix
	rm -f $(BUILD_DIR)\*.vsix
	rm -f $(DB_PATH)
	@echo [OK] Clean complete

# ========== Help ==========
help:
	@echo "Devo Build Makefile"
	@echo "===================="
	@echo ""
	@echo "Build:"
	@echo "  make build         Build frontend + backend"
	@echo "  make build-web     Build frontend only"
	@echo "  make build-go      Build backend only"
	@echo ""
	@echo "Development:"
	@echo "  make dev           Start frontend dev server + backend (web mode)"
	@echo "  make dev-web       Start frontend dev server only"
	@echo ""
	@echo "Run:"
	@echo "  make run-web       Build and run in Web mode"
	@echo "  make run-tui       Build and run in TUI mode"
	@echo ""
	@echo "Test:"
	@echo "  make test          Run all tests"
	@echo "  make test-web      Run frontend unit tests"
	@echo "  make test-e2e      Run E2E tests"
	@echo "  make test-go       Run backend tests"
	@echo ""
	@echo "VS Code Extension:"
	@echo "  make vsix          Build VS Code extension .vsix package"
	@echo ""
	@echo "Other:"
	@echo "  make lint          Code linting"
	@echo "  make clean         Remove all build artifacts"
	@echo ""
	@echo "Environment Variables:"
	@echo "  DEVO_LLM_BASE_URL   LLM API base URL"
	@echo "  DEVO_LLM_API_KEY    LLM API key"
	@echo "  DEVO_LLM_MODEL      LLM model name"
	@echo "  DEVO_DB_PATH        Database path (default: .env/devo.db)"
	@echo "  DEVO_LOG_PATH       Log file path"