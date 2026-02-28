import type {
  AgentInfo,
  ConnectionInfo,
  SessionListItem,
  MessageInfo,
  ToolCallInfo,
  AvailableCommand,
  TimelineItem,
} from '../types';

// API layer that wraps Wails Go backend calls.
// When the Go backend is connected (running inside the Wails desktop app),
// calls go directly to the real Go methods. In dev mode (plain browser),
// calls throw so the UI can handle the "not connected" state gracefully.

function isWailsAvailable(): boolean {
  return typeof window !== 'undefined' && 'go' in window;
}

async function callWails<T>(
  method: string,
  ...args: unknown[]
): Promise<T> {
  if (isWailsAvailable()) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const w = window as any;
    return w.go.main.App[method](...args);
  }
  throw new Error(`Wails not available: App.${method}`);
}

// --- Agent Management ---

export async function listAgents(): Promise<AgentInfo[]> {
  try {
    return await callWails<AgentInfo[]>('ListAvailableAgents');
  } catch {
    // Dev fallback: show well-known agents matching the Go backend
    return [
      {
        name: 'opencode',
        displayName: 'OpenCode',
        command: 'opencode',
        description: 'OpenCode ACP agent',
        installed: false,
      },
      {
        name: 'claude-code-acp',
        displayName: 'Claude Code',
        command: 'claude-code-acp',
        description: 'Anthropic Claude Code with ACP support',
        installed: false,
      },
      {
        name: 'codex-acp',
        displayName: 'Codex CLI',
        command: 'codex-acp',
        description: 'OpenAI Codex CLI with ACP support',
        installed: false,
      },
      {
        name: 'gemini',
        displayName: 'Gemini CLI',
        command: 'gemini',
        description: 'Google Gemini CLI with ACP support',
        installed: false,
      },
    ];
  }
}

export async function connectAgent(
  agentName: string,
  cwd: string
): Promise<string> {
  return await callWails<string>('ConnectAgent', agentName, cwd);
}

export async function disconnectAgent(connectionID: string): Promise<void> {
  await callWails<void>('DisconnectAgent', connectionID);
}

// --- Connection Management ---

export async function listConnections(): Promise<ConnectionInfo[]> {
  try {
    return await callWails<ConnectionInfo[]>('ListConnections');
  } catch {
    return [];
  }
}

// --- Session Management ---

export async function createSession(
  connectionID: string,
  cwd: string
): Promise<string> {
  return await callWails<string>('NewSession', connectionID, cwd);
}

export async function listSessions(
  connectionID?: string
): Promise<SessionListItem[]> {
  try {
    const sessions = await callWails<SessionListItem[]>('ListSessions');
    if (!connectionID) {
      return sessions;
    }
    return sessions.filter((s) => s.connectionId === connectionID);
  } catch {
    return [];
  }
}

export async function getSessionHistory(
  sessionID: string
): Promise<{
  messages: MessageInfo[];
  toolCalls: ToolCallInfo[];
} | null> {
  try {
    const history = await callWails<{
      messages: MessageInfo[];
      toolCalls: ToolCallInfo[];
    } | null>('GetSessionHistory', sessionID);
    return history;
  } catch {
    return null;
  }
}

// --- Prompting ---

export async function sendPrompt(
  connectionID: string,
  sessionID: string,
  text: string
): Promise<void> {
  await callWails<void>('SendPrompt', connectionID, sessionID, text);
}

export async function cancelPrompt(
  connectionID: string,
  sessionID: string
): Promise<void> {
  await callWails<void>('CancelPrompt', connectionID, sessionID);
}

// --- Permissions ---

// Go's RespondPermission takes (connectionID, optionID) — only 2 args.
// The session/toolCall context is already tracked server-side via the
// pending permissions channel keyed by connectionID.
export async function respondPermission(
  connectionID: string,
  optionID: string
): Promise<void> {
  await callWails<void>('RespondPermission', connectionID, optionID);
}

// --- Directory Picker ---

export async function pickDirectory(): Promise<string> {
  return await callWails<string>('SelectDirectory');
}

// --- Settings ---

export async function getSettings(): Promise<{
  theme: string;
  defaultAgent: string;
  defaultCwd: string;
  autoApprove: boolean;
}> {
  return await callWails('GetSettings');
}

export async function saveSettings(settings: {
  theme: string;
  defaultAgent: string;
  defaultCwd: string;
  autoApprove: boolean;
}): Promise<void> {
  await callWails<void>('SaveSettings', settings);
}

// --- File System ---

export async function listFiles(
  dir: string
): Promise<{ name: string; path: string; isDir: boolean; size: number }[]> {
  return await callWails('ListFiles', dir);
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
    const tA = a.type === 'message' ? a.data.timestamp : a.data.timestamp;
    const tB = b.type === 'message' ? b.data.timestamp : b.data.timestamp;
    return tA.localeCompare(tB);
  });

  return items;
}
