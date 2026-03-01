import { useEffect, useRef, useState } from 'react';
import { Cpu, Folder, WifiOff, Palette, ChevronUp } from 'lucide-react';
import { clsx } from 'clsx';
import { useAppStore } from '../../stores/appStore';
import { themes } from '../../lib/themes';

export function StatusBar() {
  const {
    activeSession,
    connections,
    cwd,
    models,
    currentModelId,
    themeId,
    setTheme,
    setModelPickerOpen,
  } = useAppStore();
  const [showThemePicker, setShowThemePicker] = useState(false);
  const themePickerRef = useRef<HTMLDivElement>(null);
  const themeBtnRef = useRef<HTMLButtonElement>(null);

  const activeConn = activeSession
    ? connections.find((c) => c.id === activeSession.connectionID)
    : null;

  const currentModel = models.find((m) => m.modelId === currentModelId);
  const themeList = Object.values(themes);

  // Close theme picker on click outside
  useEffect(() => {
    if (!showThemePicker) return;
    const handler = (e: MouseEvent) => {
      const target = e.target as Node;
      if (
        themePickerRef.current &&
        !themePickerRef.current.contains(target) &&
        themeBtnRef.current &&
        !themeBtnRef.current.contains(target)
      ) {
        setShowThemePicker(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [showThemePicker]);

  return (
    <div className="relative flex items-center justify-between h-6 px-3 bg-[var(--bg-secondary)] border-t border-[var(--border-subtle)] text-[10px] text-[var(--text-muted)] select-none">
      {/* Left */}
      <div className="flex items-center gap-3">
        {activeConn ? (
          <div className="flex items-center gap-1">
            <div className="w-1.5 h-1.5 rounded-full bg-[var(--success)] animate-pulse-ember" />
            <span className="text-[var(--text-secondary)]">{activeConn.displayName}</span>
          </div>
        ) : (
          <div className="flex items-center gap-1">
            <WifiOff className="w-2.5 h-2.5" />
            <span>Disconnected</span>
          </div>
        )}
      </div>

      {/* Center */}
      {cwd && (
        <div className="flex items-center gap-1 text-[var(--text-muted)]">
          <Folder className="w-2.5 h-2.5" />
          <span className="truncate max-w-[250px] font-mono">{cwd}</span>
        </div>
      )}

      {/* Right */}
      <div className="flex items-center gap-2">
        {/* Model picker trigger */}
        {activeSession && models.length > 0 && (
          <button
            onClick={() => setModelPickerOpen(true)}
            className={clsx(
              'flex items-center gap-1.5 h-4 px-2 rounded',
              'bg-[var(--bg-tertiary)] border border-[var(--border-subtle)]',
              'hover:border-[var(--accent)] hover:text-[var(--accent)]',
              'transition-all duration-150 group'
            )}
            title="Change model (Ctrl+K)"
          >
            <Cpu className="w-2.5 h-2.5 text-[var(--text-muted)] group-hover:text-[var(--accent)] transition-colors" />
            <span className="text-[10px] text-[var(--text-secondary)] group-hover:text-[var(--accent)] transition-colors max-w-[160px] truncate">
              {currentModel?.name || currentModelId || 'Select model'}
            </span>
            <kbd className="hidden sm:inline ml-1 px-1 py-px rounded bg-[var(--bg-primary)] border border-[var(--border-subtle)] text-[8px] text-[var(--text-muted)] font-mono group-hover:border-[var(--accent)] group-hover:text-[var(--accent)] transition-colors">
              ⌘K
            </kbd>
            <ChevronUp className="w-2 h-2 text-[var(--text-muted)] group-hover:text-[var(--accent)] transition-colors sm:hidden" />
          </button>
        )}

        {activeSession && (
          <span className="opacity-50 font-mono">
            {activeSession.sessionID.slice(0, 8)}
          </span>
        )}

        {/* Theme picker */}
        <button
          ref={themeBtnRef}
          onClick={() => setShowThemePicker(!showThemePicker)}
          className="flex items-center gap-1 px-1 py-0.5 rounded hover:bg-[var(--bg-tertiary)] transition-colors"
          title="Change theme"
        >
          <Palette className="w-2.5 h-2.5" />
        </button>

        {showThemePicker && (
          <div
            ref={themePickerRef}
            className="absolute bottom-full right-2 mb-1 bg-[var(--bg-elevated)] border border-[var(--border)] rounded-lg shadow-elevated overflow-hidden animate-fade-in z-50"
          >
            {themeList.map((t) => (
              <button
                key={t.id}
                onClick={() => {
                  setTheme(t.id);
                  setShowThemePicker(false);
                }}
                className={clsx(
                  'flex items-center gap-2 px-3 py-1.5 text-[11px] w-full text-left transition-colors',
                  t.id === themeId
                    ? 'bg-[var(--accent-muted)] text-[var(--accent)]'
                    : 'hover:bg-[var(--bg-tertiary)] text-[var(--text-secondary)]'
                )}
              >
                <div
                  className="w-3 h-3 rounded-full border border-[var(--border)]"
                  style={{ background: t.colors.accent }}
                />
                <span>{t.name}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
