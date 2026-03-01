import { useEffect, useState } from 'react';
import {
  Bot,
  ChevronDown,
  ChevronRight,
  Folder,
  Hammer,
  MessageSquare,
  Plus,
  Settings,
  X,
  Zap,
} from 'lucide-react';
import { clsx } from 'clsx';
import { useAppStore } from '../../stores/appStore';
import {
  listAgents,
  connectAgent,
  disconnectAgent,
  listConnections,
  listSessions,
  getSessionHistory,
  getSessionModels,
  getSessionModes,
  createSession,
  pickDirectory,
} from '../../lib/api';
import type { ConnectionInfo, SessionListItem } from '../../types';

export function Sidebar() {
  const {
    agents,
    setAgents,
    connections,
    setConnections,
    activeSession,
    setActiveSession,
    cwd,
    setCwd,
    clearSession,
    setMessages,
    setToolCalls,
    setSessionModels,
    setSessionModes,
    setLoading,
    setError,
  } = useAppStore();

  const [selectedAgent, setSelectedAgent] = useState('');
  const [expandedConns, setExpandedConns] = useState<Set<string>>(new Set());
  const [connSessions, setConnSessions] = useState<
    Record<string, SessionListItem[]>
  >({});
  const [agentDropdownOpen, setAgentDropdownOpen] = useState(false);

  // Load agents on mount
  useEffect(() => {
    listAgents().then(setAgents);
    listConnections().then(setConnections);
  }, [setAgents, setConnections]);

  const toggleConn = async (connId: string) => {
    const next = new Set(expandedConns);
    if (next.has(connId)) {
      next.delete(connId);
    } else {
      next.add(connId);
      try {
        const sessions = await listSessions(connId);
        setConnSessions((prev) => ({ ...prev, [connId]: sessions }));
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        setError(`Falha ao listar sessoes: ${message}`);
      }
    }
    setExpandedConns(next);
  };

  const handleConnect = async () => {
    if (!selectedAgent) {
      setError('Selecione um agente para conectar.');
      return;
    }
    if (!cwd) {
      setError('Selecione um diretorio antes de conectar.');
      return;
    }

    setLoading(true);
    try {
      const connId = await connectAgent(selectedAgent, cwd);
      const sessionId = await createSession(connId, cwd);
      const [modelsInfo, modesInfo] = await Promise.all([
        getSessionModels(sessionId),
        getSessionModes(sessionId),
      ]);
      const agent = agents.find((a) => a.name === selectedAgent);
      const conn: ConnectionInfo = {
        id: connId,
        agentName: selectedAgent,
        displayName: agent?.displayName || selectedAgent,
        sessions: [sessionId],
        integrator: selectedAgent,
      };

      setConnections([...connections, conn]);
      setActiveSession({ connectionID: connId, sessionID: sessionId });
      clearSession();
      if (modelsInfo) {
        setSessionModels(modelsInfo.models, modelsInfo.currentModelId);
      }
      if (modesInfo) {
        setSessionModes(modesInfo.modes, modesInfo.currentModeId);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(`Falha ao conectar agente: ${message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleDisconnect = async (connId: string) => {
    try {
      await disconnectAgent(connId);
      setConnections(connections.filter((c) => c.id !== connId));
      if (activeSession?.connectionID === connId) {
        setActiveSession(null);
        clearSession();
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(`Falha ao desconectar agente: ${message}`);
    }
  };

  const handleNewSession = async (connId: string) => {
    if (!cwd) {
      setError('Selecione um diretorio antes de criar nova sessao.');
      return;
    }

    try {
      const sessionId = await createSession(connId, cwd);
      setActiveSession({ connectionID: connId, sessionID: sessionId });
      clearSession();
      const [modelsInfo, modesInfo, sessions] = await Promise.all([
        getSessionModels(sessionId),
        getSessionModes(sessionId),
        listSessions(connId),
      ]);
      if (modelsInfo) {
        setSessionModels(modelsInfo.models, modelsInfo.currentModelId);
      }
      if (modesInfo) {
        setSessionModes(modesInfo.modes, modesInfo.currentModeId);
      }
      setConnSessions((prev) => ({ ...prev, [connId]: sessions }));
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(`Falha ao criar sessao: ${message}`);
    }
  };

  const handleSelectSession = async (connectionID: string, sessionID: string) => {
    setActiveSession({ connectionID, sessionID });
    clearSession();

    setLoading(true);
    try {
      const [history, modelsInfo, modesInfo] = await Promise.all([
        getSessionHistory(sessionID),
        getSessionModels(sessionID),
        getSessionModes(sessionID),
      ]);
      if (history) {
        setMessages(history.messages || []);
        setToolCalls(history.toolCalls || []);
      }
      if (modelsInfo) {
        setSessionModels(modelsInfo.models, modelsInfo.currentModelId);
      }
      if (modesInfo) {
        setSessionModes(modesInfo.modes, modesInfo.currentModeId);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(`Falha ao carregar historico da sessao: ${message}`);
    } finally {
      setLoading(false);
    }
  };

  const handlePickDirectory = async () => {
    try {
      const dir = await pickDirectory();
      if (dir) setCwd(dir);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(`Falha ao selecionar diretorio: ${message}`);
    }
  };

  const selected = agents.find((a) => a.name === selectedAgent);

  return (
    <div className="flex flex-col h-full w-[260px] bg-[var(--bg-secondary)] border-r border-[var(--border-subtle)]">
      {/* ── Header / Brand ── */}
      <div className="flex items-center gap-2.5 px-4 py-3 wails-drag">
        <div className="w-7 h-7 rounded-lg bg-[var(--accent-muted)] flex items-center justify-center">
          <Hammer className="w-4 h-4 text-[var(--accent)]" />
        </div>
        <div className="flex flex-col">
          <span className="text-sm font-semibold tracking-wide text-[var(--text-primary)]">
            ByteSmith
          </span>
          <span className="text-[9px] text-[var(--text-muted)] font-mono tracking-widest uppercase">
            forge
          </span>
        </div>
      </div>

      {/* ── Agent Selector ── */}
      <div className="px-3 py-2.5 border-t border-[var(--border-subtle)]">
        <label className="text-[10px] text-[var(--text-muted)] mb-1 block uppercase tracking-wider font-medium">
          Agent
        </label>
        <div className="relative">
          <button
            onClick={() => setAgentDropdownOpen(!agentDropdownOpen)}
            className={clsx(
              'w-full flex items-center justify-between px-2.5 py-1.5',
              'bg-[var(--bg-tertiary)] border border-[var(--border-subtle)] rounded-md text-xs',
              'hover:border-[var(--border)] transition-all duration-150',
              agentDropdownOpen && 'border-[var(--accent)] glow-accent-sm'
            )}
          >
            <span className="flex items-center gap-2 text-[var(--text-primary)]">
              <Bot className="w-3.5 h-3.5 text-[var(--accent)]" />
              {selected?.displayName || 'Select agent...'}
            </span>
            <ChevronDown
              className={clsx(
                'w-3 h-3 text-[var(--text-muted)] transition-transform duration-150',
                agentDropdownOpen && 'rotate-180'
              )}
            />
          </button>

          {agentDropdownOpen && (
            <div className="absolute z-50 top-full left-0 right-0 mt-1 bg-[var(--bg-elevated)] border border-[var(--border)] rounded-md shadow-elevated overflow-hidden animate-fade-in">
              {agents.map((agent) => (
                <button
                  key={agent.name}
                  disabled={!agent.installed}
                  onClick={() => {
                    setSelectedAgent(agent.name);
                    setAgentDropdownOpen(false);
                  }}
                  className={clsx(
                    'w-full flex items-center gap-2 px-2.5 py-2 text-xs text-left transition-colors',
                    !agent.installed && 'opacity-30 cursor-not-allowed',
                    agent.name === selectedAgent
                      ? 'bg-[var(--accent-muted)] text-[var(--accent)]'
                      : 'hover:bg-[var(--bg-tertiary)] text-[var(--text-primary)]'
                  )}
                >
                  <Bot className="w-3.5 h-3.5 shrink-0" />
                  <div className="flex-1 min-w-0">
                    <div className="truncate font-medium">{agent.displayName}</div>
                    <div className="text-[10px] text-[var(--text-muted)] truncate">
                      {agent.description}
                    </div>
                  </div>
                  {!agent.installed && (
                    <span className="text-[9px] text-[var(--warning)] shrink-0 font-mono">
                      N/A
                    </span>
                  )}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Working Directory */}
        <label className="text-[10px] text-[var(--text-muted)] mt-2.5 mb-1 block uppercase tracking-wider font-medium">
          Directory
        </label>
        <button
          onClick={handlePickDirectory}
          className="w-full flex items-center gap-2 px-2.5 py-1.5 bg-[var(--bg-tertiary)] border border-[var(--border-subtle)] rounded-md text-xs hover:border-[var(--border)] transition-all duration-150 text-left"
        >
          <Folder className="w-3.5 h-3.5 text-[var(--text-muted)] shrink-0" />
          <span className="truncate text-[var(--text-secondary)] font-mono text-[11px]">
            {cwd || 'Select directory...'}
          </span>
        </button>

        {/* Connect Button */}
        <button
          onClick={handleConnect}
          disabled={!selectedAgent || !cwd}
          className={clsx(
            'w-full mt-2.5 flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-all duration-200',
            selectedAgent && cwd
              ? 'bg-[var(--accent)] hover:bg-[var(--accent-hover)] text-white shadow-glow-sm hover:shadow-glow'
              : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)] cursor-not-allowed border border-[var(--border-subtle)]'
          )}
        >
          <Zap className="w-3 h-3" />
          Connect
        </button>
      </div>

      {/* ── Connections List ── */}
      <div className="flex-1 overflow-y-auto px-2 py-2 border-t border-[var(--border-subtle)]">
        <div className="text-[9px] text-[var(--text-muted)] px-2 mb-2 uppercase tracking-widest font-medium">
          Sessions
        </div>

        {connections.length === 0 && (
          <div className="px-2 py-6 text-[11px] text-[var(--text-muted)] text-center">
            No active connections
          </div>
        )}

        {connections.map((conn) => (
          <div key={conn.id} className="mb-0.5">
            {/* Connection header */}
            <div className="flex items-center group">
              <button
                onClick={() => toggleConn(conn.id)}
                className="flex-1 flex items-center gap-1.5 px-2 py-1 rounded text-xs hover:bg-[var(--bg-tertiary)] transition-colors"
              >
                {expandedConns.has(conn.id) ? (
                  <ChevronDown className="w-3 h-3 text-[var(--text-muted)]" />
                ) : (
                  <ChevronRight className="w-3 h-3 text-[var(--text-muted)]" />
                )}
                <Bot className="w-3 h-3 text-[var(--accent)]" />
                <span className="truncate text-[var(--text-primary)]">{conn.displayName}</span>
                <div className="w-1.5 h-1.5 rounded-full ml-auto shrink-0 bg-[var(--success)]" />
              </button>
              <button
                onClick={() => handleDisconnect(conn.id)}
                className="p-1 opacity-0 group-hover:opacity-100 hover:text-[var(--error)] transition-all"
                title="Disconnect"
              >
                <X className="w-3 h-3" />
              </button>
            </div>

            {/* Sessions */}
            {expandedConns.has(conn.id) && (
              <div className="ml-5 mt-0.5 animate-fade-in">
                {(connSessions[conn.id] || []).map((session) => (
                  <button
                    key={session.id}
                    onClick={() =>
                      handleSelectSession(conn.id, session.id)
                    }
                    className={clsx(
                      'w-full flex items-center gap-1.5 px-2 py-1 rounded text-[11px] transition-all duration-150',
                      activeSession?.sessionID === session.id
                        ? 'bg-[var(--accent-muted)] text-[var(--accent)] font-medium'
                        : 'hover:bg-[var(--bg-tertiary)] text-[var(--text-secondary)]'
                    )}
                  >
                    <MessageSquare className="w-2.5 h-2.5 shrink-0" />
                    <span className="truncate">
                      {session.cwd.split('/').pop() || session.id.slice(0, 8)}
                    </span>
                    <span className="ml-auto text-[9px] opacity-40 font-mono">
                      {session.messageCount}
                    </span>
                  </button>
                ))}
                <button
                  onClick={() => handleNewSession(conn.id)}
                  className="w-full flex items-center gap-1.5 px-2 py-1 rounded text-[11px] text-[var(--text-muted)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-secondary)] transition-colors"
                >
                  <Plus className="w-2.5 h-2.5" />
                  New Session
                </button>
              </div>
            )}
          </div>
        ))}
      </div>

      {/* ── Footer ── */}
      <div className="px-3 py-1.5 border-t border-[var(--border-subtle)]">
        <button className="flex items-center gap-2 px-2 py-1 rounded text-[11px] text-[var(--text-muted)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-secondary)] transition-colors w-full">
          <Settings className="w-3.5 h-3.5" />
          Settings
        </button>
      </div>
    </div>
  );
}
