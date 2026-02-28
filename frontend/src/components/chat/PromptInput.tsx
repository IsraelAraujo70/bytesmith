import { useState, useRef, useEffect, useCallback } from 'react';
import { clsx } from 'clsx';
import { Send, Square, Loader2 } from 'lucide-react';
import { useAppStore } from '../../stores/appStore';
import { sendPrompt, cancelPrompt, getAvailableCommands } from '../../lib/api';
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

  // Load commands when session changes
  useEffect(() => {
    if (activeSession) {
      getAvailableCommands(
        activeSession.connectionID,
        activeSession.sessionID
      ).then(setCommands);
    }
  }, [activeSession, setCommands]);

  const filteredCommands = commands.filter((cmd) =>
    cmd.name.toLowerCase().includes(slashFilter.toLowerCase())
  );

  const handleTextChange = (value: string) => {
    setText(value);

    // Detect slash command trigger
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

    // Add user message to store
    addMessage({
      role: 'user',
      content,
      timestamp: new Date().toISOString(),
    });

    setLoading(true);

    // Send to backend
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
    <div className="relative border-t border-[var(--border)] bg-[var(--bg-secondary)]">
      {/* Slash command autocomplete */}
      {showSlash && filteredCommands.length > 0 && (
        <div className="absolute bottom-full left-4 right-4 mb-1 bg-[var(--bg-tertiary)] border border-[var(--border)] rounded-lg shadow-xl overflow-hidden max-h-[200px] overflow-y-auto">
          {filteredCommands.map((cmd, i) => (
            <button
              key={cmd.name}
              onClick={() => selectCommand(cmd)}
              className={clsx(
                'w-full flex items-center gap-3 px-3 py-2 text-left transition-colors',
                i === selectedIdx
                  ? 'bg-[var(--accent)] bg-opacity-20'
                  : 'hover:bg-[var(--bg-secondary)]'
              )}
            >
              <span className="text-sm font-mono text-[var(--accent)]">
                {cmd.name}
              </span>
              <span className="text-xs text-[var(--text-secondary)] truncate">
                {cmd.description}
              </span>
              {cmd.hint && (
                <span className="ml-auto text-[10px] text-[var(--text-secondary)] opacity-50">
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
              ? 'Agent is thinking...'
              : 'Type a message... (/ for commands)'
          }
          rows={1}
          className={clsx(
            'flex-1 resize-none bg-[var(--bg-tertiary)] border border-[var(--border)] rounded-xl px-4 py-2.5 text-sm',
            'text-[var(--text-primary)] placeholder:text-[var(--text-secondary)]',
            'focus:outline-none focus:border-[var(--accent)] transition-colors',
            'min-h-[40px] max-h-[192px]',
            disabled && 'opacity-50 cursor-not-allowed'
          )}
        />

        {loading ? (
          <button
            onClick={handleCancel}
            className="shrink-0 w-10 h-10 flex items-center justify-center rounded-xl bg-[var(--error)] bg-opacity-20 text-[var(--error)] hover:bg-opacity-30 transition-colors"
            title="Cancel"
          >
            <Square className="w-4 h-4" />
          </button>
        ) : (
          <button
            onClick={handleSubmit}
            disabled={!text.trim() || disabled}
            className={clsx(
              'shrink-0 w-10 h-10 flex items-center justify-center rounded-xl transition-colors',
              text.trim() && !disabled
                ? 'bg-[var(--accent)] text-white hover:bg-[var(--accent-hover)]'
                : 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)] cursor-not-allowed'
            )}
            title="Send"
          >
            <Send className="w-4 h-4" />
          </button>
        )}
      </div>
    </div>
  );
}
