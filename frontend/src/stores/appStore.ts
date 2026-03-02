import { create } from 'zustand';
import type {
  AgentInfo,
  ConnectionInfo,
  MessageInfo,
  MessageKind,
  ToolCallInfo,
  SessionModelInfo,
  SessionModeInfo,
  PlanEntry,
  PermissionRequest,
  AvailableCommand,
  EmbeddedTerminalSession,
  EmbeddedTerminalTab,
  TimelineItem,
} from '../types';
import { buildTimeline } from '../lib/api';
import { themes, defaultThemeId, applyTheme } from '../lib/themes';
import type { ThemeDefinition } from '../lib/themes';

interface ActiveSession {
  connectionID: string;
  sessionID: string;
}

const MAX_SESSION_NAV_HISTORY = 100;

function sessionKey(connectionID: string, sessionID: string): string {
  return `${connectionID}::${sessionID}`;
}

function shouldInsertChunkSeparator(
  existing: string,
  incoming: string,
  kind: MessageKind
): boolean {
  if (kind !== 'thought') {
    return false;
  }
  if (!existing || !incoming) {
    return false;
  }

  const tail = existing[existing.length - 1];
  const head = incoming[0];

  if (!tail || !head) {
    return false;
  }
  if (tail === '\n' || head === '\n') {
    return false;
  }
  if (/\s/.test(tail) || /\s/.test(head)) {
    return false;
  }
  if (/[.!?:;,\])}]/.test(tail)) {
    return true;
  }
  if (/[a-z0-9]/.test(tail) && /[A-Z]/.test(head)) {
    return true;
  }

  return false;
}

function mergeMessageChunk(existing: string, incoming: string, kind: MessageKind): string {
  if (!existing) {
    return incoming;
  }
  if (shouldInsertChunkSeparator(existing, incoming, kind)) {
    return `${existing}\n${incoming}`;
  }
  return existing + incoming;
}

interface AppState {
  // Theme
  themeId: string;
  theme: ThemeDefinition;
  setTheme: (themeId: string) => void;

  // Agents
  agents: AgentInfo[];
  setAgents: (agents: AgentInfo[]) => void;

  // Connections
  connections: ConnectionInfo[];
  setConnections: (connections: ConnectionInfo[]) => void;
  addConnection: (conn: ConnectionInfo) => void;
  removeConnection: (id: string) => void;

  // Active session
  activeSession: ActiveSession | null;
  setActiveSession: (session: ActiveSession | null) => void;
  sessionReadOnly: boolean;
  setSessionReadOnly: (readOnly: boolean) => void;

  // Session navigation history
  sessionNavStack: ActiveSession[];
  sessionNavIndex: number;
  visitSession: (session: ActiveSession) => void;
  goBackSession: () => ActiveSession | null;
  goForwardSession: () => ActiveSession | null;
  canGoBackSession: () => boolean;
  canGoForwardSession: () => boolean;

  // Working directory
  cwd: string;
  setCwd: (cwd: string) => void;

  // Embedded terminal
  terminalPanelOpen: boolean;
  setTerminalPanelOpen: (open: boolean) => void;
  terminals: EmbeddedTerminalTab[];
  activeTerminalId: string | null;
  addTerminal: (terminal: EmbeddedTerminalSession) => void;
  removeTerminal: (terminalId: string) => void;
  setActiveTerminal: (terminalId: string | null) => void;
  appendTerminalOutput: (terminalId: string, data: string) => void;
  markTerminalExited: (terminalId: string, exitCode: number) => void;

  // Messages
  messages: MessageInfo[];
  setMessages: (messages: MessageInfo[]) => void;
  addMessage: (message: MessageInfo) => void;
  appendAgentMessageChunk: (
    messageId: string,
    text: string,
    kind: MessageKind
  ) => void;
  finalizeAgentMessage: (
    messageId: string,
    content?: string,
    kind?: MessageKind
  ) => void;

  // Tool calls
  toolCalls: ToolCallInfo[];
  setToolCalls: (toolCalls: ToolCallInfo[]) => void;
  addToolCall: (toolCall: ToolCallInfo) => void;
  updateToolCall: (id: string, updates: Partial<ToolCallInfo>) => void;

  // Timeline (computed from messages + tool calls)
  getTimeline: () => TimelineItem[];

  // Plan
  plan: PlanEntry[];
  setPlan: (plan: PlanEntry[]) => void;

  // Commands
  commands: AvailableCommand[];
  setCommands: (commands: AvailableCommand[]) => void;

  // Session models
  models: SessionModelInfo[];
  currentModelId: string;
  setSessionModels: (models: SessionModelInfo[], currentModelId: string) => void;

  // Session modes
  modes: SessionModeInfo[];
  currentModeId: string;
  setSessionModes: (modes: SessionModeInfo[], currentModeId: string) => void;

