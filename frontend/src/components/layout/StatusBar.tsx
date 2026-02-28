import { useState } from 'react';
import { Folder, Wifi, WifiOff, Palette } from 'lucide-react';
import { clsx } from 'clsx';
import { useAppStore } from '../../stores/appStore';
import { setSessionModel } from '../../lib/api';
import { themes } from '../../lib/themes';

export function StatusBar() {
  const {
    activeSession,
    connections,
    cwd,
    models,
    currentModelId,
    setSessionModels,
    setError,
    themeId,
    setTheme,
  } = useAppStore();
  const [updatingModel, setUpdatingModel] = useState(false);
  const [showThemePicker, setShowThemePicker] = useState(false);

  const activeConn = activeSession
    ? connections.find((c) => c.id === activeSession.connectionID)
    : null;

  const handleModelChange = async (modelId: string) => {
    if (!activeSession) return;

    setUpdatingModel(true);
    try {
      await setSessionModel(
        activeSession.connectionID,
        activeSession.sessionID,
        modelId
      );
      setSessionModels(models, modelId);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(`Falha ao trocar modelo: ${message}`);
    } finally {
      setUpdatingModel(false);
    }
  };

  const themeList = Object.values(themes);

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
        {activeSession && models.length > 0 && (
          <select
            value={currentModelId || models[0].modelId}
            onChange={(e) => handleModelChange(e.target.value)}
            disabled={updatingModel}
            className="h-4 min-w-[140px] bg-[var(--bg-tertiary)] border border-[var(--border-subtle)] rounded px-1 text-[10px] text-[var(--text-secondary)] disabled:opacity-50 focus:outline-none focus:border-[var(--accent)]"
            title="Model"
          >
            {models.map((model) => (
              <option key={model.modelId} value={model.modelId}>
                {model.name}
              </option>
            ))}
          </select>
        )}

        {activeSession && (
          <span className="opacity-50 font-mono">
            {activeSession.sessionID.slice(0, 8)}
          </span>
        )}

        {/* Theme picker */}
        <button
          onClick={() => setShowThemePicker(!showThemePicker)}
          className="flex items-center gap-1 px-1 py-0.5 rounded hover:bg-[var(--bg-tertiary)] transition-colors"
          title="Change theme"
        >
          <Palette className="w-2.5 h-2.5" />
        </button>

        {showThemePicker && (
          <div className="absolute bottom-full right-2 mb-1 bg-[var(--bg-elevated)] border border-[var(--border)] rounded-lg shadow-elevated overflow-hidden animate-fade-in z-50">
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
