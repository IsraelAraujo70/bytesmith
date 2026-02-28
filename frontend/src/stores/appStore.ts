import { create } from 'zustand';
import type {
  AgentInfo,
  ConnectionInfo,
  MessageInfo,
  ToolCallInfo,
  PlanEntry,
  PermissionRequest,
  AvailableCommand,
  TimelineItem,
} from '../types';
import { buildTimeline } from '../lib/api';

interface ActiveSession {
  connectionID: string;
  sessionID: string;
}

interface AppState {
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

  // Working directory
  cwd: string;
  setCwd: (cwd: string) => void;

  // Messages
  messages: MessageInfo[];
  setMessages: (messages: MessageInfo[]) => void;
  addMessage: (message: MessageInfo) => void;
  appendToLastAgentMessage: (text: string) => void;

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

  // Permission requests
  permissionRequests: PermissionRequest[];
  addPermissionRequest: (req: PermissionRequest) => void;
  removePermissionRequest: (toolCallId: string) => void;

  // Loading / processing state
  loading: boolean;
  setLoading: (loading: boolean) => void;

  // Error
  error: string | null;
  setError: (error: string | null) => void;

  // Clear session data
  clearSession: () => void;
}

export const useAppStore = create<AppState>((set, get) => ({
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

  // Working directory
  cwd: '',
  setCwd: (cwd) => set({ cwd }),

  // Messages
  messages: [],
  setMessages: (messages) => set({ messages }),
  addMessage: (message) =>
    set((s) => ({ messages: [...s.messages, message] })),
  appendToLastAgentMessage: (text) =>
    set((s) => {
      const msgs = [...s.messages];
      const lastIdx = msgs.length - 1;
      if (lastIdx >= 0 && msgs[lastIdx].role === 'agent') {
        msgs[lastIdx] = {
          ...msgs[lastIdx],
          content: msgs[lastIdx].content + text,
        };
      } else {
        msgs.push({
          role: 'agent',
          content: text,
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

  // Permission requests
  permissionRequests: [],
  addPermissionRequest: (req) =>
    set((s) => ({
      permissionRequests: [...s.permissionRequests, req],
    })),
  removePermissionRequest: (toolCallId) =>
    set((s) => ({
      permissionRequests: s.permissionRequests.filter(
        (r) => r.toolCallId !== toolCallId
      ),
    })),

  // Loading
  loading: false,
  setLoading: (loading) => set({ loading }),

  // Error
  error: null,
  setError: (error) => set({ error }),

  // Clear session
  clearSession: () =>
    set({
      messages: [],
      toolCalls: [],
      plan: [],
      permissionRequests: [],
      loading: false,
      error: null,
    }),
}));
