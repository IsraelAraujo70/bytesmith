import { useState, useRef, useEffect, useCallback } from 'react';
import { clsx } from 'clsx';
import { Send, Square, Flame } from 'lucide-react';
import { useAppStore } from '../../stores/appStore';
import { sendPrompt, cancelPrompt } from '../../lib/api';
import type { AvailableCommand } from '../../types';

export function PromptInput() {
  const {
    activeSession,
    loading,
    commands,
    setCommands,
    addMessage,
    setLoading,
  } = useAppStore();

  const [text, setText] = useState('');
  const [showSlash, setShowSlash] = useState(false);
  const [slashFilter, setSlashFilter] = useState('');
  const [selectedIdx, setSelectedIdx] = useState(0);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

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
  }, [activeSession, setCommands]);

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
      role: 'user',
      content,
      timestamp: new Date().toISOString(),
    });

    setLoading(true);

    sendPrompt(
      activeSession.connectionID,
      activeSession.sessionID,
      content
    );
  }, [text, activeSession, loading, addMessage, setLoading]);

  const handleCancel = useCallback(() => {
    if (!activeSession) return;
    cancelPrompt(activeSession.connectionID, activeSession.sessionID);
    setLoading(false);
  }, [activeSession, setLoading]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
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

  const disabled = !activeSession;

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
              ? 'Connect to an agent to start...'
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
