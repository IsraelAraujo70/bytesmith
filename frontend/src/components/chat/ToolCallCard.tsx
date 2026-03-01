import { useEffect, useMemo, useState } from 'react';
import { clsx } from 'clsx';
import {
  ChevronDown,
  ChevronRight,
  AlertTriangle,
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
import { buildUnifiedDiff } from '../../lib/diff';

interface ToolCallCardProps {
  toolCall: ToolCallInfo;
}

interface PermissionDeniedDetails {
  message: string;
  input?: string;
  output?: string;
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
  const diffParts = (toolCall.parts || []).filter((part) => part.type === 'diff');

  const structuredDiffs = useMemo(
    () =>
      diffParts.map((part, idx) =>
        buildUnifiedDiff(
          (part.path || '').trim() || `file-${idx + 1}`,
          part.oldText || '',
          part.newText || ''
        )
      ),
    [diffParts]
  );

  const fallbackHasDiff = useMemo(
    () =>
      /(^@@)/m.test(content) ||
      /(^\+\+\+ )/m.test(content) ||
      /(^--- )/m.test(content) ||
      /(^Diff:)/m.test(content),
    [content]
  );

  const hasStructuredDiff = structuredDiffs.length > 0;
  const hasContent = hasStructuredDiff || content.length > 0;
  const isRunning = toolCall.status === 'in_progress';
  const permissionDenied = useMemo(
    () => parsePermissionDenied(content),
    [content]
  );

  const diffSummary = useMemo(() => {
    if (toolCall.diffSummary) {
      return toolCall.diffSummary;
    }

    if (!hasStructuredDiff) {
      return null;
    }

    return structuredDiffs.reduce(
      (acc, diff) => ({
        additions: acc.additions + diff.additions,
        deletions: acc.deletions + diff.deletions,
        files: acc.files + 1,
      }),
      { additions: 0, deletions: 0, files: 0 }
    );
  }, [toolCall.diffSummary, hasStructuredDiff, structuredDiffs]);

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

          {diffSummary && (diffSummary.additions > 0 || diffSummary.deletions > 0 || diffSummary.files > 0) && (
            <span className="text-[9px] font-mono text-[var(--text-muted)]">
              <span className="text-[var(--success)]">+{diffSummary.additions}</span>{' '}
              <span className="text-[var(--error)]">-{diffSummary.deletions}</span>
            </span>
          )}

          {/* Status badge */}
          <div className={clsx('flex items-center gap-1 text-[9px] font-mono', status.color)}>
            <StatusIcon className={clsx('w-2.5 h-2.5', status.spin && 'animate-spin')} />
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
          <div className="border-t border-[var(--border-subtle)] px-2.5 py-2 max-h-[340px] overflow-auto space-y-2">
            {permissionDenied ? (
              <PermissionDeniedView details={permissionDenied} />
            ) : hasStructuredDiff ? (
              <StructuredDiffView diffs={structuredDiffs} />
            ) : fallbackHasDiff ? (
              <LegacyDiffView content={content} />
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

function PermissionDeniedView({
  details,
}: {
  details: PermissionDeniedDetails;
}) {
  return (
    <div className="space-y-2">
      <div className="rounded border border-[var(--error)] bg-[var(--error-muted)] px-2.5 py-2">
        <div className="flex items-start gap-2 text-[var(--error)]">
          <AlertTriangle className="w-3.5 h-3.5 shrink-0 mt-0.5" />
          <div className="min-w-0">
            <div className="text-[10px] uppercase tracking-wide font-semibold">
              Permission Rejected
            </div>
            <div className="text-[11px] leading-relaxed break-words font-mono text-[var(--text-primary)] mt-1">
              {details.message}
            </div>
          </div>
        </div>
      </div>

      {details.input && <ToolPayloadSection title="Input" body={details.input} />}
      {details.output && <ToolPayloadSection title="Output" body={details.output} />}
    </div>
  );
}

function ToolPayloadSection({
  title,
  body,
}: {
  title: string;
  body: string;
}) {
  return (
    <details className="rounded border border-[var(--border-subtle)] bg-[var(--bg-primary)]" open={title === 'Output'}>
      <summary className="cursor-pointer select-none px-2.5 py-1.5 text-[10px] uppercase tracking-wide font-semibold text-[var(--text-muted)]">
        {title}
      </summary>
      <div className="border-t border-[var(--border-subtle)] px-2.5 py-2">
        <pre className="text-[11px] text-[var(--text-secondary)] whitespace-pre-wrap break-words font-mono leading-relaxed">
          {body}
        </pre>
      </div>
    </details>
  );
}

function parsePermissionDenied(content: string): PermissionDeniedDetails | null {
  if (!content) {
    return null;
  }

  const normalized = content.toLowerCase();
  const hasPermissionDenial =
    normalized.includes('user rejected permission') ||
    normalized.includes('rejected permission to use this specific tool call') ||
    normalized.includes('permission denied');

  if (!hasPermissionDenial) {
    return null;
  }

  const errorLine =
    content.match(/(?:^|\n)Error:\s*([^\n]+)/i)?.[1]?.trim() || '';
  const input = extractSection(content, 'Input');
  const output = extractSection(content, 'Output');
  let outputError = '';

  if (output) {
    try {
      const parsed = JSON.parse(output) as { error?: unknown };
      if (typeof parsed.error === 'string' && parsed.error.trim() !== '') {
        outputError = parsed.error.trim();
      }
    } catch {
      outputError = '';
    }
  }

  const message = errorLine || outputError || 'The user rejected this tool action.';
  return {
    message,
    ...(input ? { input } : {}),
    ...(output ? { output } : {}),
  };
}

function extractSection(content: string, title: string): string {
  const pattern = new RegExp(
    `(?:^|\\n)${title}:\\n([\\s\\S]*?)(?=\\n\\n(?:[A-Za-z][^\\n]{0,60}:\\n)|$)`,
    'i'
  );
  const match = content.match(pattern);
  if (!match) {
    return '';
  }
  return match[1].trim();
}

function StructuredDiffView({
  diffs,
}: {
  diffs: ReturnType<typeof buildUnifiedDiff>[];
}) {
  return (
    <div className="space-y-2">
      {diffs.map((diff, idx) => (
        <div key={`${diff.path}-${idx}`} className="rounded border border-[var(--border-subtle)] overflow-hidden">
          <div className="px-2 py-1 text-[10px] font-mono bg-[var(--bg-tertiary)] text-[var(--text-secondary)] border-b border-[var(--border-subtle)]">
            {diff.path}
          </div>
          <div className="text-[11px] font-mono leading-relaxed p-1">
            {diff.rows.map((row, rowIndex) => {
              let className = 'text-[var(--text-secondary)]';
              if (row.kind === 'add') {
                className = 'diff-add';
              } else if (row.kind === 'del') {
                className = 'diff-del';
              } else if (row.kind === 'hunk') {
                className = 'diff-hunk';
              }

              return (
                <div
                  key={`${diff.path}-${rowIndex}`}
                  className={clsx('px-1 whitespace-pre-wrap rounded-sm', className)}
                >
                  {row.text}
                </div>
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
}

function LegacyDiffView({ content }: { content: string }) {
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
