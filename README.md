# ByteSmith

**Universal Desktop Client for AI Coding Agents**

## What is ByteSmith?

ByteSmith is a standalone desktop application focused on OpenCode and Codex App Server. Think of it as ChatGPT desktop, but for coding agents — without needing a full IDE.

It supports OpenCode via local server mode (`opencode serve`) and Codex via `codex app-server`.

Built with [Wails v2](https://wails.io) (Go backend + React/TypeScript frontend). Binary size is ~9.7 MB.

## Features

- [x] Multi-agent support (connect to multiple agents simultaneously)
- [x] Chat interface with markdown rendering
- [x] Tool call visualization with status tracking
- [x] Permission management (approve/deny agent actions)
- [x] File system access (agents can read/write files)
- [x] Terminal integration (agents can run commands)
- [x] Slash command autocomplete
- [x] Session history
- [x] Agent auto-discovery (detects installed agents)
- [x] Dark theme
- [ ] Diff viewer
- [ ] File explorer
- [ ] Agent marketplace/registry
- [ ] Light theme
- [ ] Session persistence (SQLite)

## Supported Agents

| Agent | Command | Website |
|-------|---------|---------|
| OpenCode | `opencode serve` | [opencode.ai](https://opencode.ai) |
| Codex App Server | `codex app-server` | [github.com/openai/codex](https://github.com/openai/codex) |

## Quick Start

### Prerequisites

- Go 1.23+
- Node.js 18+
- Wails v2

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Build

```bash
git clone https://github.com/YOUR_USER/bytesmith.git
cd bytesmith

# Linux (webkit2gtk 4.1)
wails build -tags webkit2_41

# macOS / Linux (webkit2gtk 4.0)
wails build
```

### Run

```bash
./build/bin/bytesmith
```

## Development

```bash
# Live reload (Linux with webkit2gtk 4.1)
wails dev -tags webkit2_41

# Live reload (macOS / Linux with webkit2gtk 4.0)
wails dev
```

The Vite dev server provides hot reload for frontend changes. A dev server also runs on `http://localhost:34115` for browser-based development with access to Go methods.

## Architecture

```
┌──────────────────────────────────────────────────┐
│                   ByteSmith UI                   │
│          React + TypeScript + Tailwind           │
│  ┌──────────┐ ┌───────────┐ ┌─────────────────┐ │
│  │ Chat UI  │ │ Tool Call │ │   Permission    │ │
│  │          │ │  Cards    │ │    Dialogs      │ │
│  └──────────┘ └───────────┘ └─────────────────┘ │
├──────────────────────────────────────────────────┤
│           Wails Bindings + Runtime Events        │
│                   (Go ↔ JS)                      │
├──────────────────────────────────────────────────┤
│                  Go Backend                      │
│  ┌──────────┐ ┌───────────┐ ┌─────────────────┐ │
│  │   ACP    │ │  Agent    │ │    Session      │ │
│  │  Client  │ │  Manager  │ │     Store       │ │
│  ├──────────┤ ├───────────┤ ├─────────────────┤ │
│  │   FS     │ │ Terminal  │ │   Discovery     │ │
│  │ Provider │ │ Provider  │ │                 │ │
│  └──────────┘ └───────────┘ └─────────────────┘ │
├──────────────────────────────────────────────────┤
│  OpenCode HTTP/SSE runtime + Codex stdio runtime │
├──────────────────────────────────────────────────┤
│      opencode serve + codex app-server process   │
└──────────────────────────────────────────────────┘
```

- **Go backend**: OpenCode server runtime + Codex app-server runtime, agent process manager, file system provider, terminal provider, session store, agent discovery.
- **React frontend**: Chat UI, tool call cards with status tracking, permission dialogs, agent picker, slash command autocomplete.
- **Communication**: Wails bindings for request/response calls (Go ↔ JS) and Wails runtime events for streaming data from agents to the UI.

## Project Structure

```
bytesmith/
├── main.go                    # Thin entrypoint
├── bootstrap.go               # Wails options/bootstrap
├── assets.go                  # Embedded frontend assets
├── app_lifecycle.go           # Wails adapter (bind + lifecycle forwarding)
├── wails.json                 # Wails project config
├── internal/
│   ├── backend/
│   │   └── app_*.go           # Backend app logic split by domain
│   ├── acp/
│   │   ├── client.go          # ACP JSON-RPC client
│   │   ├── transport_stdio.go # stdio transport layer
│   │   ├── methods.go         # ACP method constants
│   │   └── types_*.go         # ACP protocol/domain types
│   ├── agent/
│   │   ├── config.go          # Agent configuration
│   │   ├── discovery.go       # Auto-discover installed agents
│   │   └── manager.go         # Agent process lifecycle
│   ├── config/                # App configuration
│   ├── fs/
│   │   └── provider.go        # File system operations
│   ├── session/
│   │   └── store.go           # Session history
│   └── terminal/
│       └── provider.go        # Terminal/command execution
├── frontend/
│   ├── src/
│   │   ├── App.tsx            # Root component
│   │   ├── main.tsx           # React entrypoint
│   │   ├── components/        # UI components
│   │   ├── hooks/             # React hooks
│   │   ├── lib/               # Utilities
│   │   ├── stores/            # State management
│   │   └── types.ts           # TypeScript types
│   ├── wailsjs/               # Auto-generated Wails bindings
│   ├── package.json
│   ├── tailwind.config.js
│   ├── postcss.config.js
│   ├── tsconfig.json
│   └── vite.config.ts
└── build/
    ├── appicon.png
    ├── darwin/                 # macOS build assets
    └── windows/                # Windows build assets
```

## Runtime Model

- **OpenCode**: ByteSmith talks to `opencode serve` over HTTP/SSE on localhost.
- **Codex App Server**: ByteSmith talks to `codex app-server` over stdio JSON-RPC.
- **Flow**: The client sends user prompts, streams updates, tracks tool calls, and handles permission decisions.

## Configuration

ByteSmith stores its configuration at `~/.config/bytesmith/config.json`.

```jsonc
{
  "agents": [
    {
      "name": "opencode",
      "command": "opencode",
      "args": [],
      "enabled": true
    },
    {
      "name": "codex-app-server",
      "command": "codex",
      "args": ["app-server"],
      "enabled": true
    }
  ],
  "defaultAgent": "opencode",
  "theme": "dark"
}
```

Agents are also auto-discovered from your `$PATH` — if ByteSmith detects a known agent binary, it will appear in the agent picker automatically.

## Contributing

Contributions are welcome! To get started:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run `wails dev -tags webkit2_41` to test locally
5. Commit your changes (`git commit -m 'feat: add my feature'`)
6. Push to your branch (`git push origin feature/my-feature`)
7. Open a Pull Request

Please follow [Conventional Commits](https://www.conventionalcommits.org/) for commit messages.

## License

Licensed under the [Apache License 2.0](LICENSE).

Copyright 2025 ByteSmith Contributors.
