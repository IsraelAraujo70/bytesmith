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
  MessageKind,
  PermissionRequest,
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
    addPermissionRequest,
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
          });
        } else {
          addToolCall({
            id: data.toolCallId,
            title: data.title,
            kind: data.kind,
            status: data.status as 'pending' | 'in_progress' | 'completed' | 'failed',
            content: data.content || '',
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

    // Permission request
    EventsOn('agent:permission', (data: PermissionRequest) => {
      addPermissionRequest(data);
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

    return () => {
      EventsOff('agent:message');
      EventsOff('agent:toolcall');
      EventsOff('agent:plan');
      EventsOff('agent:commands');
      EventsOff('agent:models');
      EventsOff('agent:permission');
      EventsOff('agent:prompt-done');
      EventsOff('agent:error');
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
    addPermissionRequest,
    setSessionLoading,
    setError,
  ]);
}
