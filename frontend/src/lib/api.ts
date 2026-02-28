import type {
  AgentInfo,
  ConnectionInfo,
  SessionListItem,
  MessageInfo,
  ToolCallInfo,
  AvailableCommand,
  PermissionRequest,
  PlanEntry,
  TimelineItem,
} from '../types';

// Mock API that wraps Wails Go calls with fallbacks for development.
// When the Go backend is connected, these will call the real Wails bindings.
// In dev mode (browser), they return mock data.

function isWailsAvailable(): boolean {
  return typeof window !== 'undefined' && 'go' in window;
}

async function callWails<T>(
  namespace: string,
  method: string,
  ...args: unknown[]
): Promise<T> {
  if (isWailsAvailable()) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const w = window as any;
    return w.go.main.App[method](...args);
  }
  throw new Error(`Wails not available: ${namespace}.${method}`);
}

// --- Agent Management ---

export async function listAgents(): Promise<AgentInfo[]> {
  try {
    return await callWails<AgentInfo[]>('App', 'ListAgents');
  } catch {
    return [
      {
        name: 'claude-code',
        displayName: 'Claude Code',
        command: 'claude',
        description: 'Anthropic Claude Code agent',
        installed: true,
      },
      {
        name: 'codex',
        displayName: 'OpenAI Codex',
        command: 'codex',
        description: 'OpenAI Codex CLI agent',
        installed: false,
      },
      {
        name: 'gemini-cli',
        displayName: 'Gemini CLI',
        command: 'gemini',
        description: 'Google Gemini CLI agent',
        installed: true,
      },
    ];
  }
}

export async function connectAgent(
  agentName: string,
  cwd: string
): Promise<string> {
  try {
    return await callWails<string>('App', 'ConnectAgent', agentName, cwd);
  } catch {
    return `mock-conn-${Date.now()}`;
  }
}

export async function disconnectAgent(connectionID: string): Promise<void> {
  try {
    await callWails<void>('App', 'DisconnectAgent', connectionID);
  } catch {
    // mock: no-op
  }
}

// --- Connection Management ---

export async function listConnections(): Promise<ConnectionInfo[]> {
  try {
    return await callWails<ConnectionInfo[]>('App', 'ListConnections');
  } catch {
    return [];
  }
}

// --- Session Management ---

export async function createSession(
  connectionID: string,
  cwd: string
): Promise<string> {
  try {
    return await callWails<string>('App', 'CreateSession', connectionID, cwd);
  } catch {
    return `mock-session-${Date.now()}`;
  }
}

export async function listSessions(
  connectionID: string
): Promise<SessionListItem[]> {
  try {
    return await callWails<SessionListItem[]>(
      'App',
      'ListSessions',
      connectionID
    );
  } catch {
    return [];
  }
}

export async function getSessionMessages(
  connectionID: string,
  sessionID: string
): Promise<MessageInfo[]> {
  try {
    return await callWails<MessageInfo[]>(
      'App',
      'GetSessionMessages',
      connectionID,
      sessionID
    );
  } catch {
    return [];
  }
}

export async function getSessionToolCalls(
  connectionID: string,
  sessionID: string
): Promise<ToolCallInfo[]> {
  try {
    return await callWails<ToolCallInfo[]>(
      'App',
      'GetSessionToolCalls',
      connectionID,
      sessionID
    );
  } catch {
    return [];
  }
}

// --- Prompting ---

export async function sendPrompt(
  connectionID: string,
  sessionID: string,
  text: string
): Promise<void> {
  try {
    await callWails<void>(
      'App',
      'SendPrompt',
      connectionID,
      sessionID,
      text
    );
  } catch {
    // mock: no-op, events will simulate response
  }
}

export async function cancelPrompt(
  connectionID: string,
  sessionID: string
): Promise<void> {
  try {
    await callWails<void>('App', 'CancelPrompt', connectionID, sessionID);
  } catch {
    // mock: no-op
  }
}

// --- Permissions ---

export async function respondPermission(
  connectionID: string,
  sessionID: string,
  toolCallID: string,
  optionId: string
): Promise<void> {
  try {
    await callWails<void>(
      'App',
      'RespondPermission',
      connectionID,
      sessionID,
      toolCallID,
      optionId
    );
  } catch {
    // mock: no-op
  }
}

// --- Commands ---

export async function getAvailableCommands(
  connectionID: string,
  sessionID: string
): Promise<AvailableCommand[]> {
  try {
    return await callWails<AvailableCommand[]>(
      'App',
      'GetAvailableCommands',
      connectionID,
      sessionID
    );
  } catch {
    return [
      { name: '/help', description: 'Show available commands' },
      { name: '/clear', description: 'Clear the conversation' },
      { name: '/compact', description: 'Compact the conversation' },
    ];
  }
}

// --- Directory Picker ---

export async function pickDirectory(): Promise<string> {
  try {
    return await callWails<string>('App', 'PickDirectory');
  } catch {
    return '/home/user/projects';
  }
}

// --- Utility: build timeline from messages + tool calls ---

export function buildTimeline(
  messages: MessageInfo[],
  toolCalls: ToolCallInfo[]
): TimelineItem[] {
  const items: TimelineItem[] = [];

  for (const m of messages) {
    items.push({ type: 'message', data: m });
  }
  for (const tc of toolCalls) {
    items.push({ type: 'toolcall', data: tc });
  }

  items.sort((a, b) => {
    const tA =
      a.type === 'message' ? a.data.timestamp : a.data.timestamp;
    const tB =
      b.type === 'message' ? b.data.timestamp : b.data.timestamp;
    return tA.localeCompare(tB);
  });

  return items;
}
