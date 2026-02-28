import { clsx } from 'clsx';
import { Shield, AlertCircle } from 'lucide-react';
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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-60">
      <div className="bg-[var(--bg-secondary)] border border-[var(--border)] rounded-2xl shadow-2xl max-w-md w-full mx-4 overflow-hidden">
        {/* Header */}
        <div className="flex items-center gap-3 px-5 py-4 border-b border-[var(--border)]">
          <div className="w-10 h-10 rounded-xl bg-[var(--warning)] bg-opacity-20 flex items-center justify-center">
            <Shield className="w-5 h-5 text-[var(--warning)]" />
          </div>
          <div>
            <h3 className="text-sm font-semibold">Permission Required</h3>
            <p className="text-xs text-[var(--text-secondary)]">
              The agent wants to perform an action
            </p>
          </div>
        </div>

        {/* Body */}
        <div className="px-5 py-4">
          <div className="flex items-start gap-2 mb-3">
            <AlertCircle className="w-4 h-4 text-[var(--text-secondary)] shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium">{request.title}</p>
              <p className="text-xs text-[var(--text-secondary)] mt-0.5">
                Kind: {request.kind}
              </p>
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="px-5 py-4 border-t border-[var(--border)] flex flex-wrap gap-2">
          {request.options.map((opt) => {
            const isAllow = opt.kind.startsWith('allow');
            const isReject = opt.kind.startsWith('reject');

            return (
              <button
                key={opt.optionId}
                onClick={() => handleOption(opt.optionId)}
                className={clsx(
                  'flex-1 min-w-[120px] px-4 py-2 rounded-lg text-sm font-medium transition-colors',
                  isAllow &&
                    'bg-[var(--success)] bg-opacity-20 text-[var(--success)] hover:bg-opacity-30 border border-[var(--success)] border-opacity-30',
                  isReject &&
                    'bg-[var(--error)] bg-opacity-20 text-[var(--error)] hover:bg-opacity-30 border border-[var(--error)] border-opacity-30',
                  !isAllow &&
                    !isReject &&
                    'bg-[var(--bg-tertiary)] text-[var(--text-primary)] hover:bg-[var(--border)] border border-[var(--border)]'
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
