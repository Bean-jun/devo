.PHONY: all build build-web build-go dev dev-web run-web run-tui test test-web test-e2e clean lint vsix desktop

APP_NAME := devo
GO_ENTRY := ./cmd/devo/
WEB_DIR  := web
BUILD_DIR := build
VSIX_DIR := vscode-extension
DB_PATH  := .env/devo.db

# ========== OS Detection ==========
UNAME_S := $(shell uname -s 2>/dev/null || echo Windows)
ifeq ($(UNAME_S),Linux)
  NULL_DEV := /dev/null
  EXE_EXT  :=
else ifeq ($(UNAME_S),Darwin)
  NULL_DEV := /dev/null
  EXE_EXT  :=
else
  NULL_DEV := NUL
  EXE_EXT  := .exe
endif

# ========== Version ==========
VERSION := $(shell cat VERSION 2>/dev/null || type VERSION 2>$(NULL_DEV))
GIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
GIT_DIRTY := $(shell git status --porcelain 2>/dev/null)
ifneq ($(GIT_DIRTY),)
  DIRTY_SUFFIX := -dirty
else
  DIRTY_SUFFIX :=
endif
FULL_VERSION := $(VERSION)-$(GIT_HASH)$(DIRTY_SUFFIX)

ELECTRON_DIR := electron
DESKTOP_BIN_DIR := $(ELECTRON_DIR)/resources/bin

# garble (Go obfuscator) - optional, fallback to go build if not installed
# go install mvdan.cc/garble@v0.15.0
GARBLE := $(shell command -v garble 2>/dev/null)
ifeq ($(GARBLE),)
  BUILD_CMD := go
  BUILD_FLAGS := 
  OBFUSCATED := 
else
  BUILD_CMD := $(GARBLE)
#   BUILD_FLAGS := -literals -tiny # 混淆&精简模式
  BUILD_FLAGS := -tiny # 精简模式，不混淆，只压缩二进制文件
  OBFUSCATED :=  (obfuscated)
endif

# upx (binary compressor) - optional, skip gracefully if not installed
UPX := $(shell command -v upx 2>/dev/null)

# ========== Build ==========
all: build

build: build-web build-go vsix desktop
	@echo "[OK] Build complete"

# ========== Frontend ==========
build-web:
	@echo "[BUILD] Frontend (version: $(FULL_VERSION))..."
	cd $(WEB_DIR) && npm install && VITE_APP_VERSION=$(VERSION) npm run build

