import { useState, useRef, useEffect, useCallback } from 'react';
import { clsx } from 'clsx';
import { Send, Square, Flame, Cpu, SlidersHorizontal, ChevronDown, Shield } from 'lucide-react';
import { useAppStore } from '../../stores/appStore';
import { sendPrompt, cancelPrompt, setSessionAccessMode, setSessionMode } from '../../lib/api';
import type { AvailableCommand } from '../../types';

export function PromptInput() {
  const {
    activeSession,
    commands,
    setCommands,
    addMessage,
    setSessionLoading,
    isSessionLoading,
    sessionReadOnly,
    models,
    currentModelId,
    modes,
    currentModeId,
    setSessionModes,
    accessModes,
    currentAccessModeId,
    setSessionAccessModes,
    setError,
    setModelPickerOpen,
  } = useAppStore();

  const [text, setText] = useState('');
  const [showSlash, setShowSlash] = useState(false);
  const [slashFilter, setSlashFilter] = useState('');
  const [selectedIdx, setSelectedIdx] = useState(0);
  const [modeMenuOpen, setModeMenuOpen] = useState(false);
  const [accessMenuOpen, setAccessMenuOpen] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const modeButtonRef = useRef<HTMLButtonElement>(null);
  const modeMenuRef = useRef<HTMLDivElement>(null);
  const accessButtonRef = useRef<HTMLButtonElement>(null);
  const accessMenuRef = useRef<HTMLDivElement>(null);

  // Auto-resize textarea
  useEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = 'auto';
    const maxH = 8 * 24; // ~8 lines
    el.style.height = `${Math.min(el.scrollHeight, maxH)}px`;
  }, [text]);

  // Reset command list when changing session
  useEffect(() => {
    setCommands([]);
    setModeMenuOpen(false);
    setAccessMenuOpen(false);
  }, [activeSession, setCommands]);

  useEffect(() => {
    if (!modeMenuOpen && !accessMenuOpen) return;
    const handler = (e: MouseEvent) => {
      const target = e.target as Node;
      const outsideMode =
        (!modeMenuRef.current || !modeMenuRef.current.contains(target)) &&
        (!modeButtonRef.current || !modeButtonRef.current.contains(target));
      const outsideAccess =
        (!accessMenuRef.current || !accessMenuRef.current.contains(target)) &&
        (!accessButtonRef.current || !accessButtonRef.current.contains(target));

      if (outsideMode && outsideAccess) {
        setModeMenuOpen(false);
        setAccessMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [modeMenuOpen, accessMenuOpen]);

  const loading = activeSession
    ? isSessionLoading(activeSession.connectionID, activeSession.sessionID)
    : false;

  const filteredCommands = commands.filter((cmd) =>
    cmd.name.toLowerCase().includes(slashFilter.toLowerCase())
  );

  const handleTextChange = (value: string) => {
    setText(value);

    if (value.startsWith('/')) {
      setShowSlash(true);
      setSlashFilter(value);
      setSelectedIdx(0);
    } else {
      setShowSlash(false);
    }
  };

  const handleSubmit = useCallback(() => {
    if (!text.trim() || !activeSession || loading) return;

    const content = text.trim();
    setText('');
    setShowSlash(false);

    addMessage({
      id: crypto.randomUUID(),
      role: 'user',
      content,
      timestamp: new Date().toISOString(),
    });

    setSessionLoading(activeSession.connectionID, activeSession.sessionID, true);

    sendPrompt(
      activeSession.connectionID,
      activeSession.sessionID,
      content
    );
  }, [text, activeSession, loading, addMessage, setSessionLoading]);

  const handleCancel = useCallback(() => {
    if (!activeSession) return;
    cancelPrompt(activeSession.connectionID, activeSession.sessionID);
    setSessionLoading(activeSession.connectionID, activeSession.sessionID, false);
  }, [activeSession, setSessionLoading]);

  const handleModeChange = useCallback(async (modeId: string) => {
    if (!activeSession) return;
    if (modeId === currentModeId) {
      setModeMenuOpen(false);
      return;
    }

    try {
      await setSessionMode(activeSession.connectionID, activeSession.sessionID, modeId);
      setSessionModes(modes, modeId);
      setModeMenuOpen(false);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(`Failed to switch mode: ${message}`);
    }
  }, [activeSession, currentModeId, modes, setSessionModes, setError]);

  const handleAccessModeChange = useCallback(async (modeId: string) => {
    if (!activeSession) return;
    if (modeId === currentAccessModeId) {
      setAccessMenuOpen(false);
      return;
    }

    try {
      await setSessionAccessMode(activeSession.connectionID, activeSession.sessionID, modeId);
      setSessionAccessModes(accessModes, modeId);
      setAccessMenuOpen(false);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(`Failed to switch access mode: ${message}`);
    }
  }, [activeSession, currentAccessModeId, accessModes, setSessionAccessModes, setError]);

  const cycleModesBackward = useCallback(() => {
    if (!activeSession || modes.length < 2) return;

    const currentIndex = modes.findIndex((mode) => mode.modeId === currentModeId);
    const startIndex = currentIndex >= 0 ? currentIndex : 0;
    const nextIndex = (startIndex - 1 + modes.length) % modes.length;
    const nextMode = modes[nextIndex];
    if (nextMode) {
      void handleModeChange(nextMode.modeId);
    }
  }, [activeSession, modes, currentModeId, handleModeChange]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!showSlash && e.key === 'Tab' && e.shiftKey && modes.length > 1) {
      e.preventDefault();
      cycleModesBackward();
      return;
    }

    // Slash command navigation
    if (showSlash && filteredCommands.length > 0) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setSelectedIdx((i) =>
          i < filteredCommands.length - 1 ? i + 1 : 0
        );
        return;
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        setSelectedIdx((i) =>
          i > 0 ? i - 1 : filteredCommands.length - 1
        );
        return;
      }
      if (e.key === 'Tab' || (e.key === 'Enter' && !e.shiftKey)) {
        e.preventDefault();
        const cmd = filteredCommands[selectedIdx];
        if (cmd) {
          setText(cmd.name + ' ');
          setShowSlash(false);
        }
        return;
      }
      if (e.key === 'Escape') {
        setShowSlash(false);
        return;
      }
    }

    // Submit on Enter (Shift+Enter for newline)
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const selectCommand = (cmd: AvailableCommand) => {
    setText(cmd.name + ' ');
    setShowSlash(false);
    textareaRef.current?.focus();
  };

  const disabled = !activeSession || sessionReadOnly;
  const currentModel = models.find((m) => m.modelId === currentModelId);
  const currentMode = modes.find((m) => m.modeId === currentModeId) || modes[0];
  const currentAccessMode =
    accessModes.find((m) => m.modeId === currentAccessModeId) || accessModes[0];

  return (
    <div className="relative border-t border-[var(--border-subtle)] bg-[var(--bg-secondary)]">
      {/* Slash command autocomplete */}
      {showSlash && filteredCommands.length > 0 && (
        <div className="absolute bottom-full left-4 right-4 mb-1 bg-[var(--bg-elevated)] border border-[var(--border)] rounded-md shadow-elevated overflow-hidden max-h-[200px] overflow-y-auto animate-fade-in">
          {filteredCommands.map((cmd, i) => (
            <button
              key={cmd.name}
              onClick={() => selectCommand(cmd)}
              className={clsx(
                'w-full flex items-center gap-2.5 px-3 py-1.5 text-left transition-colors',
                i === selectedIdx
                  ? 'bg-[var(--accent-muted)] text-[var(--accent)]'
                  : 'hover:bg-[var(--bg-tertiary)] text-[var(--text-primary)]'
              )}
            >
              <span className="text-xs font-mono text-[var(--accent)]">
                {cmd.name}
              </span>
              <span className="text-[11px] text-[var(--text-muted)] truncate">
                {cmd.description}
              </span>
              {cmd.hint && (
                <span className="ml-auto text-[9px] text-[var(--text-muted)] opacity-40 font-mono">
                  {cmd.hint}
                </span>
              )}
            </button>
          ))}
        </div>
      )}

      {/* Session chips (model + mode) */}
      {activeSession && (currentModel || currentMode || currentAccessMode) && (
        <div className="flex items-center gap-2 px-4 pt-2 pb-0">
          {currentModel && (
            <button
              onClick={() => setModelPickerOpen(true)}
              className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-[var(--bg-tertiary)] border border-[var(--border-subtle)] hover:border-[var(--accent)] text-[10px] text-[var(--text-muted)] hover:text-[var(--accent)] transition-all duration-150"
              title="Change model (Ctrl+K)"
            >
              <Cpu className="w-2.5 h-2.5" />
              <span className="font-mono">{currentModel.name}</span>
            </button>
          )}

          {currentMode && (
            <div className="relative">
              <button
                ref={modeButtonRef}
                onClick={() => {
                  setModeMenuOpen((open) => !open);
                  setAccessMenuOpen(false);
                }}
                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-[var(--bg-tertiary)] border border-[var(--border-subtle)] hover:border-[var(--accent)] text-[10px] text-[var(--text-muted)] hover:text-[var(--accent)] transition-all duration-150"
                title="Change agent mode (Shift+Tab to cycle)"
              >
                <SlidersHorizontal className="w-2.5 h-2.5" />
                <span className="font-mono">{currentMode.name}</span>
                <ChevronDown className="w-2.5 h-2.5 opacity-60" />
              </button>

              {modeMenuOpen && modes.length > 0 && (
                <div
                  ref={modeMenuRef}
                  className="absolute left-0 top-full mt-1 min-w-[180px] bg-[var(--bg-elevated)] border border-[var(--border)] rounded-md shadow-elevated overflow-hidden z-20 animate-fade-in"
                >
                  {modes.map((mode) => (
                    <button
                      key={mode.modeId}
                      onClick={() => {
                        void handleModeChange(mode.modeId);
                      }}
                      className={clsx(
                        'w-full flex items-center px-2.5 py-1.5 text-left text-[11px] transition-colors',
                        mode.modeId === currentModeId
                          ? 'bg-[var(--accent-muted)] text-[var(--accent)]'
                          : 'hover:bg-[var(--bg-tertiary)] text-[var(--text-primary)]'
                      )}
                    >
                      <span className="truncate">{mode.name}</span>
                      <span className="ml-auto text-[9px] opacity-50 font-mono">
                        {mode.modeId}
                      </span>
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}

          {currentAccessMode && (
            <div className="relative">
              <button
                ref={accessButtonRef}
                onClick={() => {
                  setAccessMenuOpen((open) => !open);
                  setModeMenuOpen(false);
                }}
                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-[var(--bg-tertiary)] border border-[var(--border-subtle)] hover:border-[var(--accent)] text-[10px] text-[var(--text-muted)] hover:text-[var(--accent)] transition-all duration-150"
                title="Change access mode"
              >
                <Shield className="w-2.5 h-2.5" />
                <span className="font-mono">{currentAccessMode.name}</span>
                <ChevronDown className="w-2.5 h-2.5 opacity-60" />
              </button>

              {accessMenuOpen && accessModes.length > 0 && (
                <div
                  ref={accessMenuRef}
                  className="absolute left-0 top-full mt-1 min-w-[180px] bg-[var(--bg-elevated)] border border-[var(--border)] rounded-md shadow-elevated overflow-hidden z-20 animate-fade-in"
                >
                  {accessModes.map((mode) => (
                    <button
                      key={mode.modeId}
                      onClick={() => {
                        void handleAccessModeChange(mode.modeId);
                      }}
                      className={clsx(
                        'w-full flex items-center px-2.5 py-1.5 text-left text-[11px] transition-colors',
                        mode.modeId === currentAccessModeId
                          ? 'bg-[var(--accent-muted)] text-[var(--accent)]'
                          : 'hover:bg-[var(--bg-tertiary)] text-[var(--text-primary)]'
                      )}
                    >
                      <span className="truncate">{mode.name}</span>
                      <span className="ml-auto text-[9px] opacity-50 font-mono">
                        {mode.modeId}
                      </span>
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Input area */}
      <div className="flex items-end gap-2 p-3">
        <textarea
          ref={textareaRef}
          value={text}
          onChange={(e) => handleTextChange(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={disabled}
          placeholder={
            disabled
              ? sessionReadOnly
                ? 'Session in read-only mode (resume failed)...'
                : 'Connect to an agent to start...'
              : loading
              ? 'Forging response...'
              : 'Type a message... (/ for commands)'
          }
          rows={1}
          className={clsx(
            'flex-1 resize-none bg-[var(--bg-tertiary)] border border-[var(--border-subtle)] rounded-lg px-3.5 py-2 text-sm',
            'text-[var(--text-primary)] placeholder:text-[var(--text-muted)]',
            'focus:outline-none focus:border-[var(--accent)] focus:shadow-glow-sm transition-all duration-200',
            'min-h-[38px] max-h-[192px]',
            disabled && 'opacity-40 cursor-not-allowed'
          )}
        />

        {loading ? (
          <button
            onClick={handleCancel}
            className="shrink-0 w-9 h-9 flex items-center justify-center rounded-lg bg-[var(--error-muted)] text-[var(--error)] hover:bg-[var(--error)] hover:text-white transition-all duration-200"
            title="Cancel"
          >
            <Square className="w-3.5 h-3.5" />
          </button>
        ) : (
          <button
            onClick={handleSubmit}
            disabled={!text.trim() || disabled}
            className={clsx(
              'shrink-0 w-9 h-9 flex items-center justify-center rounded-lg transition-all duration-200',
              text.trim() && !disabled
                ? 'bg-[var(--accent)] text-white hover:bg-[var(--accent-hover)] shadow-glow-sm hover:shadow-glow'
                : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)] cursor-not-allowed'
            )}
            title="Send"
          >
            {text.trim() && !disabled ? (
              <Flame className="w-3.5 h-3.5" />
            ) : (
              <Send className="w-3.5 h-3.5" />
            )}
          </button>
        )}
      </div>
    </div>
  );
}
