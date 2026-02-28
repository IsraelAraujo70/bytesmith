import { Folder, Wifi, WifiOff } from 'lucide-react';
import { useAppStore } from '../../stores/appStore';

export function StatusBar() {
  const { activeSession, connections, cwd } = useAppStore();

  const activeConn = activeSession
    ? connections.find((c) => c.id === activeSession.connectionID)
    : null;

  return (
    <div className="flex items-center justify-between h-7 px-3 bg-[var(--bg-secondary)] border-t border-[var(--border)] text-[11px] text-[var(--text-secondary)] select-none">
      {/* Left */}
      <div className="flex items-center gap-3">
        {activeConn ? (
          <div className="flex items-center gap-1.5">
            <Wifi className="w-3 h-3 text-[var(--success)]" />
            <span>{activeConn.displayName}</span>
          </div>
        ) : (
          <div className="flex items-center gap-1.5">
            <WifiOff className="w-3 h-3 text-[var(--text-secondary)]" />
            <span>Not connected</span>
          </div>
        )}
      </div>

      {/* Center */}
      {cwd && (
        <div className="flex items-center gap-1.5">
          <Folder className="w-3 h-3" />
          <span className="truncate max-w-[300px]">{cwd}</span>
        </div>
      )}

      {/* Right */}
      <div className="flex items-center gap-1.5">
        {activeSession && (
          <span className="opacity-60">
            Session: {activeSession.sessionID.slice(0, 8)}...
          </span>
        )}
      </div>
    </div>
  );
}
