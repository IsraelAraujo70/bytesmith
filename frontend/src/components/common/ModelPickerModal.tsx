import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Cpu, Search, X, ChevronRight, ChevronDown, Check, Sparkles } from 'lucide-react';
import { clsx } from 'clsx';
import { useAppStore } from '../../stores/appStore';
import { setSessionModel } from '../../lib/api';
import type { SessionModelInfo } from '../../types';

// ─── Provider extraction helpers ───────────────────────────────

const KNOWN_PROVIDERS: Record<string, string> = {
  anthropic: 'Anthropic',
  claude: 'Anthropic',
  openai: 'OpenAI',
  gpt: 'OpenAI',
  'o1': 'OpenAI',
  'o3': 'OpenAI',
  'o4': 'OpenAI',
  google: 'Google',
  meta: 'Meta',
  llama: 'Meta',
  mistral: 'Mistral',
  codestral: 'Mistral',
  deepseek: 'DeepSeek',
  cohere: 'Cohere',
  command: 'Cohere',
  perplexity: 'Perplexity',
  groq: 'Groq',
  together: 'Together',
  fireworks: 'Fireworks',
  xai: 'xAI',
  grok: 'xAI',
};

const PROVIDER_ICONS: Record<string, string> = {
  Anthropic: '🟠',
  OpenAI: '🟢',
  Google: '🔵',
  Meta: '🟣',
  Mistral: '🟡',
  DeepSeek: '🔷',
  xAI: '⚪',
  Other: '⚙️',
};

function extractProvider(model: SessionModelInfo): string {
  const id = model.modelId.toLowerCase();
  const name = model.name.toLowerCase();

  // Check for slash-separated provider (e.g. "anthropic/claude-3-opus")
  if (id.includes('/')) {
    const prefix = id.split('/')[0];
    if (KNOWN_PROVIDERS[prefix]) return KNOWN_PROVIDERS[prefix];
  }

  // Check known prefixes against modelId and name
  for (const [key, provider] of Object.entries(KNOWN_PROVIDERS)) {
    if (id.startsWith(key) || name.startsWith(key)) {
      return provider;
    }
  }

  return 'Other';
}

// ─── Grouped model type ────────────────────────────────────────

interface ModelGroup {
  provider: string;
  icon: string;
  models: SessionModelInfo[];
}

function groupByProvider(models: SessionModelInfo[]): ModelGroup[] {
  const groups = new Map<string, SessionModelInfo[]>();

  for (const model of models) {
    const provider = extractProvider(model);
    if (!groups.has(provider)) {
      groups.set(provider, []);
    }
    groups.get(provider)!.push(model);
  }

  // Sort groups: providers with more models first, then alphabetically
  return Array.from(groups.entries())
    .sort(([a, modelsA], [b, modelsB]) => {
      if (modelsA.length !== modelsB.length) return modelsB.length - modelsA.length;
      return a.localeCompare(b);
    })
    .map(([provider, models]) => ({
      provider,
      icon: PROVIDER_ICONS[provider] || PROVIDER_ICONS.Other,
      models: models.sort((a, b) => a.name.localeCompare(b.name)),
    }));
}

// ─── Fuzzy-ish search ──────────────────────────────────────────

function matchesSearch(model: SessionModelInfo, provider: string, query: string): boolean {
  if (!query) return true;
  const q = query.toLowerCase();
  return (
    model.name.toLowerCase().includes(q) ||
    model.modelId.toLowerCase().includes(q) ||
    provider.toLowerCase().includes(q)
  );
}

// ─── Flat item for keyboard navigation ─────────────────────────

interface FlatItem {
  type: 'provider' | 'model';
  provider: string;
  model?: SessionModelInfo;
  index: number;
}

// ─── Component ─────────────────────────────────────────────────