  // Permission requests
  permissionRequests: PermissionRequest[];
  addPermissionRequest: (req: PermissionRequest) => void;
  removePermissionRequest: (requestId: string) => void;

  // Model picker modal
  modelPickerOpen: boolean;
  setModelPickerOpen: (open: boolean) => void;

  // Sidebar collapsed
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;

  // Loading / processing state
  loading: boolean;
  setLoading: (loading: boolean) => void;
  sessionLoading: Record<string, boolean>;
  setSessionLoading: (
    connectionID: string,
    sessionID: string,
    loading: boolean
  ) => void;
  isSessionLoading: (connectionID: string, sessionID: string) => boolean;

  // Error
  error: string | null;
  setError: (error: string | null) => void;

  // Clear session data
  clearSession: () => void;
}

// Resolve initial theme from localStorage or default
function getInitialTheme(): { id: string; theme: ThemeDefinition } {
  const stored = typeof localStorage !== 'undefined'
    ? localStorage.getItem('bytesmith-theme')
    : null;
  const id = stored && themes[stored] ? stored : defaultThemeId;
  return { id, theme: themes[id] };
}

const initial = getInitialTheme();

// Apply theme on load
applyTheme(initial.theme);

export const useAppStore = create<AppState>((set, get) => ({
  // Theme
  themeId: initial.id,
  theme: initial.theme,
  setTheme: (themeId) => {
    const theme = themes[themeId];
    if (!theme) return;
    applyTheme(theme);
    localStorage.setItem('bytesmith-theme', themeId);
    set({ themeId, theme });
  },

  // Agents
  agents: [],
  setAgents: (agents) => set({ agents }),

  // Connections
  connections: [],
  setConnections: (connections) => set({ connections }),
  addConnection: (conn) =>
    set((s) => ({ connections: [...s.connections, conn] })),
  removeConnection: (id) =>
    set((s) => ({
      connections: s.connections.filter((c) => c.id !== id),
    })),

  // Active session
  activeSession: null,
  setActiveSession: (session) => set({ activeSession: session }),
  sessionReadOnly: false,
  setSessionReadOnly: (sessionReadOnly) => set({ sessionReadOnly }),

  // Session navigation history
  sessionNavStack: [],
  sessionNavIndex: -1,
  visitSession: (session) =>
    set((s) => {
      const current = s.sessionNavStack[s.sessionNavIndex];
      if (
        current &&
        current.connectionID === session.connectionID &&
        current.sessionID === session.sessionID
      ) {
        return {};
      }

      const nextStack = s.sessionNavStack
        .slice(0, s.sessionNavIndex + 1)
        .concat(session);

      if (nextStack.length > MAX_SESSION_NAV_HISTORY) {
        const trimmed = nextStack.slice(nextStack.length - MAX_SESSION_NAV_HISTORY);
        return {
          sessionNavStack: trimmed,
          sessionNavIndex: trimmed.length - 1,
        };
      }

      return {
        sessionNavStack: nextStack,
        sessionNavIndex: nextStack.length - 1,
      };
    }),
  goBackSession: () => {
    const s = get();
    if (s.sessionNavIndex <= 0) {
      return null;
    }
    const nextIndex = s.sessionNavIndex - 1;
    const target = s.sessionNavStack[nextIndex] || null;
    set({ sessionNavIndex: nextIndex });
    return target;
  },
  goForwardSession: () => {
    const s = get();
    if (s.sessionNavIndex >= s.sessionNavStack.length - 1) {
      return null;
    }
    const nextIndex = s.sessionNavIndex + 1;
    const target = s.sessionNavStack[nextIndex] || null;
    set({ sessionNavIndex: nextIndex });
    return target;
  },
  canGoBackSession: () => get().sessionNavIndex > 0,
  canGoForwardSession: () =>
    get().sessionNavIndex >= 0 &&
    get().sessionNavIndex < get().sessionNavStack.length - 1,

  // Working directory
  cwd: '',
  setCwd: (cwd) => set({ cwd }),

  // Embedded terminal
  terminalPanelOpen: false,
  setTerminalPanelOpen: (terminalPanelOpen) => set({ terminalPanelOpen }),
  terminals: [],
  activeTerminalId: null,
  addTerminal: (terminal) =>
    set((s) => {
      const exists = s.terminals.some((t) => t.id === terminal.id);
      if (exists) {
        return {
          terminalPanelOpen: true,
          activeTerminalId: terminal.id,
        };
      }

      return {
        terminalPanelOpen: true,
        activeTerminalId: terminal.id,
        terminals: [
          ...s.terminals,
          {
            ...terminal,
            buffer: '',
            exited: false,
          },
        ],
      };
    }),
  removeTerminal: (terminalId) =>
    set((s) => {
      const next = s.terminals.filter((t) => t.id !== terminalId);
      let activeTerminalId = s.activeTerminalId;
      if (activeTerminalId === terminalId) {
        activeTerminalId = next.length > 0 ? next[next.length - 1].id : null;
      }

      return {
        terminals: next,
        activeTerminalId,
        terminalPanelOpen: next.length > 0 ? s.terminalPanelOpen : false,
      };
    }),
  setActiveTerminal: (activeTerminalId) => set({ activeTerminalId }),
  appendTerminalOutput: (terminalId, data) =>
    set((s) => ({
      terminals: s.terminals.map((t) =>
        t.id === terminalId
          ? {
              ...t,
              buffer: t.buffer + data,
            }
          : t
      ),
    })),
  markTerminalExited: (terminalId, exitCode) =>
    set((s) => ({
      terminals: s.terminals.map((t) =>
        t.id === terminalId
          ? {
              ...t,
              exited: true,
              exitCode,
            }
          : t
      ),
    })),

  // Messages
  messages: [],
  setMessages: (messages) => set({ messages }),
  addMessage: (message) =>
    set((s) => ({ messages: [...s.messages, message] })),
  appendAgentMessageChunk: (messageId, text, kind) =>
    set((s) => {
      const msgs = [...s.messages];
      const idx = msgs.findIndex((m) => m.id === messageId);
      if (idx >= 0) {
        msgs[idx] = {
          ...msgs[idx],
          content: mergeMessageChunk(msgs[idx].content, text, kind),
          kind: msgs[idx].kind ?? kind,
        };
      } else {
        msgs.push({
          id: messageId,
          role: 'agent',
          content: text,
          kind,
          timestamp: new Date().toISOString(),
        });
      }
      return { messages: msgs };
    }),
  finalizeAgentMessage: (messageId, content, kind = 'text') =>
    set((s) => {
      if (content === undefined) {
        return {};
      }
      const msgs = [...s.messages];
      const idx = msgs.findIndex((m) => m.id === messageId);
      if (idx >= 0) {
        msgs[idx] = {
          ...msgs[idx],
          content,
          kind: msgs[idx].kind ?? kind,
        };
      } else {
        msgs.push({
          id: messageId,
          role: 'agent',
          content,
          kind,
          timestamp: new Date().toISOString(),
        });
      }
      return { messages: msgs };
    }),

  // Tool calls
  toolCalls: [],
  setToolCalls: (toolCalls) => set({ toolCalls }),
  addToolCall: (toolCall) =>
    set((s) => ({ toolCalls: [...s.toolCalls, toolCall] })),
  updateToolCall: (id, updates) =>
    set((s) => ({
      toolCalls: s.toolCalls.map((tc) =>
        tc.id === id ? { ...tc, ...updates } : tc
      ),
    })),

  // Timeline
  getTimeline: () => {
    const { messages, toolCalls } = get();
    return buildTimeline(messages, toolCalls);
  },

  // Plan
  plan: [],
  setPlan: (plan) => set({ plan }),

  // Commands
  commands: [],
  setCommands: (commands) => set({ commands }),

  // Session models
  models: [],
  currentModelId: '',
  setSessionModels: (models, currentModelId) => set({ models, currentModelId }),

  // Session modes
  modes: [],
  currentModeId: '',
  setSessionModes: (modes, currentModeId) => set({ modes, currentModeId }),

  // Permission requests
  permissionRequests: [],
  addPermissionRequest: (req) =>
    set((s) => ({
      permissionRequests: [...s.permissionRequests, req],
    })),
  removePermissionRequest: (requestId) =>
    set((s) => ({
      permissionRequests: s.permissionRequests.filter(
        (r) => r.requestId !== requestId
      ),
    })),

  // Sidebar
  sidebarCollapsed: false,
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),

  // Model picker
  modelPickerOpen: false,
  setModelPickerOpen: (open) => set({ modelPickerOpen: open }),

  // Loading
  loading: false,
  setLoading: (loading) => set({ loading }),
  sessionLoading: {},
  setSessionLoading: (connectionID, sessionID, loading) =>
    set((s) => {
      const key = sessionKey(connectionID, sessionID);
      const next = { ...s.sessionLoading };
      if (loading) {
        next[key] = true;
      } else {
        delete next[key];
      }
      return { sessionLoading: next };
    }),
  isSessionLoading: (connectionID, sessionID) => {
    const key = sessionKey(connectionID, sessionID);
    return Boolean(get().sessionLoading[key]);
  },

  // Error
  error: null,
  setError: (error) => set({ error }),

  // Clear session
  clearSession: () =>
    set({
      messages: [],
      toolCalls: [],
      plan: [],
      models: [],
      currentModelId: '',
      modes: [],
      currentModeId: '',
      sessionReadOnly: false,
      loading: false,
      error: null,
    }),
}));
