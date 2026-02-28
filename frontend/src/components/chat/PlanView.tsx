import { useState } from 'react';
import { clsx } from 'clsx';
import {
  ChevronDown,
  ChevronRight,
  CheckCircle2,
  Circle,
  Clock,
  ListTodo,
} from 'lucide-react';
import type { PlanEntry } from '../../types';

interface PlanViewProps {
  entries: PlanEntry[];
}

const statusIcons: Record<string, React.ElementType> = {
  completed: CheckCircle2,
  in_progress: Clock,
  pending: Circle,
};

const priorityColors: Record<string, string> = {
  high: 'text-[var(--error)]',
  medium: 'text-[var(--warning)]',
  low: 'text-[var(--text-muted)]',
};

export function PlanView({ entries }: PlanViewProps) {
  const [collapsed, setCollapsed] = useState(false);

  if (entries.length === 0) return null;

  const completedCount = entries.filter(
    (e) => e.status === 'completed'
  ).length;
  const progress = entries.length > 0 ? (completedCount / entries.length) * 100 : 0;

  return (
    <div className="mx-5 my-1.5 animate-fade-in">
      <div className="bg-[var(--bg-secondary)] border border-[var(--border-subtle)] rounded-md overflow-hidden">
        {/* Header */}
        <button
          onClick={() => setCollapsed(!collapsed)}
          className="w-full flex items-center gap-2 px-2.5 py-1.5 hover:bg-[var(--bg-tertiary)] transition-colors"
        >
          <ListTodo className="w-3.5 h-3.5 text-[var(--accent)]" />
          <span className="text-[11px] font-medium flex-1 text-left text-[var(--text-secondary)]">
            Plan
          </span>

          {/* Progress bar */}
          <div className="w-16 h-1 bg-[var(--bg-tertiary)] rounded-full overflow-hidden">
            <div
              className="h-full bg-[var(--accent)] rounded-full transition-all duration-500"
              style={{ width: `${progress}%` }}
            />
          </div>

          <span className="text-[9px] text-[var(--text-muted)] font-mono">
            {completedCount}/{entries.length}
          </span>
          {collapsed ? (
            <ChevronRight className="w-3 h-3 text-[var(--text-muted)]" />
          ) : (
            <ChevronDown className="w-3 h-3 text-[var(--text-muted)]" />
          )}
        </button>

        {/* Entries */}
        {!collapsed && (
          <div className="border-t border-[var(--border-subtle)] px-2.5 py-1.5 space-y-1">
            {entries.map((entry, i) => {
              const StatusIcon =
                statusIcons[entry.status] || statusIcons.pending;

              return (
                <div key={i} className="flex items-start gap-2 py-0.5">
                  <StatusIcon
                    className={clsx(
                      'w-3.5 h-3.5 shrink-0 mt-0.5',
                      entry.status === 'completed'
                        ? 'text-[var(--success)]'
                        : entry.status === 'in_progress'
                        ? 'text-[var(--accent)]'
                        : 'text-[var(--text-muted)]'
                    )}
                  />
                  <span
                    className={clsx(
                      'text-[11px] flex-1 leading-relaxed',
                      entry.status === 'completed'
                        ? 'line-through text-[var(--text-muted)]'
                        : entry.status === 'in_progress'
                        ? 'text-[var(--text-primary)]'
                        : 'text-[var(--text-secondary)]'
                    )}
                  >
                    {entry.content}
                  </span>
                  {entry.priority && entry.priority !== 'normal' && (
                    <span
                      className={clsx(
                        'text-[9px] px-1 py-0.5 rounded font-mono',
                        priorityColors[entry.priority] ||
                          'text-[var(--text-muted)]'
                      )}
                    >
                      {entry.priority}
                    </span>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
