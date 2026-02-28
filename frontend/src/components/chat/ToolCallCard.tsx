import { useState } from 'react';
import { clsx } from 'clsx';
import {
  ChevronDown,
  ChevronRight,
  FileText,
  Pencil,
  Terminal,
  Search,
  Globe,
  Loader2,
  Check,
  X,
  Eye,
  FolderOpen,
  Wrench,
} from 'lucide-react';
import type { ToolCallInfo } from '../../types';

interface ToolCallCardProps {
  toolCall: ToolCallInfo;
}

const kindIcons: Record<string, React.ElementType> = {
  read: Eye,
  write: Pencil,
  edit: Pencil,
  execute: Terminal,
  bash: Terminal,
  search: Search,
  glob: FolderOpen,
  grep: Search,
  web: Globe,
  file: FileText,
};

const statusConfig = {
  pending: {
    color: 'text-[var(--warning)]',
    bg: 'bg-[var(--warning)]',
    icon: Loader2,
    label: 'Pending',
    spin: false,
  },
  in_progress: {
    color: 'text-[var(--accent)]',
    bg: 'bg-[var(--accent)]',
    icon: Loader2,
    label: 'Running',
    spin: true,
  },
  completed: {
    color: 'text-[var(--success)]',
    bg: 'bg-[var(--success)]',
    icon: Check,
    label: 'Done',
    spin: false,
  },
  failed: {
    color: 'text-[var(--error)]',
    bg: 'bg-[var(--error)]',
    icon: X,
    label: 'Failed',
    spin: false,
  },
};

export function ToolCallCard({ toolCall }: ToolCallCardProps) {
  const [expanded, setExpanded] = useState(false);

  const IconComponent = kindIcons[toolCall.kind] || Wrench;
  const status = statusConfig[toolCall.status] || statusConfig.pending;
  const StatusIcon = status.icon;

  const hasDiff = toolCall.content.includes('<<<') || toolCall.content.includes('---');

  return (
    <div className="mx-4 my-1.5">
      <div className="bg-[var(--bg-secondary)] border border-[var(--border)] rounded-lg overflow-hidden">
        {/* Header */}
        <button
          onClick={() => setExpanded(!expanded)}
          className="w-full flex items-center gap-2 px-3 py-2 hover:bg-[var(--bg-tertiary)] transition-colors text-left"
        >
          <IconComponent className="w-4 h-4 text-[var(--text-secondary)] shrink-0" />

          <span className="flex-1 text-xs font-medium truncate">
            {toolCall.title}
          </span>

          {/* Status badge */}
          <div
            className={clsx(
              'flex items-center gap-1 text-[10px] px-1.5 py-0.5 rounded-full',
              status.color
            )}
          >
            <StatusIcon
              className={clsx('w-3 h-3', status.spin && 'animate-spin')}
            />
            <span>{status.label}</span>
          </div>

          {toolCall.content && (
            expanded ? (
              <ChevronDown className="w-3.5 h-3.5 text-[var(--text-secondary)] shrink-0" />
            ) : (
              <ChevronRight className="w-3.5 h-3.5 text-[var(--text-secondary)] shrink-0" />
            )
          )}
        </button>

        {/* Content */}
        {expanded && toolCall.content && (
          <div className="border-t border-[var(--border)] px-3 py-2 max-h-[300px] overflow-auto">
            {hasDiff ? (
              <DiffView content={toolCall.content} />
            ) : (
              <pre className="text-xs text-[var(--text-secondary)] whitespace-pre-wrap break-words font-mono">
                {toolCall.content}
              </pre>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function DiffView({ content }: { content: string }) {
  const lines = content.split('\n');

  return (
    <div className="text-xs font-mono">
      {lines.map((line, i) => {
        let className = 'text-[var(--text-secondary)]';
        if (line.startsWith('+') && !line.startsWith('+++')) {
          className = 'text-[var(--success)] bg-[var(--success)] bg-opacity-10';
        } else if (line.startsWith('-') && !line.startsWith('---')) {
          className = 'text-[var(--error)] bg-[var(--error)] bg-opacity-10';
        } else if (line.startsWith('@@')) {
          className = 'text-[var(--accent)]';
        }

        return (
          <div key={i} className={clsx('px-1 whitespace-pre-wrap', className)}>
            {line}
          </div>
        );
      })}
    </div>
  );
}
