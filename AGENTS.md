# AGENTS.md

Este arquivo define o setup inicial (`/init`) e os padrões operacionais para contribuir no ByteSmith.

## /init

Execute este checklist no início de cada sessão:

1. Confirme branch e estado do repo:
   - `git branch --show-current`
   - `git status --short`
2. Confirme branch principal do projeto:
   - Atualmente a branch principal é `master` (não `main`).
3. Valide toolchain mínima:
   - `go version`
   - `node -v`
   - `npm -v`
4. Para desenvolvimento desktop Linux:
   - `wails dev -tags webkit2_41`
5. Para validação rápida antes de commit:
   - `go test ./...`
   - `cd frontend && npm run build`

## Decisões consolidadas desta sessão

1. Janela inicial no Linux/Wayland deve abrir **maximizada** (não fullscreen).
2. O ajuste robusto de tamanho é feito no `OnDomReady`:
   - `WindowUnfullscreen`
   - `WindowSetMaxSize(16384, 16384)`
   - `ScreenGetAll` + `WindowSetSize` com tamanho lógico da tela atual/primária
   - `WindowMaximise` com retries curtos
3. Terminal deve ser **embutido dentro do app** (PTY interno), não abrir `ghostty` externo.
4. Atalho `Ctrl/Cmd + J` abre o terminal embutido.
5. Estrutura modular é obrigatória:
   - `main.go` fino
   - bootstrap em `bootstrap.go`
   - lifecycle adapter em `app_lifecycle.go`
   - backend segmentado em `internal/backend/app_*.go`
   - tipos ACP segmentados em `internal/acp/types_*.go`

## Padrões de produto e escopo

1. Manter suporte ativo focado em:
   - `opencode`
   - `codex-app-server`
2. Evitar features legadas não usadas (ex.: launchers externos de terminal).

## Wails bindings e arquivos gerados

Quando métodos Go expostos mudarem:

1. Regenerar bindings Wails (quando CLI disponível).
2. Revisar e commitar artefatos gerados se houver mudança:
   - `frontend/wailsjs/go/main/App.d.ts`
   - `frontend/wailsjs/go/main/App.js`
   - `frontend/wailsjs/go/models.ts`
   - `frontend/package.json.md5` (quando alterado pelo fluxo Wails)

## Busca e exploração local

1. Preferir `mgrep` para busca semântica local.
2. Se `mgrep` estiver indisponível (ex.: quota), usar fallback com `rg`.

