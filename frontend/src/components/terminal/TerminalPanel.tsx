import { useCallback, useEffect } from 'react';
import { clsx } from 'clsx';
import { Plus, SquareTerminal, X } from 'lucide-react';
import '@xterm/xterm/css/xterm.css';
import { useAppStore } from '../../stores/appStore';
import {
  closeEmbeddedTerminal,
  createEmbeddedTerminal,
} from '../../lib/api';
import { TerminalView } from './TerminalView';

function tabLabel(index: number, shell: string): string {
  const normalized = shell?.trim() || 'shell';
  return `Terminal ${index + 1} (${normalized})`;
}

export function TerminalPanel() {
  const {
    cwd,
    setError,
    terminals,
    terminalPanelOpen,
    setTerminalPanelOpen,
    activeTerminalId,
    setActiveTerminal,
    addTerminal,
    removeTerminal,
  } = useAppStore();

  useEffect(() => {
    if (activeTerminalId || terminals.length === 0) {
      return;
    }
    setActiveTerminal(terminals[terminals.length - 1].id);
  }, [activeTerminalId, setActiveTerminal, terminals]);

  const handleCreateTerminal = useCallback(async () => {
    if (!cwd || !cwd.trim()) {
      setError('Select a working directory before opening a terminal.');
      return;
    }
    try {
      const created = await createEmbeddedTerminal(cwd);
      addTerminal(created);
    } catch (err) {
      const reason = err instanceof Error ? err.message : String(err);
      setError(`Failed to create terminal: ${reason}`);
    }
  }, [addTerminal, cwd, setError]);

  const handleCloseTab = useCallback(
    async (terminalID: string) => {
      try {
        await closeEmbeddedTerminal(terminalID);
      } catch (err) {
        const reason = err instanceof Error ? err.message : String(err);
        setError(`Failed to close terminal: ${reason}`);
      }
      removeTerminal(terminalID);
    },
    [removeTerminal, setError]
  );

  if (!terminalPanelOpen) {
    return null;
  }

  return (
    <section className="flex h-72 min-h-[180px] flex-col border-t border-[var(--border-subtle)] bg-[var(--bg-secondary)]">
      <header className="flex h-8 shrink-0 items-center justify-between border-b border-[var(--border-subtle)] px-2">
        <div className="flex min-w-0 items-center gap-1 overflow-x-auto pr-2">
          {terminals.map((terminal, index) => {
            const active = terminal.id === activeTerminalId;
            return (
              <button
                key={terminal.id}
                onClick={() => setActiveTerminal(terminal.id)}
                className={clsx(
                  'group flex h-6 items-center gap-1.5 rounded border px-2 text-[10px] transition-colors',
                  active
                    ? 'border-[var(--accent)] bg-[var(--accent-muted)] text-[var(--accent)]'
                    : 'border-[var(--border-subtle)] text-[var(--text-muted)] hover:border-[var(--border)] hover:text-[var(--text-secondary)]'
                )}
              >
                <SquareTerminal className="h-3 w-3 shrink-0" />
                <span className="max-w-[220px] truncate">
                  {tabLabel(index, terminal.shell)}
                </span>
                {terminal.exited && (
                  <span className="text-[9px] text-[var(--warning)]">
                    exit {terminal.exitCode ?? 0}
                  </span>
                )}
                <span
                  role="button"
                  tabIndex={0}
                  onClick={(event) => {
                    event.stopPropagation();
                    void handleCloseTab(terminal.id);
                  }}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                      event.preventDefault();
                      event.stopPropagation();
                      void handleCloseTab(terminal.id);
                    }
                  }}
                  className="hidden rounded p-0.5 group-hover:inline-flex hover:bg-[var(--bg-tertiary)]"
                  aria-label="Close terminal tab"
                >
                  <X className="h-3 w-3" />
                </span>
              </button>
            );
          })}
        </div>

        <div className="flex items-center gap-1">
          <button
            onClick={() => {
              void handleCreateTerminal();
            }}
            className="flex h-6 items-center gap-1 rounded border border-[var(--border-subtle)] px-2 text-[10px] text-[var(--text-muted)] transition-colors hover:border-[var(--accent)] hover:text-[var(--accent)]"
            title="New terminal"
          >
            <Plus className="h-3 w-3" />
            <span>New</span>
          </button>

          <button
            onClick={() => setTerminalPanelOpen(false)}
            className="flex h-6 w-6 items-center justify-center rounded border border-[var(--border-subtle)] text-[var(--text-muted)] transition-colors hover:border-[var(--accent)] hover:text-[var(--accent)]"
            title="Hide terminal panel"
          >
            <X className="h-3 w-3" />
          </button>
        </div>
      </header>

      <div className="relative min-h-0 flex-1 bg-[var(--bg-primary)]">
        {terminals.length === 0 && (
          <div className="flex h-full items-center justify-center text-xs text-[var(--text-muted)]">
            No terminal tabs open.
          </div>
        )}

        {terminals.map((terminal) => (
          <TerminalView
            key={terminal.id}
            terminal={terminal}
            active={terminal.id === activeTerminalId}
          />
        ))}
      </div>
    </section>
  );
}
