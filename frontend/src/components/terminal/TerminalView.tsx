import { useCallback, useEffect, useRef } from 'react';
import { clsx } from 'clsx';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import type { EmbeddedTerminalTab } from '../../types';
import { resizeEmbeddedTerminal, writeEmbeddedTerminal } from '../../lib/api';
import { useAppStore } from '../../stores/appStore';

interface TerminalViewProps {
  terminal: EmbeddedTerminalTab;
  active: boolean;
}

export function TerminalView({ terminal, active }: TerminalViewProps) {
  const setError = useAppStore((s) => s.setError);
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<XTerm | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const writtenRef = useRef(0);
  const exitedRef = useRef(terminal.exited);

  useEffect(() => {
    exitedRef.current = terminal.exited;
  }, [terminal.exited]);

  const fitAndSync = useCallback(() => {
    const term = termRef.current;
    const fitAddon = fitRef.current;
    if (!term || !fitAddon) {
      return;
    }

    fitAddon.fit();
    if (term.cols <= 0 || term.rows <= 0) {
      return;
    }

    void resizeEmbeddedTerminal(terminal.id, term.cols, term.rows).catch(() => {
      // Ignore transient resize errors while a tab is shutting down.
    });
  }, [terminal.id]);

  useEffect(() => {
    const host = containerRef.current;
    if (!host) {
      return;
    }

    const rootStyles = getComputedStyle(document.documentElement);
    const term = new XTerm({
      cursorBlink: true,
      fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
      fontSize: 12,
      scrollback: 8000,
      theme: {
        background: rootStyles.getPropertyValue('--bg-primary').trim() || '#0c0b0a',
        foreground: rootStyles.getPropertyValue('--text-primary').trim() || '#e8e4df',
        cursor: rootStyles.getPropertyValue('--accent').trim() || '#e07a2f',
      },
    });
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(host);

    termRef.current = term;
    fitRef.current = fitAddon;
    writtenRef.current = 0;

    const resizeObserver = new ResizeObserver(() => {
      if (active) {
        fitAndSync();
      }
    });
    resizeObserver.observe(host);

    const onWindowResize = () => {
      if (active) {
        fitAndSync();
      }
    };
    window.addEventListener('resize', onWindowResize);

    const onDataDispose = term.onData((data) => {
      if (exitedRef.current) {
        return;
      }
      void writeEmbeddedTerminal(terminal.id, data).catch((err) => {
        const reason = err instanceof Error ? err.message : String(err);
        setError(`Failed to write to terminal: ${reason}`);
      });
    });

    return () => {
      onDataDispose.dispose();
      window.removeEventListener('resize', onWindowResize);
      resizeObserver.disconnect();
      fitRef.current = null;
      termRef.current = null;
      term.dispose();
    };
  }, [active, fitAndSync, setError, terminal.id]);

  useEffect(() => {
    const term = termRef.current;
    if (!term) {
      return;
    }

    if (terminal.buffer.length < writtenRef.current) {
      term.reset();
      writtenRef.current = 0;
    }

    const chunk = terminal.buffer.slice(writtenRef.current);
    if (!chunk) {
      return;
    }

    term.write(chunk);
    writtenRef.current = terminal.buffer.length;
  }, [terminal.buffer]);

  useEffect(() => {
    if (!active) {
      return;
    }
    requestAnimationFrame(() => {
      fitAndSync();
      termRef.current?.focus();
    });
  }, [active, fitAndSync]);

  return (
    <div className={clsx('h-full w-full bytesmith-terminal', !active && 'hidden')}>
      <div ref={containerRef} className="h-full w-full" />
    </div>
  );
}
