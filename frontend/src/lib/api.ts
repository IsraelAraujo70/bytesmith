import type {
  AgentInfo,
  ConnectionInfo,
  SessionListItem,
  SessionModelsInfo,
  SessionModesInfo,
  ResumeHistoricalResult,
  MessageInfo,
  ToolCallInfo,
  AvailableCommand,
  EmbeddedTerminalSession,
  TimelineItem,
} from "../types";

// API layer that wraps Wails Go backend calls.
// When the Go backend is connected (running inside the Wails desktop app),
// calls go directly to the real Go methods. In dev mode (plain browser),
// calls throw so the UI can handle the "not connected" state gracefully.

function isWailsAvailable(): boolean {
  return typeof window !== "undefined" && "go" in window;
}

async function callWails<T>(method: string, ...args: unknown[]): Promise<T> {
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
    return await callWails<AgentInfo[]>("ListAvailableAgents");
  } catch {
    // Dev fallback: show well-known agents matching the Go backend
    return [
      {
        name: "opencode",
        displayName: "OpenCode",
        command: "opencode",
        description: "OpenCode ACP agent",
        installed: false,
      },
      {
        name: "codex-app-server",
        displayName: "Codex App Server",
        command: "codex",
        description: "OpenAI Codex app-server (compat mode)",
        installed: false,
      },
    ];
  }
}

export async function connectAgent(
  agentName: string,
  cwd: string,
): Promise<string> {
  return await callWails<string>("ConnectAgent", agentName, cwd);
}

export async function disconnectAgent(connectionID: string): Promise<void> {
  await callWails<void>("DisconnectAgent", connectionID);
}

// --- Connection Management ---

export async function listConnections(): Promise<ConnectionInfo[]> {
  try {
    return await callWails<ConnectionInfo[]>("ListConnections");
  } catch {
    return [];
  }
}

// --- Session Management ---

export async function createSession(
  connectionID: string,
  cwd: string,
): Promise<string> {
  return await callWails<string>("NewSession", connectionID, cwd);
}

export async function listSessions(
  connectionID?: string,
): Promise<SessionListItem[]> {
  try {
    const sessions = await callWails<SessionListItem[]>("ListSessions");
    if (!connectionID) {
      return sessions;
    }
    return sessions.filter((s) => s.connectionId === connectionID);
  } catch {
    return [];
  }
}

export async function listRemoteSessions(
  connectionID: string,
  cwd: string,
  cursor = "",
): Promise<{ sessions: SessionListItem[]; nextCursor?: string; unsupported?: boolean }> {
  return await callWails("ListRemoteSessions", connectionID, cwd, cursor);
}

export async function loadRemoteSession(
  connectionID: string,
  sessionID: string,
  cwd: string,
): Promise<void> {
  await callWails<void>("LoadRemoteSession", connectionID, sessionID, cwd);
}

export async function resumeSession(
  connectionID: string,
  sessionID: string,
  cwd: string,
): Promise<void> {
  await callWails<void>("ResumeSession", connectionID, sessionID, cwd);
}

export async function resumeHistoricalSession(
  sessionID: string,
): Promise<ResumeHistoricalResult> {
  return await callWails<ResumeHistoricalResult>("ResumeHistoricalSession", sessionID);
}

export async function getSessionHistory(sessionID: string): Promise<{
  messages: MessageInfo[];
  toolCalls: ToolCallInfo[];
} | null> {
  try {
    const history = await callWails<{
      messages: MessageInfo[];
      toolCalls: ToolCallInfo[];
    } | null>("GetSessionHistory", sessionID);
    return history;
  } catch {
    return null;
  }
}

export async function getSessionModels(
  sessionID: string,
): Promise<SessionModelsInfo | null> {
  try {
    return await callWails<SessionModelsInfo | null>(
      "GetSessionModels",
      sessionID,
    );
  } catch {
    return null;
  }
}

