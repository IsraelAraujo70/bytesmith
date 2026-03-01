export type UnifiedDiffRowKind = 'hunk' | 'context' | 'add' | 'del';

export interface UnifiedDiffRow {
  kind: UnifiedDiffRowKind;
  text: string;
}

export interface UnifiedDiffData {
  path: string;
  additions: number;
  deletions: number;
  rows: UnifiedDiffRow[];
}

type Op = { kind: 'context' | 'add' | 'del'; text: string };

function splitLines(input: string): string[] {
  const normalized = (input || '').replace(/\r\n/g, '\n');
  const lines = normalized.split('\n');
  if (lines.length > 0 && lines[lines.length - 1] === '') {
    lines.pop();
  }
  if (lines.length === 1 && lines[0] === '') {
    return [];
  }
  return lines;
}

function buildOps(oldLines: string[], newLines: string[]): Op[] {
  const m = oldLines.length;
  const n = newLines.length;

  const dp: number[][] = Array.from({ length: m + 1 }, () =>
    Array.from({ length: n + 1 }, () => 0)
  );

  for (let i = m - 1; i >= 0; i--) {
    for (let j = n - 1; j >= 0; j--) {
      if (oldLines[i] === newLines[j]) {
        dp[i][j] = dp[i + 1][j + 1] + 1;
      } else {
        dp[i][j] = Math.max(dp[i + 1][j], dp[i][j + 1]);
      }
    }
  }

  const ops: Op[] = [];
  let i = 0;
  let j = 0;

  while (i < m && j < n) {
    if (oldLines[i] === newLines[j]) {
      ops.push({ kind: 'context', text: oldLines[i] });
      i++;
      j++;
      continue;
    }

    if (dp[i + 1][j] >= dp[i][j + 1]) {
      ops.push({ kind: 'del', text: oldLines[i] });
      i++;
    } else {
      ops.push({ kind: 'add', text: newLines[j] });
      j++;
    }
  }

  while (i < m) {
    ops.push({ kind: 'del', text: oldLines[i] });
    i++;
  }

  while (j < n) {
    ops.push({ kind: 'add', text: newLines[j] });
    j++;
  }

  return ops;
}

export function buildUnifiedDiff(path: string, oldText: string, newText: string): UnifiedDiffData {
  const oldLines = splitLines(oldText);
  const newLines = splitLines(newText);
  const ops = buildOps(oldLines, newLines);

  let additions = 0;
  let deletions = 0;
  const rows: UnifiedDiffRow[] = [];

  rows.push({
    kind: 'hunk',
    text: `@@ -1,${oldLines.length} +1,${newLines.length} @@`,
  });

  for (const op of ops) {
    if (op.kind === 'add') {
      additions++;
      rows.push({ kind: 'add', text: `+${op.text}` });
    } else if (op.kind === 'del') {
      deletions++;
      rows.push({ kind: 'del', text: `-${op.text}` });
    } else {
      rows.push({ kind: 'context', text: ` ${op.text}` });
    }
  }

  return {
    path,
    additions,
    deletions,
    rows,
  };
}
