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
  PermissionRequest,
} from '../types';

export function useWailsEvents() {
  const {
    activeSession,
    appendToLastAgentMessage,
    addToolCall,
    updateToolCall,
    setPlan,
    setCommands,
    addPermissionRequest,
    setLoading,
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
        appendToLastAgentMessage(data.text);
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
          });
        } else {
          addToolCall({
            id: data.toolCallId,
            title: data.title,
            kind: data.kind,
            status: data.status as 'pending' | 'in_progress' | 'completed' | 'failed',
            content: '',
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

    // Permission request
    EventsOn('agent:permission', (data: PermissionRequest) => {
      if (
        activeSession &&
        data.connectionId === activeSession.connectionID &&
        data.sessionId === activeSession.sessionID
      ) {
        addPermissionRequest(data);
      }
    });

    // Prompt done
    EventsOn('agent:prompt-done', (data: PromptDoneEvent) => {
      if (
        activeSession &&
        data.connectionId === activeSession.connectionID &&
        data.sessionId === activeSession.sessionID
      ) {
        setLoading(false);
      }
    });

    // Error
    EventsOn('agent:error', (data: AgentErrorEvent) => {
      if (
        activeSession &&
        data.connectionId === activeSession.connectionID &&
        data.sessionId === activeSession.sessionID
      ) {
        setError(data.error);
        setLoading(false);
      }
    });

    return () => {
      EventsOff('agent:message');
      EventsOff('agent:toolcall');
      EventsOff('agent:plan');
      EventsOff('agent:commands');
      EventsOff('agent:permission');
      EventsOff('agent:prompt-done');
      EventsOff('agent:error');
    };
  }, [
    activeSession,
    appendToLastAgentMessage,
    addToolCall,
    updateToolCall,
    setPlan,
    setCommands,
    addPermissionRequest,
    setLoading,
    setError,
  ]);
}
