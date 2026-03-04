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
  integrator: string;
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

export interface SessionModelInfo {
  modelId: string;
  name: string;
}

export interface SessionModelsInfo {
  currentModelId: string;
  models: SessionModelInfo[];
}

export interface SessionModeInfo {
  modeId: string;
  name: string;
}

export interface SessionModesInfo {
  currentModeId: string;
  modes: SessionModeInfo[];
}

export type MessageKind = 'text' | 'thought';

export interface MessageInfo {
  id: string;
  role: 'user' | 'agent' | 'system';
  content: string;
  kind?: MessageKind;
  timestamp: string;
}

export interface ToolCallPartInfo {
  type: string;
  text?: string;
  path?: string;
  oldText?: string;
  newText?: string;
  terminalId?: string;
}

export interface ToolCallDiffSummaryInfo {
  additions: number;
  deletions: number;
  files: number;
}

export interface ToolCallInfo {
  id: string;
  title: string;
  kind: string;
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
  content: string;
  parts?: ToolCallPartInfo[];
  diffSummary?: ToolCallDiffSummaryInfo;
  timestamp: string;
}

export interface ResumeHistoricalResult {
  connectionId: string;
  sessionId: string;
  resumed: boolean;
  reason?: string;
}

export interface PlanEntry {
  content: string;
  priority: string;
  status: string;
}

export interface PermissionRequest {
  requestId: string;
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

export interface QuestionRequest {
  requestId: string;
  connectionId: string;
  sessionId: string;
  toolCallId: string;
  questions: QuestionItem[];
}

export interface QuestionItem {
  id: string;
  header: string;
  question: string;
  multiple: boolean;
  isOther: boolean;
  isSecret: boolean;
  options: QuestionOption[];
}

export interface QuestionOption {
  label: string;
  description: string;
}

export interface AvailableCommand {
  name: string;
  description: string;
  hint?: string;
}

export interface EmbeddedTerminalSession {
  id: string;
  cwd: string;
  shell: string;
}

export interface EmbeddedTerminalTab extends EmbeddedTerminalSession {
  buffer: string;
  exited: boolean;
  exitCode?: number;
}

// Events from Go backend — field names must match Go map keys exactly.
// Go uses camelCase JSON keys: connectionId, sessionId, toolCallId, etc.
export interface AgentMessageEvent {
  connectionId: string;
  sessionId: string;
  messageId: string;
  text: string;
  type: string;
  isFinal: boolean;
  content?: string;
}

export interface AgentToolCallEvent {
  connectionId: string;
  sessionId: string;
  toolCallId: string;
  title: string;
  kind: string;
  status: string;
  content?: string;
  parts?: ToolCallPartInfo[];
  diffSummary?: ToolCallDiffSummaryInfo;
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

export interface AgentModelsEvent {
  connectionId: string;
  sessionId: string;
  currentModelId: string;
  models: SessionModelInfo[];
}

export interface AgentModesEvent {
  connectionId: string;
  sessionId: string;
  currentModeId: string;
  modes: SessionModeInfo[];
}

export interface UITerminalOutputEvent {
  terminalId: string;
  data: string;
}

export interface UITerminalExitEvent {
  terminalId: string;
  exitCode: number;
}

// Timeline item is a union for rendering chat + tool calls in order
export type TimelineItem =
  | { type: 'message'; data: MessageInfo }
  | { type: 'toolcall'; data: ToolCallInfo };
