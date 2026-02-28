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
  agentDisplayName: string;
  sessions: string[];
  running: boolean;
}

export interface SessionListItem {
  id: string;
  agentName: string;
  connectionID: string;
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
  connectionID: string;
  sessionID: string;
  toolCallID: string;
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

// Events from Go backend
export interface AgentMessageEvent {
  connectionID: string;
  sessionID: string;
  text: string;
}

export interface AgentToolCallEvent {
  connectionID: string;
  sessionID: string;
  toolCallID: string;
  title: string;
  kind: string;
  status: string;
  content: string;
}

export interface AgentPlanEvent {
  connectionID: string;
  sessionID: string;
  entries: PlanEntry[];
}

export interface PromptDoneEvent {
  connectionID: string;
  sessionID: string;
  stopReason: string;
}

// Timeline item is a union for rendering chat + tool calls in order
export type TimelineItem =
  | { type: 'message'; data: MessageInfo }
  | { type: 'toolcall'; data: ToolCallInfo };