# ========== Backend ==========
build-go:
	@echo "[BUILD] Backend$(OBFUSCATED) (version: $(FULL_VERSION))..."
	mkdir -p $(BUILD_DIR)
	@echo "[BUILD]   Windows (amd64)..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(BUILD_CMD) $(BUILD_FLAGS) build -trimpath -ldflags="-s -w -X main.Version=$(FULL_VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(GO_ENTRY)
	-$(if $(UPX),upx -9 $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe 2>/dev/null,)
	@echo "[BUILD]   Linux (amd64)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(BUILD_CMD) $(BUILD_FLAGS) build -trimpath -ldflags="-s -w -X main.Version=$(FULL_VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(GO_ENTRY)
	-$(if $(UPX),upx -9 $(BUILD_DIR)/$(APP_NAME)-linux-amd64 2>/dev/null,)
	@echo "[BUILD]   macOS (amd64)..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(BUILD_CMD) $(BUILD_FLAGS) build -trimpath -ldflags="-s -w -X main.Version=$(FULL_VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(GO_ENTRY)
	-$(if $(UPX),upx -9 $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 --force-macos 2>/dev/null,)
	@echo "[BUILD]   macOS (arm64)..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(BUILD_CMD) $(BUILD_FLAGS) build -trimpath -ldflags="-s -w -X main.Version=$(FULL_VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(GO_ENTRY)
	-$(if $(UPX),upx -9 $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 --force-macos 2>/dev/null,)
	@echo "[OK] 3 platforms (4 binaries) built"

# ========== VS Code Extension ==========
vsix:
	@echo "[VSIX] Syncing version $(FULL_VERSION) to extension..."
	cd $(VSIX_DIR) && node -e "var p=require('./package.json');p.version='$(VERSION)';require('fs').writeFileSync('package.json',JSON.stringify(p,null,2)+'\n')"
	@echo "[VSIX] Copying binaries to extension..."
	mkdir -p $(VSIX_DIR)/bin
	cp $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(VSIX_DIR)/bin/$(APP_NAME)-windows-amd64.exe
	cp $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(VSIX_DIR)/bin/$(APP_NAME)-linux-amd64
	cp $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(VSIX_DIR)/bin/$(APP_NAME)-darwin-amd64
	cp $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(VSIX_DIR)/bin/$(APP_NAME)-darwin-arm64
	@echo "[VSIX] Packaging extension..."
	cd $(VSIX_DIR) && npm install && npm run vsix
	mv $(VSIX_DIR)/$(APP_NAME)-*.vsix $(BUILD_DIR)/
	@echo "[OK] VSIX package: $(BUILD_DIR)/$(APP_NAME)-*.vsix"

# ========== Desktop (Electron) ==========

desktop:
	@echo "[DESKTOP] Syncing version..."
	cd $(ELECTRON_DIR) && node -e "var p=require('./package.json');p.version='$(VERSION)';require('fs').writeFileSync('package.json',JSON.stringify(p,null,2)+'\n')"
	@echo "[DESKTOP] Copying binaries to Electron resources..."
	mkdir -p $(DESKTOP_BIN_DIR)
	cp $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(DESKTOP_BIN_DIR)/$(APP_NAME)-windows-amd64.exe
	cp $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(DESKTOP_BIN_DIR)/$(APP_NAME)-linux-amd64
	cp $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(DESKTOP_BIN_DIR)/$(APP_NAME)-darwin-amd64
	cp $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(DESKTOP_BIN_DIR)/$(APP_NAME)-darwin-arm64
	@echo "[DESKTOP] Installing Electron dependencies..."
	cd $(ELECTRON_DIR) && npm install
	@echo "[DESKTOP] Cleaning previous Electron dist..."
	rm -rf $(ELECTRON_DIR)/dist
	@echo "[DESKTOP] Packaging Electron app..."
	cd $(ELECTRON_DIR) && npm run package
	mv $(ELECTRON_DIR)/dist/$(APP_NAME)-*.exe $(BUILD_DIR)/ 2>/dev/null || true
	mv $(ELECTRON_DIR)/dist/$(APP_NAME)-* $(BUILD_DIR)/ 2>/dev/null || true
	@echo "[OK] Desktop package complete"

# ========== Development ==========
dev:
	@echo "[DEV] Starting frontend + backend..."
	cd $(WEB_DIR) && npm run dev &
	go run $(GO_ENTRY) --web --port 8080

dev-web:
	@echo "[DEV] Frontend dev server..."
	cd $(WEB_DIR) && npm run dev

# ========== Run ==========
run-web: build
	@echo "[RUN] Web mode..."
	$(BUILD_DIR)/$(APP_NAME)$(EXE_EXT) --web --port 8080

run-tui: build
	@echo "[RUN] TUI mode..."
	$(BUILD_DIR)/$(APP_NAME)$(EXE_EXT) --tui

# ========== Test ==========
test: test-web test-go

test-web:
	@echo "[TEST] Frontend unit tests..."
	cd $(WEB_DIR) && npm test

test-e2e:
	@echo "[TEST] E2E tests..."
	cd $(WEB_DIR) && npm run test:e2e

test-go:
	@echo "[TEST] Backend tests..."
	go test ./...

# ========== Lint ==========
lint:
	@echo "[LINT] Frontend..."
	cd $(WEB_DIR) && npm run lint
	@echo "[LINT] Backend..."
	go vet ./...

# ========== Clean ==========
clean:
	@echo "[CLEAN] Removing build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(WEB_DIR)/dist
	rm -rf $(VSIX_DIR)/bin
	rm -f $(VSIX_DIR)/*.vsix
	rm -f $(BUILD_DIR)/*.vsix
	rm -rf $(ELECTRON_DIR)/node_modules
	rm -rf $(ELECTRON_DIR)/dist
	rm -rf $(DESKTOP_BIN_DIR)
	rm -f $(DB_PATH)
	@echo "[OK] Clean complete"

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