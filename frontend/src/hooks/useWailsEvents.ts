import { useEffect } from 'react';
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';
import { useAppStore } from '../stores/appStore';
import type {
  AgentMessageEvent,
  AgentToolCallEvent,
  AgentPlanEvent,
  AgentCommandsEvent,
  PromptDoneEvent,
  AgentErrorEvent,
  AgentModelsEvent,
  AgentModesEvent,
  MessageKind,
  PermissionRequest,
  QuestionRequest,
  UITerminalOutputEvent,
  UITerminalExitEvent,
} from '../types';

function mapMessageKind(type: string): MessageKind {
  const normalized = (type || '').trim().toLowerCase();
  if (normalized === 'thought' || normalized === 'reasoning') {
    return 'thought';
  }
  return 'text';
}

export function useWailsEvents() {
  const {
    activeSession,
    appendAgentMessageChunk,
    finalizeAgentMessage,
    addToolCall,
    updateToolCall,
    setPlan,
    setCommands,
    setSessionModels,
    setSessionModes,
    setSessionAccessModes,
    addPermissionRequest,
    addQuestionRequest,
    appendTerminalOutput,
    markTerminalExited,
    setSessionLoading,
    setError,
  } = useAppStore();

  useEffect(() => {
    // Agent streaming message chunks
    EventsOn('agent:message', (data: AgentMessageEvent) => {
      if (
        activeSession &&
        data.connectionId === activeSession.connectionID &&
        data.sessionId === activeSession.sessionID
      ) {
        const kind = mapMessageKind(data.type);
        if (data.text) {
          appendAgentMessageChunk(data.messageId, data.text, kind);
        }
        if (data.isFinal) {
          finalizeAgentMessage(data.messageId, data.content, kind);
        }
      }
    });

    // Tool call updates
    EventsOn('agent:toolcall', (data: AgentToolCallEvent) => {
      if (
        activeSession &&
        data.connectionId === activeSession.connectionID &&
        data.sessionId === activeSession.sessionID
      ) {
        if (data.isUpdate) {
          updateToolCall(data.toolCallId, {
            status: data.status as 'pending' | 'in_progress' | 'completed' | 'failed',
            ...(data.content !== undefined ? { content: data.content } : {}),
            ...(data.parts !== undefined ? { parts: data.parts } : {}),
            ...(data.diffSummary !== undefined ? { diffSummary: data.diffSummary } : {}),
          });
        } else {
          addToolCall({
            id: data.toolCallId,
            title: data.title,
            kind: data.kind,
            status: data.status as 'pending' | 'in_progress' | 'completed' | 'failed',
            content: data.content || '',
            parts: data.parts,
            diffSummary: data.diffSummary,
            timestamp: new Date().toISOString(),
          });
        }
      }
    });

    // Plan updates
    EventsOn('agent:plan', (data: AgentPlanEvent) => {
      if (
        activeSession &&
        data.connectionId === activeSession.connectionID &&
        data.sessionId === activeSession.sessionID
      ) {
        setPlan(data.entries);
      }
    });

    // Available commands
    EventsOn('agent:commands', (data: AgentCommandsEvent) => {
      if (
        activeSession &&
        data.connectionId === activeSession.connectionID &&
        data.sessionId === activeSession.sessionID
      ) {
        setCommands(data.commands);
      }
    });

    // Session models
    EventsOn('agent:models', (data: AgentModelsEvent) => {
      if (
        !activeSession ||
        (data.connectionId === activeSession.connectionID &&
          data.sessionId === activeSession.sessionID)
      ) {
        setSessionModels(data.models, data.currentModelId);
      }
    });

    // Session modes
    EventsOn('agent:modes', (data: AgentModesEvent) => {
      if (
        !activeSession ||
        (data.connectionId === activeSession.connectionID &&
          data.sessionId === activeSession.sessionID)
      ) {
        setSessionModes(data.modes, data.currentModeId);
      }
    });

    // Session access modes
    EventsOn('agent:access-modes', (data: AgentModesEvent) => {
      if (
        !activeSession ||
        (data.connectionId === activeSession.connectionID &&
          data.sessionId === activeSession.sessionID)
      ) {
        setSessionAccessModes(data.modes, data.currentModeId);
      }
    });

    // Permission request
    EventsOn('agent:permission', (data: PermissionRequest) => {
      addPermissionRequest(data);
    });

    // Explicit user-input question request
    EventsOn('agent:question', (data: QuestionRequest) => {
      addQuestionRequest(data);
    });

    // Prompt done
    EventsOn('agent:prompt-done', (data: PromptDoneEvent) => {
      setSessionLoading(data.connectionId, data.sessionId, false);
    });

    // Error
    EventsOn('agent:error', (data: AgentErrorEvent) => {
      if (
        activeSession &&
        data.connectionId === activeSession.connectionID &&
        data.sessionId === activeSession.sessionID
      ) {
        setError(data.error);
      }
      setSessionLoading(data.connectionId, data.sessionId, false);
    });

    // Embedded terminal streaming output
    EventsOn('ui:terminal-output', (data: UITerminalOutputEvent) => {
      appendTerminalOutput(data.terminalId, data.data || '');
    });

    // Embedded terminal lifecycle
    EventsOn('ui:terminal-exit', (data: UITerminalExitEvent) => {
      markTerminalExited(data.terminalId, data.exitCode);
    });

    return () => {
      EventsOff('agent:message');
      EventsOff('agent:toolcall');
      EventsOff('agent:plan');
      EventsOff('agent:commands');
      EventsOff('agent:models');
      EventsOff('agent:modes');
      EventsOff('agent:access-modes');
      EventsOff('agent:permission');
      EventsOff('agent:question');
      EventsOff('agent:prompt-done');
      EventsOff('agent:error');
      EventsOff('ui:terminal-output');
      EventsOff('ui:terminal-exit');
    };
  }, [
    activeSession,
    appendAgentMessageChunk,
    finalizeAgentMessage,
    addToolCall,
    updateToolCall,
    setPlan,
    setCommands,
    setSessionModels,
    setSessionModes,
    setSessionAccessModes,
    addPermissionRequest,
    addQuestionRequest,
    appendTerminalOutput,
    markTerminalExited,
    setSessionLoading,
    setError,
  ]);
}
