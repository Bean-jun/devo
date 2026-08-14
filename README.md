# Devo

English | [中文](README.zh-CN.md)

**Devo** (Developer + Evolution) is a session-centric, conversation-driven autonomous coding AI agent. It runs locally on your machine, with direct access to the file system and Shell, powered by LLMs to complete coding tasks.

---

## Features

- **Conversation-Driven** — Describe tasks in natural language; the AI autonomously plans, codes, runs, fixes, and iterates until done
- **Five-Client Operation** — Web Control Center, Terminal TUI, VS Code Extension, Electron Desktop App, Mobile touch-optimized — all from a single codebase
- **Web Control Center** — Three-panel layout with the chat always in focus; right-side panels host Files / Skills / Memory / Dashboard / Settings / Terminal / MCP / Background Processes
- **Multi-Workspace Management** — Seamlessly switch between multiple projects, with workspace-directory sync
- **Approval Gating** — Risk-based operation classification (High / Medium / Low / None), file edits with diff preview, and YOLO auto-approve mode
- **Long-Term Memory + Skill Evolution** — Remembers preferences and project experience; distills Skill instruction sets from conversations for cross-session reuse (Global + Project tiers)
- **Context Compression** — Auto-compresses long conversations into summaries, breaking through context window limits
- **Message Rollback** — Roll back to any point in history and retry without affecting the file system
- **Chain-of-Thought Reasoning** — Supports LLM Chain-of-Thought (CoT) reasoning mode with configurable reasoning intensity
- **Image Understanding** — Multimodal support for Base64 image input with image compression preprocessing
- **MCP Extensions** — Dynamic external tool integration via the MCP protocol, with workspace-level isolation
- **Background Process Management** — Real-time streaming output in blocking mode, with Prompt Cache monitoring
- **Structured Logging** — Log levels and trace-based request tracking
- **Cross-Platform** — Full support for Linux, macOS, and Windows

---

## Architecture Overview

| Tier | Modules | Responsibilities |
|:----:|---------|------------------|
| **UI** | Web · TUI · Mobile · VS Code · Desktop | Multi-client user interaction entry points |
| ↓ | | |
| **Interface** | REST API · SSE Event Stream · Approval Bridge | HTTP serving, real-time push, approval callbacks |
| ↓ | | |
| **Task Execution** | Toolset · Python Sandbox · LLM Client · MCP · Image Compression · Path Security | Tool execution, code sandbox, model invocation |
| ↓ | | |
| **Core** | Agent Loop · Approval Gating · Context Compression · Message Rollback · Long-Term Memory · Skills · Session Archive · Concurrency Isolation · Crash Recovery | Agent loop, state management, fault tolerance |
| ↓ | | |
| **Storage** | SQLite Persistence | Session / Message / Event storage |

**Deployment Model**: The local core (required) runs on the developer's machine with direct file system access. An optional team service only handles statistics reporting and remote approval relay — **no code or conversation content is ever uploaded**.

---

## Tech Stack

