export interface AgentInfo {
  name: string;
  displayName: string;
  command: string;
  description: string;
  installed: boolean;
}

export interface ConnectionInfo {
  id: string;
  agentName: string;
  displayName: string;
  sessions: string[];
}

export interface SessionListItem {
  id: string;
  agentName: string;
  connectionId: string;
  cwd: string;
  messageCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface MessageInfo {
  role: 'user' | 'agent' | 'system';
  content: string;
  timestamp: string;
}

export interface ToolCallInfo {
  id: string;
  title: string;
  kind: string;
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
  content: string;
  timestamp: string;
}

export interface PlanEntry {
  content: string;
  priority: string;
  status: string;
}

export interface PermissionRequest {
  connectionId: string;
  sessionId: string;
  toolCallId: string;
  title: string;
  kind: string;
  options: PermissionOption[];
}

export interface PermissionOption {
  optionId: string;
  name: string;
  kind: string;
}

export interface AvailableCommand {
  name: string;
  description: string;
  hint?: string;
}

// Events from Go backend — field names must match Go map keys exactly.
// Go uses camelCase JSON keys: connectionId, sessionId, toolCallId, etc.
export interface AgentMessageEvent {
  connectionId: string;
  sessionId: string;
  text: string;
  type: string;
}

export interface AgentToolCallEvent {
  connectionId: string;
  sessionId: string;
  toolCallId: string;
  title: string;
  kind: string;
  status: string;
  isUpdate: boolean;
}

export interface AgentPlanEvent {
  connectionId: string;
  sessionId: string;
  entries: PlanEntry[];
}

export interface AgentCommandsEvent {
  connectionId: string;
  sessionId: string;
  commands: AvailableCommand[];
}

export interface PromptDoneEvent {
  connectionId: string;
  sessionId: string;
  stopReason: string;
}

export interface AgentErrorEvent {
  connectionId: string;
  sessionId: string;
  error: string;
}

// Timeline item is a union for rendering chat + tool calls in order
export type TimelineItem =
  | { type: 'message'; data: MessageInfo }
  | { type: 'toolcall'; data: ToolCallInfo };