export async function getSessionModes(
  sessionID: string,
): Promise<SessionModesInfo | null> {
  try {
    return await callWails<SessionModesInfo | null>(
      "GetSessionModes",
      sessionID,
    );
  } catch {
    return null;
  }
}

export async function setSessionModel(
  connectionID: string,
  sessionID: string,
  modelID: string,
): Promise<void> {
  await callWails<void>("SetSessionModel", connectionID, sessionID, modelID);
}

export async function setSessionMode(
  connectionID: string,
  sessionID: string,
  modeID: string,
): Promise<void> {
  await callWails<void>("SetSessionMode", connectionID, sessionID, modeID);
}

export async function setSessionConfigOption(
  connectionID: string,
  sessionID: string,
  configID: string,
  value: string,
): Promise<void> {
  await callWails<void>(
    "SetSessionConfigOption",
    connectionID,
    sessionID,
    configID,
    value
  );
}

// --- Prompting ---

export async function sendPrompt(
  connectionID: string,
  sessionID: string,
  text: string,
): Promise<void> {
  await callWails<void>("SendPrompt", connectionID, sessionID, text);
}

export async function cancelPrompt(
  connectionID: string,
  sessionID: string,
): Promise<void> {
  await callWails<void>("CancelPrompt", connectionID, sessionID);
}

// --- Permissions ---

export async function respondPermission(
  sessionID: string,
  toolCallID: string,
  optionID: string,
): Promise<void> {
  await callWails<void>("RespondPermission", sessionID, toolCallID, optionID);
}

// --- Directory Picker ---

export async function pickDirectory(): Promise<string> {
  return await callWails<string>("SelectDirectory");
}

// --- Settings ---

export async function getSettings(): Promise<{
  theme: string;
  defaultAgent: string;
  defaultCwd: string;
  autoApprove: boolean;
}> {
  return await callWails("GetSettings");
}

export async function saveSettings(settings: {
  theme: string;
  defaultAgent: string;
  defaultCwd: string;
  autoApprove: boolean;
}): Promise<void> {
  await callWails<void>("SaveSettings", settings);
}

// --- File System ---

export async function listFiles(
  dir: string,
): Promise<{ name: string; path: string; isDir: boolean; size: number }[]> {
  return await callWails("ListFiles", dir);
}

// --- Embedded terminal ---

export async function createEmbeddedTerminal(
  cwd: string,
): Promise<EmbeddedTerminalSession> {
  return await callWails<EmbeddedTerminalSession>("CreateEmbeddedTerminal", cwd);
}

export async function writeEmbeddedTerminal(
  terminalID: string,
  data: string,
): Promise<void> {
  await callWails<void>("WriteEmbeddedTerminal", terminalID, data);
}

export async function resizeEmbeddedTerminal(
  terminalID: string,
  cols: number,
  rows: number,
): Promise<void> {
  await callWails<void>("ResizeEmbeddedTerminal", terminalID, cols, rows);
}

export async function closeEmbeddedTerminal(terminalID: string): Promise<void> {
  await callWails<void>("CloseEmbeddedTerminal", terminalID);
}

// --- Utility: build timeline from messages + tool calls ---

export function buildTimeline(
  messages: MessageInfo[],
  toolCalls: ToolCallInfo[],
): TimelineItem[] {
  const items: TimelineItem[] = [];

  for (const m of messages) {
    items.push({ type: "message", data: m });
  }
  for (const tc of toolCalls) {
    items.push({ type: "toolcall", data: tc });
  }

  items.sort((a, b) => {
    const tA = a.type === "message" ? a.data.timestamp : a.data.timestamp;
    const tB = b.type === "message" ? b.data.timestamp : b.data.timestamp;
    const byTime = tA.localeCompare(tB);
    if (byTime !== 0) {
      return byTime;
    }

    if (a.type !== b.type) {
      return a.type === "message" ? -1 : 1;
    }

    const idA = a.type === "message" ? a.data.id : a.data.id;
    const idB = b.type === "message" ? b.data.id : b.data.id;
    return idA.localeCompare(idB);
  });

  return items;
}
