# ByteSmith

**Universal Desktop Client for AI Coding Agents**

## What is ByteSmith?

ByteSmith is a standalone desktop application that connects to any [ACP](https://agentclientprotocol.com)-compatible coding agent. Think of it as ChatGPT desktop, but for coding agents — without needing a full IDE.

It speaks the **Agent Client Protocol (ACP)**, a standard wire protocol (JSON-RPC 2.0 over stdio) that any compliant agent can implement. Install ByteSmith once, connect to whichever agent you prefer — OpenCode, Codex CLI, Gemini CLI, Claude Code, Goose, Kiro, Augment, or your own custom agent.

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

Any agent that implements the [Agent Client Protocol](https://agentclientprotocol.com) works with ByteSmith.

| Agent | Command | Website |
|-------|---------|---------|
| OpenCode | `opencode acp` | [opencode.ai](https://opencode.ai) |
| Codex CLI | `codex-acp` | [github.com/openai/codex](https://github.com/openai/codex) |
| Gemini CLI | `gemini --acp` | [github.com/google-gemini/gemini-cli](https://github.com/google-gemini/gemini-cli) |
| Claude Code | `claude-code-acp` | [github.com/anthropics/claude-code](https://github.com/anthropics/claude-code) |
| Goose | `goose --acp` | [github.com/block/goose](https://github.com/block/goose) |
| Kiro | `kiro --acp` | [kiro.dev](https://kiro.dev) |
| Augment | `augment acp` | [augmentcode.com](https://augmentcode.com) |
| Custom agent | _your binary_ | — |

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
│        stdio (JSON-RPC 2.0) ← ACP Protocol      │
├──────────────────────────────────────────────────┤
│  Agent Process (opencode, codex, gemini, etc.)   │
└──────────────────────────────────────────────────┘
```

- **Go backend**: ACP client (JSON-RPC 2.0 over stdio), agent process manager, file system provider, terminal provider, session store, agent discovery.
- **React frontend**: Chat UI, tool call cards with status tracking, permission dialogs, agent picker, slash command autocomplete.
- **Communication**: Wails bindings for request/response calls (Go ↔ JS) and Wails runtime events for streaming data from agents to the UI.

## Project Structure

```
bytesmith/
├── main.go                    # Wails app entrypoint
├── app.go                     # App struct (Wails bindings)
├── wails.json                 # Wails project config
├── internal/
│   ├── acp/
│   │   ├── client.go          # ACP JSON-RPC client
│   │   ├── transport_stdio.go # stdio transport layer
│   │   └── types.go           # ACP protocol types
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

## How ACP Works

The **Agent Client Protocol (ACP)** is a standard protocol for communication between client applications and AI coding agents. It is to coding agents what LSP is to language servers.

- **Transport**: JSON-RPC 2.0 over stdio (the client spawns the agent as a child process)
- **Flow**: The client sends user messages to the agent, which responds with text, tool calls, or permission requests
- **Tools**: Agents request tool execution (file read/write, shell commands, etc.) and the client decides whether to approve or deny each action
- **Streaming**: Agents stream partial responses as they generate them

Learn more at [agentclientprotocol.com](https://agentclientprotocol.com).

## Configuration

ByteSmith stores its configuration at `~/.config/bytesmith/config.json`.

```jsonc
{
  "agents": [
    {
      "name": "opencode",
      "command": "opencode",
      "args": ["acp"],
      "enabled": true
    },
    {
      "name": "codex",
      "command": "codex-acp",
      "args": [],
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