export function ModelPickerModal() {
  const {
    models,
    currentModelId,
    activeSession,
    setSessionModels,
    setError,
    modelPickerOpen,
    setModelPickerOpen,
  } = useAppStore();

  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [collapsedProviders, setCollapsedProviders] = useState<Set<string>>(new Set());
  const [updating, setUpdating] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const itemRefs = useRef<Map<number, HTMLElement>>(new Map());

  // Reset state when opened
  useEffect(() => {
    if (modelPickerOpen) {
      setQuery('');
      setSelectedIndex(0);
      setCollapsedProviders(new Set());
      // Focus input after animation
      requestAnimationFrame(() => {
        inputRef.current?.focus();
      });
    }
  }, [modelPickerOpen]);

  // Filter + group
  const groups = useMemo(() => {
    const allGroups = groupByProvider(models);
    if (!query) return allGroups;

    return allGroups
      .map((group) => ({
        ...group,
        models: group.models.filter((m) => matchesSearch(m, group.provider, query)),
      }))
      .filter((group) => group.models.length > 0);
  }, [models, query]);

  // Build flat navigation list
  const flatItems = useMemo(() => {
    const items: FlatItem[] = [];
    let idx = 0;
    for (const group of groups) {
      items.push({ type: 'provider', provider: group.provider, index: idx++ });
      if (!collapsedProviders.has(group.provider)) {
        for (const model of group.models) {
          items.push({ type: 'model', provider: group.provider, model, index: idx++ });
        }
      }
    }
    return items;
  }, [groups, collapsedProviders]);

  // Count total visible models
  const totalModels = useMemo(() => {
    return groups.reduce((sum, g) => sum + g.models.length, 0);
  }, [groups]);

  // Clamp selected index
  useEffect(() => {
    if (selectedIndex >= flatItems.length) {
      setSelectedIndex(Math.max(0, flatItems.length - 1));
    }
  }, [flatItems.length, selectedIndex]);

  // Scroll selected item into view
  useEffect(() => {
    const el = itemRefs.current.get(selectedIndex);
    if (el) {
      el.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    }
  }, [selectedIndex]);

  const close = useCallback(() => {
    setModelPickerOpen(false);
  }, [setModelPickerOpen]);

  const toggleProvider = useCallback((provider: string) => {
    setCollapsedProviders((prev) => {
      const next = new Set(prev);
      if (next.has(provider)) {
        next.delete(provider);
      } else {
        next.add(provider);
      }
      return next;
    });
  }, []);

  const selectModel = useCallback(async (model: SessionModelInfo) => {
    if (!activeSession || updating) return;
    if (model.modelId === currentModelId) {
      close();
      return;
    }

    setUpdating(true);
    try {
      await setSessionModel(
        activeSession.connectionID,
        activeSession.sessionID,
        model.modelId
      );
      setSessionModels(models, model.modelId);
      close();
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(`Failed to switch model: ${message}`);
    } finally {
      setUpdating(false);
    }
  }, [activeSession, updating, currentModelId, models, setSessionModels, setError, close]);

  // Keyboard navigation
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    switch (e.key) {
      case 'Escape':
        e.preventDefault();
        close();
        break;

      case 'ArrowDown':
        e.preventDefault();
        setSelectedIndex((prev) => Math.min(prev + 1, flatItems.length - 1));
        break;

      case 'ArrowUp':
        e.preventDefault();
        setSelectedIndex((prev) => Math.max(prev - 1, 0));
        break;

      case 'Enter': {
        e.preventDefault();
        const item = flatItems[selectedIndex];
        if (!item) break;
        if (item.type === 'provider') {
          toggleProvider(item.provider);
        } else if (item.model) {
          selectModel(item.model);
        }
        break;
      }

      case 'Tab':
        e.preventDefault();
        // Tab cycles through, shift+tab goes back
        if (e.shiftKey) {
          setSelectedIndex((prev) => Math.max(prev - 1, 0));
        } else {
          setSelectedIndex((prev) => Math.min(prev + 1, flatItems.length - 1));
        }
        break;
    }
  }, [close, flatItems, selectedIndex, toggleProvider, selectModel]);

  // Close on backdrop click
  const handleBackdropClick = useCallback((e: React.MouseEvent) => {
    if (e.target === e.currentTarget) close();
  }, [close]);

  // Global Ctrl+K / Cmd+K to open
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        if (models.length > 0 && activeSession) {
          setModelPickerOpen(!modelPickerOpen);
        }
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [models.length, activeSession, modelPickerOpen, setModelPickerOpen]);

  if (!modelPickerOpen) return null;

  const currentModel = models.find((m) => m.modelId === currentModelId);

  return (
    <div
      className="fixed inset-0 z-[100] flex items-start justify-center pt-[15vh] animate-fade-in"
      style={{ background: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(4px)' }}
      onClick={handleBackdropClick}
      onKeyDown={handleKeyDown}
    >
      <div
        className="w-full max-w-[520px] bg-[var(--bg-elevated)] border border-[var(--border)] rounded-xl shadow-elevated overflow-hidden animate-slide-up"
        role="dialog"
        aria-label="Model Picker"
      >
        {/* ── Header / Search ── */}
        <div className="flex items-center gap-2 px-4 py-3 border-b border-[var(--border-subtle)]">
          <Search className="w-4 h-4 text-[var(--text-muted)] shrink-0" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setSelectedIndex(0);
            }}
            placeholder="Search models..."
            className="flex-1 bg-transparent text-sm text-[var(--text-primary)] placeholder:text-[var(--text-muted)] outline-none"
            autoComplete="off"
            spellCheck={false}
          />
          {query && (
            <button
              onClick={() => { setQuery(''); inputRef.current?.focus(); }}
              className="p-0.5 rounded hover:bg-[var(--bg-tertiary)] text-[var(--text-muted)] transition-colors"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          )}
          <kbd className="hidden sm:inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded bg-[var(--bg-tertiary)] border border-[var(--border-subtle)] text-[10px] text-[var(--text-muted)] font-mono">
            Esc
          </kbd>
        </div>

        {/* ── Current model indicator ── */}
        {currentModel && (
          <div className="flex items-center gap-2 px-4 py-2 bg-[var(--accent-muted)] border-b border-[var(--border-subtle)]">
            <Sparkles className="w-3.5 h-3.5 text-[var(--accent)]" />
            <span className="text-xs text-[var(--accent)]">Current:</span>
            <span className="text-xs text-[var(--text-primary)] font-medium truncate">
              {currentModel.name}
            </span>
            <span className="text-[10px] text-[var(--text-muted)] font-mono ml-auto">
              {extractProvider(currentModel)}
            </span>
          </div>
        )}

        {/* ── Model List ── */}
        <div
          ref={listRef}
          className="max-h-[50vh] overflow-y-auto py-1"
          role="listbox"
          aria-activedescendant={`model-item-${selectedIndex}`}
        >
          {flatItems.length === 0 && (
            <div className="flex flex-col items-center justify-center py-10 text-[var(--text-muted)]">
              <Cpu className="w-8 h-8 mb-2 opacity-30" />
              <span className="text-sm">
                {models.length === 0
                  ? 'No models available'
                  : 'No models match your search'}
              </span>
              {query && (
                <button
                  onClick={() => setQuery('')}
                  className="mt-2 text-xs text-[var(--accent)] hover:underline"
                >
                  Clear search
                </button>
              )}
            </div>
          )}

          {groups.map((group) => {
            const isCollapsed = collapsedProviders.has(group.provider);
            const providerFlatItem = flatItems.find(
              (fi) => fi.type === 'provider' && fi.provider === group.provider
            );
            const providerIdx = providerFlatItem?.index ?? -1;

            return (
              <div key={group.provider}>
                {/* Provider header */}
                <button
                  ref={(el) => { if (el) itemRefs.current.set(providerIdx, el); }}
                  id={`model-item-${providerIdx}`}
                  onClick={() => toggleProvider(group.provider)}
                  className={clsx(
                    'w-full flex items-center gap-2 px-4 py-1.5 text-left transition-colors duration-100',
                    selectedIndex === providerIdx
                      ? 'bg-[var(--bg-tertiary)]'
                      : 'hover:bg-[var(--bg-tertiary)]'
                  )}
                  role="option"
                  aria-selected={selectedIndex === providerIdx}
                >
                  {isCollapsed ? (
                    <ChevronRight className="w-3 h-3 text-[var(--text-muted)]" />
                  ) : (
                    <ChevronDown className="w-3 h-3 text-[var(--text-muted)]" />
                  )}
                  <span className="text-xs">{group.icon}</span>
                  <span className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wider">
                    {group.provider}
                  </span>
                  <span className="text-[10px] text-[var(--text-muted)] ml-auto font-mono">
                    {group.models.length}
                  </span>
                </button>

                {/* Models in this group */}
                {!isCollapsed && group.models.map((model) => {
                  const modelFlatItem = flatItems.find(
                    (fi) => fi.type === 'model' && fi.model?.modelId === model.modelId
                  );
                  const modelIdx = modelFlatItem?.index ?? -1;
                  const isCurrent = model.modelId === currentModelId;

                  return (
                    <button
                      ref={(el) => { if (el) itemRefs.current.set(modelIdx, el); }}
                      id={`model-item-${modelIdx}`}
                      key={model.modelId}
                      onClick={() => selectModel(model)}
                      disabled={updating}
                      className={clsx(
                        'w-full flex items-center gap-3 pl-10 pr-4 py-2 text-left transition-all duration-100',
                        'disabled:opacity-50',
                        selectedIndex === modelIdx
                          ? 'bg-[var(--bg-tertiary)]'
                          : 'hover:bg-[var(--bg-tertiary)]',
                        isCurrent && 'border-l-2 border-l-[var(--accent)]'
                      )}
                      role="option"
                      aria-selected={selectedIndex === modelIdx}
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span
                            className={clsx(
                              'text-sm truncate',
                              isCurrent
                                ? 'text-[var(--accent)] font-medium'
                                : 'text-[var(--text-primary)]'
                            )}
                          >
                            {model.name}
                          </span>
                          {isCurrent && (
                            <span className="shrink-0 flex items-center gap-0.5 px-1.5 py-0.5 rounded-full bg-[var(--accent-muted)] text-[9px] text-[var(--accent)] font-medium uppercase tracking-wider">
                              <Check className="w-2.5 h-2.5" />
                              active
                            </span>
                          )}
                        </div>
                        <span className="text-[11px] text-[var(--text-muted)] font-mono truncate block mt-0.5">
                          {model.modelId}
                        </span>
                      </div>

                      {/* Selection indicator */}
                      {selectedIndex === modelIdx && (
                        <div className="shrink-0 w-1.5 h-1.5 rounded-full bg-[var(--accent)]" />
                      )}
                    </button>
                  );
                })}
              </div>
            );
          })}
        </div>

        {/* ── Footer ── */}
        <div className="flex items-center justify-between px-4 py-2 border-t border-[var(--border-subtle)] text-[10px] text-[var(--text-muted)]">
          <div className="flex items-center gap-3">
            <span className="flex items-center gap-1">
              <kbd className="px-1 py-0.5 rounded bg-[var(--bg-tertiary)] border border-[var(--border-subtle)] font-mono">↑↓</kbd>
              navigate
            </span>
            <span className="flex items-center gap-1">
              <kbd className="px-1 py-0.5 rounded bg-[var(--bg-tertiary)] border border-[var(--border-subtle)] font-mono">↵</kbd>
              select
            </span>
            <span className="flex items-center gap-1">
              <kbd className="px-1 py-0.5 rounded bg-[var(--bg-tertiary)] border border-[var(--border-subtle)] font-mono">esc</kbd>
              close
            </span>
          </div>
          <span className="font-mono">
            {totalModels} model{totalModels !== 1 ? 's' : ''}
          </span>
        </div>
      </div>
    </div>
  );
}
