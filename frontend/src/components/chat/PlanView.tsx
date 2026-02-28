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
  low: 'text-[var(--text-secondary)]',
};

export function PlanView({ entries }: PlanViewProps) {
  const [collapsed, setCollapsed] = useState(false);

  if (entries.length === 0) return null;

  const completedCount = entries.filter(
    (e) => e.status === 'completed'
  ).length;

  return (
    <div className="mx-4 my-2">
      <div className="bg-[var(--bg-secondary)] border border-[var(--border)] rounded-lg overflow-hidden">
        {/* Header */}
        <button
          onClick={() => setCollapsed(!collapsed)}
          className="w-full flex items-center gap-2 px-3 py-2 hover:bg-[var(--bg-tertiary)] transition-colors"
        >
          <ListTodo className="w-4 h-4 text-[var(--accent)]" />
          <span className="text-xs font-medium flex-1 text-left">
            Plan
          </span>
          <span className="text-[10px] text-[var(--text-secondary)]">
            {completedCount}/{entries.length}
          </span>
          {collapsed ? (
            <ChevronRight className="w-3.5 h-3.5 text-[var(--text-secondary)]" />
          ) : (
            <ChevronDown className="w-3.5 h-3.5 text-[var(--text-secondary)]" />
          )}
        </button>

        {/* Entries */}
        {!collapsed && (
          <div className="border-t border-[var(--border)] px-3 py-2 space-y-1.5">
            {entries.map((entry, i) => {
              const StatusIcon =
                statusIcons[entry.status] || statusIcons.pending;

              return (
                <div key={i} className="flex items-start gap-2">
                  <StatusIcon
                    className={clsx(
                      'w-4 h-4 shrink-0 mt-0.5',
                      entry.status === 'completed'
                        ? 'text-[var(--success)]'
                        : entry.status === 'in_progress'
                        ? 'text-[var(--accent)]'
                        : 'text-[var(--text-secondary)]'
                    )}
                  />
                  <span
                    className={clsx(
                      'text-xs flex-1',
                      entry.status === 'completed'
                        ? 'line-through text-[var(--text-secondary)]'
                        : 'text-[var(--text-primary)]'
                    )}
                  >
                    {entry.content}
                  </span>
                  {entry.priority && entry.priority !== 'normal' && (
                    <span
                      className={clsx(
                        'text-[10px] px-1.5 py-0.5 rounded',
                        priorityColors[entry.priority] ||
                          'text-[var(--text-secondary)]'
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
