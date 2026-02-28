import { useEffect, useState } from 'react';
import {
  Bot,
  ChevronDown,
  ChevronRight,
  Folder,
  MessageSquare,
  Plus,
  Settings,
  Terminal,
  X,
  Wifi,
  WifiOff,
} from 'lucide-react';
import { clsx } from 'clsx';
import { useAppStore } from '../../stores/appStore';
import {
  listAgents,
  connectAgent,
  disconnectAgent,
  listConnections,
  listSessions,
  createSession,
  pickDirectory,
} from '../../lib/api';
import type { AgentInfo, ConnectionInfo, SessionListItem } from '../../types';

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
    setLoading,
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
      // Load sessions for this connection
      const sessions = await listSessions(connId);
      setConnSessions((prev) => ({ ...prev, [connId]: sessions }));
    }
    setExpandedConns(next);
  };

  const handleConnect = async () => {
    if (!selectedAgent || !cwd) return;
    setLoading(true);
    try {
      const connId = await connectAgent(selectedAgent, cwd);
      const sessionId = await createSession(connId, cwd);
      const agent = agents.find((a) => a.name === selectedAgent);
      const conn: ConnectionInfo = {
        id: connId,
        agentName: selectedAgent,
        agentDisplayName: agent?.displayName || selectedAgent,
        sessions: [sessionId],
        running: true,
      };
      setConnections([...connections, conn]);
      setActiveSession({ connectionID: connId, sessionID: sessionId });
      clearSession();
    } finally {
      setLoading(false);
    }
  };

  const handleDisconnect = async (connId: string) => {
    await disconnectAgent(connId);
    setConnections(connections.filter((c) => c.id !== connId));
    if (activeSession?.connectionID === connId) {
      setActiveSession(null);
      clearSession();
    }
  };

  const handleNewSession = async (connId: string) => {
    const sessionId = await createSession(connId, cwd);
    setActiveSession({ connectionID: connId, sessionID: sessionId });
    clearSession();
    // Refresh sessions
    const sessions = await listSessions(connId);
    setConnSessions((prev) => ({ ...prev, [connId]: sessions }));
  };

  const handleSelectSession = (connectionID: string, sessionID: string) => {
    setActiveSession({ connectionID, sessionID });
    clearSession();
  };

  const handlePickDirectory = async () => {
    const dir = await pickDirectory();
    if (dir) setCwd(dir);
  };

  const installedAgents = agents.filter((a) => a.installed);
  const selected = agents.find((a) => a.name === selectedAgent);

  return (
    <div className="flex flex-col h-full w-[280px] bg-[var(--bg-secondary)] border-r border-[var(--border)]">
      {/* Header */}
      <div className="flex items-center gap-2 px-4 py-3 border-b border-[var(--border)] wails-drag">
        <Terminal className="w-5 h-5 text-[var(--accent)]" />
        <span className="font-semibold text-sm tracking-wide">ByteSmith</span>
      </div>

      {/* Agent Selector */}
      <div className="px-3 py-3 border-b border-[var(--border)]">
        <label className="text-xs text-[var(--text-secondary)] mb-1.5 block">
          Agent
        </label>
        <div className="relative">
          <button
            onClick={() => setAgentDropdownOpen(!agentDropdownOpen)}
            className="w-full flex items-center justify-between px-3 py-2 bg-[var(--bg-tertiary)] border border-[var(--border)] rounded-lg text-sm hover:border-[var(--accent)] transition-colors"
          >
            <span className="flex items-center gap-2">
              <Bot className="w-4 h-4" />
              {selected?.displayName || 'Select agent...'}
            </span>
            <ChevronDown className="w-4 h-4 text-[var(--text-secondary)]" />
          </button>

          {agentDropdownOpen && (
            <div className="absolute z-50 top-full left-0 right-0 mt-1 bg-[var(--bg-tertiary)] border border-[var(--border)] rounded-lg shadow-xl overflow-hidden">
              {agents.map((agent) => (
                <button
                  key={agent.name}
                  disabled={!agent.installed}
                  onClick={() => {
                    setSelectedAgent(agent.name);
                    setAgentDropdownOpen(false);
                  }}
                  className={clsx(
                    'w-full flex items-center gap-2 px-3 py-2 text-sm text-left hover:bg-[var(--bg-secondary)] transition-colors',
                    !agent.installed && 'opacity-40 cursor-not-allowed',
                    agent.name === selectedAgent && 'bg-[var(--bg-secondary)]'
                  )}
                >
                  <Bot className="w-4 h-4 shrink-0" />
                  <div className="flex-1 min-w-0">
                    <div className="truncate">{agent.displayName}</div>
                    <div className="text-xs text-[var(--text-secondary)] truncate">
                      {agent.description}
                    </div>
                  </div>
                  {!agent.installed && (
                    <span className="text-[10px] text-[var(--warning)] shrink-0">
                      not installed
                    </span>
                  )}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Working Directory */}
        <label className="text-xs text-[var(--text-secondary)] mt-3 mb-1.5 block">
          Directory
        </label>
        <button
          onClick={handlePickDirectory}
          className="w-full flex items-center gap-2 px-3 py-2 bg-[var(--bg-tertiary)] border border-[var(--border)] rounded-lg text-sm hover:border-[var(--accent)] transition-colors text-left"
        >
          <Folder className="w-4 h-4 text-[var(--text-secondary)] shrink-0" />
          <span className="truncate text-[var(--text-secondary)]">
            {cwd || 'Select directory...'}
          </span>
        </button>

        {/* Connect Button */}
        <button
          onClick={handleConnect}
          disabled={!selectedAgent || !cwd}
          className={clsx(
            'w-full mt-3 flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
            selectedAgent && cwd
              ? 'bg-[var(--accent)] hover:bg-[var(--accent-hover)] text-white'
              : 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)] cursor-not-allowed'
          )}
        >
          <Wifi className="w-4 h-4" />
          Connect
        </button>
      </div>

      {/* Connections List */}
      <div className="flex-1 overflow-y-auto px-2 py-2">
        <div className="text-xs text-[var(--text-secondary)] px-2 mb-2 uppercase tracking-wider">
          Connections
        </div>

        {connections.length === 0 && (
          <div className="px-2 py-4 text-xs text-[var(--text-secondary)] text-center">
            No active connections
          </div>
        )}

        {connections.map((conn) => (
          <div key={conn.id} className="mb-1">
            {/* Connection header */}
            <div className="flex items-center group">
              <button
                onClick={() => toggleConn(conn.id)}
                className="flex-1 flex items-center gap-1.5 px-2 py-1.5 rounded-md text-sm hover:bg-[var(--bg-tertiary)] transition-colors"
              >
                {expandedConns.has(conn.id) ? (
                  <ChevronDown className="w-3.5 h-3.5 text-[var(--text-secondary)]" />
                ) : (
                  <ChevronRight className="w-3.5 h-3.5 text-[var(--text-secondary)]" />
                )}
                <Bot className="w-3.5 h-3.5 text-[var(--accent)]" />
                <span className="truncate">{conn.agentDisplayName}</span>
                <span
                  className={clsx(
                    'w-2 h-2 rounded-full ml-auto shrink-0',
                    conn.running ? 'bg-[var(--success)]' : 'bg-[var(--error)]'
                  )}
                />
              </button>
              <button
                onClick={() => handleDisconnect(conn.id)}
                className="p-1 opacity-0 group-hover:opacity-100 hover:text-[var(--error)] transition-all"
                title="Disconnect"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            </div>

            {/* Sessions */}
            {expandedConns.has(conn.id) && (
              <div className="ml-5 mt-0.5">
                {(connSessions[conn.id] || []).map((session) => (
                  <button
                    key={session.id}
                    onClick={() =>
                      handleSelectSession(conn.id, session.id)
                    }
                    className={clsx(
                      'w-full flex items-center gap-2 px-2 py-1.5 rounded-md text-xs transition-colors',
                      activeSession?.sessionID === session.id
                        ? 'bg-[var(--accent)] bg-opacity-20 text-[var(--accent-hover)]'
                        : 'hover:bg-[var(--bg-tertiary)] text-[var(--text-secondary)]'
                    )}
                  >
                    <MessageSquare className="w-3 h-3 shrink-0" />
                    <span className="truncate">
                      {session.cwd.split('/').pop() || session.id.slice(0, 8)}
                    </span>
                    <span className="ml-auto text-[10px] opacity-60">
                      {session.messageCount}
                    </span>
                  </button>
                ))}
                <button
                  onClick={() => handleNewSession(conn.id)}
                  className="w-full flex items-center gap-2 px-2 py-1.5 rounded-md text-xs text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] transition-colors"
                >
                  <Plus className="w-3 h-3" />
                  New Session
                </button>
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Footer */}
      <div className="px-3 py-2 border-t border-[var(--border)]">
        <button className="flex items-center gap-2 px-2 py-1.5 rounded-md text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] transition-colors w-full">
          <Settings className="w-4 h-4" />
          Settings
        </button>
      </div>
    </div>
  );
}
