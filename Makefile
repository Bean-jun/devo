.PHONY: all build build-web build-go dev dev-web run-web run-tui test test-web test-e2e clean lint

APP_NAME := devo
GO_ENTRY := cmd/devo/main.go
WEB_DIR  := web
BUILD_DIR := build
DB_PATH  := .env/devo.db

# ========== Build ==========
all: build

build: build-web build-go
	@echo [OK] Build complete: $(BUILD_DIR)/$(APP_NAME).exe

build-web:
	@echo [BUILD] Frontend...
	cd $(WEB_DIR) && npm install && npm run build

build-go:
	@echo [BUILD] Backend...
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME).exe $(GO_ENTRY)

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
	@echo "Other:"
	@echo "  make lint          Code linting"
	@echo "  make clean         Remove build artifacts"
	@echo ""
	@echo "Environment Variables:"
	@echo "  DEVO_LLM_BASE_URL   LLM API base URL"
	@echo "  DEVO_LLM_API_KEY    LLM API key"
	@echo "  DEVO_LLM_MODEL      LLM model name"
	@echo "  DEVO_DB_PATH        Database path (default: .env/devo.db)"
	@echo "  DEVO_LOG_PATH       Log file path"