| Tier | Technology |
|------|------------|
| Backend Language | Go 1.25+ |
| CLI Framework | [Cobra](https://github.com/spf13/cobra) |
| TUI Framework | [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| Frontend Framework | Vue 3 + TypeScript + Vite |
| State Management | Pinia |
| Routing | Vue Router |
| Markdown Rendering | marked + highlight.js |
| Icon Library | Phosphor Icons |
| Database | SQLite (GORM) |
| LLM Protocol | OpenAI-compatible API |
| Testing | Vitest + Playwright (Frontend) / Go testing (Backend) |
| Desktop | Electron |

---

## Quick Start

### Prerequisites

- Go 1.25+
- Node.js 22+ (for Web frontend builds)
- Python 3.8+ (for command execution sandbox)

### Configuration

Devo supports multiple ways to configure LLMs — no manual config file editing required:

**Option 1: Web UI Configuration (Recommended)**

After starting Devo, a configuration onboarding dialog will appear automatically on first use. You can also add and manage models anytime via the **Settings Panel → Global Settings** on the right side.

**Option 2: CLI Configuration**

```bash
# Interactive configuration wizard (recommended for first-time use)
devo config onboard

# Add a model directly
devo config models add --name "GPT-4o" --api-key "sk-xxx" --model "gpt-4o"

# Manage models
devo config models list                # List all models
devo config models activate --id gpt-4o  # Activate a model
devo config models test --id gpt-4o      # Test model connectivity
devo config models remove --id gpt-4o    # Remove a model
```

**Option 3: Environment Variables**

```bash
export DEVO_LLM_API_KEY="sk-your-key-here"
export DEVO_LLM_BASE_URL="https://api.openai.com/v1"
export DEVO_LLM_MODEL="gpt-4o"
```

Environment variables such as `DEVO_DB_PATH` and `DEVO_LOG_PATH` are also supported to override default paths.

### Build

```bash
make build          # Build backend + Web frontend + VS Code extension + Electron desktop
make build-web      # Build Web frontend only
make build-go       # Build Go backend only (4 platform binaries)
make vsix           # Package VS Code extension only
make desktop        # Package Electron desktop app only
```

### Run

```bash
# Web mode (default, auto-opens browser)
devo

# Specify port and working directory
devo -web -port 9090 -workspace /path/to/project

# TUI terminal mode
devo -tui

# View help
devo --help
```

### Development Mode

```bash
make dev            # Start frontend dev server + backend simultaneously (web mode)
make dev-web        # Start frontend dev server only
make run-web        # Build and run in Web mode
make run-tui        # Build and run in TUI mode
```

### Testing

```bash
make test           # Run all tests (frontend + backend)
make test-web       # Frontend unit tests
make test-e2e       # Frontend E2E tests (Playwright)
make test-go        # Backend tests
make lint           # Code linting (frontend ESLint + backend go vet)
```

### Quick API Tryout

```bash
# Create a session
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{"working_directory": "/path/to/your/project"}'

# Send a message
curl -X POST http://localhost:8080/api/v1/sessions/{id}/messages \
  -H "Content-Type: application/json" \
  -d '{"content": "Help me refactor this module"}'
```

For the full API documentation, see [Architecture Design](docs/2-architecture.md).

---

## Project Structure

```
Devo/
├── cmd/devo/              ← Program entry point (main.go)
├── internal/              ← Go internal packages
│   ├── cli/               ← CLI command definitions & app bootstrap
│   │   ├── commands/      ← Cobra commands (root / config)
│   │   ├── app.go         ← Application initialization
│   │   ├── server.go      ← HTTP server startup
│   │   └── platform.go    ← Platform detection
│   ├── config/            ← Configuration management (global/project/paths)
│   ├── core/              ← Core layer
│   │   ├── agentloop/     ← Agent loop, state machine, approval handling, crash recovery, rollback
│   │   ├── approval/      ← Approval gating & risk classification
│   │   ├── archive/       ← Session Markdown archiving
│   │   ├── compressor/    ← Context compression
│   │   ├── concurrency/   ← Concurrency control (path locking)
│   │   ├── memory/        ← Long-term memory management
│   │   ├── prompt/        ← System prompt assembly, agents.md loading, directory tree
│   │   ├── session/       ← Session model, event bus, in-memory store
│   │   ├── skills/        ← Skills management & experience crystallization
│   │   └── tokenmeter/    ← Token metering
│   ├── interfaces/        ← Interface layer
│   │   ├── rest/          ← REST API + SSE event stream (20+ handlers)
│   │   └── tui/           ← Bubble Tea TUI interface
│   │       ├── api/       ← TUI backend API client
│   │       ├── components/← TUI components (status bar / toast / styles)
│   │       ├── overlays/  ← TUI overlays (approval / background / commands / dashboard / MCP, etc.)
│   │       ├── renderer/  ← Markdown renderer
│   │       └── types/     ← TUI type definitions
│   ├── pkg/               ← Shared utility packages
│   │   ├── logging/       ← Structured logging
│   │   └── process/       ← Process management
│   ├── storage/           ← Storage layer
│   │   └── sqlite/        ← SQLite persistence (sessions / messages / events / rollback / SSE / usage)
│   ├── taskexec/          ← Task execution layer
│   │   ├── imageproc/     ← Image compression preprocessing
│   │   ├── llmclient/     ← LLM client (OpenAI-compatible protocol)
│   │   ├── mcp/           ← MCP protocol client management
│   │   ├── pathsec/       ← Path security validation, .gitignore parsing
│   │   └── tools/         ← Toolset (read/write files, execute commands, search, glob, diff, etc.)
│   └── update/            ← Version update checking
├── web/                   ← Web frontend (Vue 3 + TypeScript + Vite)
│   ├── src/
│   │   ├── layouts/       ← Three layouts: BrowserLayout / VscodeLayout / MobileLayout
│   │   ├── components/    ← Chat / layout / modals / commands / mobile / editor
│   │   ├── panels/        ← Right-side panels (files / skills / memory / dashboard / settings / terminal / mcp / background)
│   │   ├── stores/        ← Pinia state management
│   │   ├── composables/   ← Reusable logic
│   │   ├── types/         ← TypeScript type definitions
│   │   └── styles/        ← CSS variables / base styles / animations
│   └── e2e/               ← Playwright E2E tests
├── electron/              ← Electron desktop app
│   ├── main.js            ← Electron main process
│   ├── welcome.html       ← Welcome page
│   └── resources/         ← Platform binaries & icons
├── vscode-extension/      ← VS Code extension
│   ├── dist/              ← Compiled extension code
│   ├── bin/               ← Per-platform backend binaries
│   └── esbuild.js         ← Extension bundling script
├── docs/                  ← Design documents
│   ├── 1-PRD.md           ← Product Requirements Document
│   ├── 2-architecture.md  ← Full Architecture Design
│   ├── 3-cli-architecture.md ← CLI Architecture Design
│   ├── 3-frontend-design.md  ← Frontend Design Specification
│   ├── 4-web-architecture.md ← Web Frontend Engineering Architecture
│   ├── 5-web-testing.md   ← Web Testing Strategy
│   ├── 6-agent-loop-event-driven-refactor.md ← Agent Loop Event-Driven Refactor
│   ├── 7-electron-desktop-architecture.md ← Electron Desktop Architecture
│   ├── 8-session-state-robustness-test.md  ← Session State Robustness Testing
│   ├── 9-performance-optimization.md       ← Performance Optimization
│   ├── 10-mobile-layout-design.md          ← Mobile Layout Design
│   ├── 11-llm-reasoning-cot-design.md      ← LLM CoT Reasoning Design
│   ├── 12-image-input-design.md            ← Image Input Design
│   └── 13-llm-config-onboarding-design.md  ← LLM Config Onboarding Design
├── build/                 ← Build output directory
├── .github/workflows/     ← CI/CD workflows
│   ├── ci.yml             ← Continuous Integration (lint + test + build)
│   └── release.yml        ← Release (multi-platform build + packaging)
├── .devo/                 ← Runtime data
│   ├── config.json        ← Global configuration
│   └── sessions/          ← Session archives
├── Makefile               ← Build scripts
├── VERSION                ← Version number
└── go.mod                 ← Go module definition
```

---

## Further Reading

- [PRD](docs/1-PRD.md) — Product requirements & typical workflows
- [Full Architecture Design](docs/2-architecture.md) — Detailed system design document
- [CLI Architecture](docs/3-cli-architecture.md) — CLI & TUI architecture
- [Frontend Design](docs/3-frontend-design.md) — Web frontend UI/UX design
- [Web Frontend Engineering Architecture](docs/4-web-architecture.md) — Web frontend engineering approach
- [Web Testing Strategy](docs/5-web-testing.md) — Frontend testing strategy
- [Agent Loop Event-Driven Refactor](docs/6-agent-loop-event-driven-refactor.md) — Agent loop refactoring design
- [Electron Desktop Architecture](docs/7-electron-desktop-architecture.md) — Desktop app architecture design
- [Session State Robustness Testing](docs/8-session-state-robustness-test.md) — State machine testing approach
- [Performance Optimization](docs/9-performance-optimization.md) — Performance optimization strategies
- [Mobile Layout Design](docs/10-mobile-layout-design.md) — Mobile adaptation approach
- [LLM CoT Reasoning Design](docs/11-llm-reasoning-cot-design.md) — LLM Chain-of-Thought reasoning integration
- [Image Input Design](docs/12-image-input-design.md) — Multimodal image input approach
- [LLM Config Onboarding Design](docs/13-llm-config-onboarding-design.md) — First-time configuration onboarding

## License

[MIT](LICENSE)