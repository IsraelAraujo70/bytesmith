import { useEffect, useMemo, useState } from 'react';
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
    borderColor: 'border-l-[var(--warning)]',
    icon: Loader2,
    label: 'Pending',
    spin: false,
  },
  in_progress: {
    color: 'text-[var(--accent)]',
    borderColor: 'border-l-[var(--accent)]',
    icon: Loader2,
    label: 'Running',
    spin: true,
  },
  completed: {
    color: 'text-[var(--success)]',
    borderColor: 'border-l-[var(--success)]',
    icon: Check,
    label: 'Done',
    spin: false,
  },
  failed: {
    color: 'text-[var(--error)]',
    borderColor: 'border-l-[var(--error)]',
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

  const content = (toolCall.content || '').trim();
  const hasContent = content.length > 0;
  const hasDiff = useMemo(
    () =>
      /(^@@)/m.test(content) ||
      /(^\+\+\+ )/m.test(content) ||
      /(^--- )/m.test(content) ||
      /(^Diff:)/m.test(content),
    [content]
  );
  const isRunning = toolCall.status === 'in_progress';

  useEffect(() => {
    if (isRunning && hasContent) {
      setExpanded(true);
    }
  }, [isRunning, hasContent]);

  return (
    <div className="mx-5 my-1.5 animate-fade-in">
      <div
        className={clsx(
          'bg-[var(--bg-secondary)] border border-[var(--border-subtle)] rounded-md overflow-hidden transition-all duration-200',
          'border-l-2',
          status.borderColor,
          isRunning && 'animate-glow-pulse'
        )}
      >
        {/* Header */}
        <button
          onClick={() => hasContent && setExpanded(!expanded)}
          className="w-full flex items-center gap-2 px-2.5 py-1.5 hover:bg-[var(--bg-tertiary)] transition-colors text-left"
        >
          <IconComponent className="w-3.5 h-3.5 text-[var(--text-muted)] shrink-0" />

          <span className="flex-1 text-[11px] font-medium truncate text-[var(--text-secondary)]">
            {toolCall.title}
          </span>

          {/* Status badge */}
          <div
            className={clsx(
              'flex items-center gap-1 text-[9px] font-mono',
              status.color
            )}
          >
            <StatusIcon
              className={clsx('w-2.5 h-2.5', status.spin && 'animate-spin')}
            />
            <span>{status.label}</span>
          </div>

          {hasContent && (
            expanded ? (
              <ChevronDown className="w-3 h-3 text-[var(--text-muted)] shrink-0" />
            ) : (
              <ChevronRight className="w-3 h-3 text-[var(--text-muted)] shrink-0" />
            )
          )}
        </button>

        {/* Content */}
        {expanded && hasContent && (
          <div className="border-t border-[var(--border-subtle)] px-2.5 py-2 max-h-[300px] overflow-auto">
            {hasDiff ? (
              <DiffView content={content} />
            ) : (
              <pre className="text-[11px] text-[var(--text-secondary)] whitespace-pre-wrap break-words font-mono leading-relaxed">
                {content}
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
    <div className="text-[11px] font-mono leading-relaxed">
      {lines.map((line, i) => {
        let className = 'text-[var(--text-secondary)]';
        if (line.startsWith('+') && !line.startsWith('+++')) {
          className = 'diff-add';
        } else if (line.startsWith('-') && !line.startsWith('---')) {
          className = 'diff-del';
        } else if (line.startsWith('@@')) {
          className = 'diff-hunk';
        }

        return (
          <div key={i} className={clsx('px-1 whitespace-pre-wrap rounded-sm', className)}>
            {line}
          </div>
        );
      })}
    </div>
  );
}
