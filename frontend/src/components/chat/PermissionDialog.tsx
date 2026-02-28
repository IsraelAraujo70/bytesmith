import { clsx } from 'clsx';
import { Shield, AlertTriangle } from 'lucide-react';
import { useAppStore } from '../../stores/appStore';
import { respondPermission } from '../../lib/api';

export function PermissionDialog() {
  const { permissionRequests, removePermissionRequest, activeSession } =
    useAppStore();

  // Only show the first request for the active session
  const request = permissionRequests.find(
    (r) =>
      activeSession &&
      r.connectionId === activeSession.connectionID &&
      r.sessionId === activeSession.sessionID
  );

  if (!request) return null;

  const handleOption = async (optionId: string) => {
    await respondPermission(request.connectionId, optionId);
    removePermissionRequest(request.toolCallId);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
      <div className="bg-[var(--bg-elevated)] border border-[var(--border)] rounded-xl shadow-elevated max-w-md w-full mx-4 overflow-hidden animate-slide-up">
        {/* Header */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-[var(--border-subtle)]">
          <div className="w-8 h-8 rounded-lg bg-[var(--warning-muted)] flex items-center justify-center">
            <Shield className="w-4 h-4 text-[var(--warning)]" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-[var(--text-primary)]">Permission Required</h3>
            <p className="text-[10px] text-[var(--text-muted)]">
              The agent wants to perform an action
            </p>
          </div>
        </div>

        {/* Body */}
        <div className="px-4 py-3">
          <div className="flex items-start gap-2">
            <AlertTriangle className="w-3.5 h-3.5 text-[var(--warning)] shrink-0 mt-0.5" />
            <div>
              <p className="text-xs font-medium text-[var(--text-primary)]">{request.title}</p>
              <p className="text-[10px] text-[var(--text-muted)] mt-0.5 font-mono">
                {request.kind}
              </p>
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="px-4 py-3 border-t border-[var(--border-subtle)] flex flex-wrap gap-2">
          {request.options.map((opt) => {
            const isAllow = opt.kind.startsWith('allow');
            const isReject = opt.kind.startsWith('reject');

            return (
              <button
                key={opt.optionId}
                onClick={() => handleOption(opt.optionId)}
                className={clsx(
                  'flex-1 min-w-[100px] px-3 py-1.5 rounded-md text-xs font-medium transition-all duration-200',
                  isAllow &&
                    'bg-[var(--success-muted)] text-[var(--success)] hover:bg-[var(--success)] hover:text-white border border-transparent',
                  isReject &&
                    'bg-[var(--error-muted)] text-[var(--error)] hover:bg-[var(--error)] hover:text-white border border-transparent',
                  !isAllow &&
                    !isReject &&
                    'bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:bg-[var(--border)] border border-[var(--border-subtle)]'
                )}
              >
                {opt.name}
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
