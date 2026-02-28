import { useEffect } from 'react';
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';
import { useAppStore } from '../stores/appStore';
import type {
  AgentMessageEvent,
  AgentToolCallEvent,
  AgentPlanEvent,
  PromptDoneEvent,
  PermissionRequest,
  AvailableCommand,
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
        data.connectionID === activeSession.connectionID &&
        data.sessionID === activeSession.sessionID
      ) {
        appendToLastAgentMessage(data.text);
      }
    });

    // Tool call updates
    EventsOn('agent:toolcall', (data: AgentToolCallEvent) => {
      if (
        activeSession &&
        data.connectionID === activeSession.connectionID &&
        data.sessionID === activeSession.sessionID
      ) {
        const existing = useAppStore
          .getState()
          .toolCalls.find((tc) => tc.id === data.toolCallID);
        if (existing) {
          updateToolCall(data.toolCallID, {
            status: data.status as 'pending' | 'in_progress' | 'completed' | 'failed',
            content: data.content,
          });
        } else {
          addToolCall({
            id: data.toolCallID,
            title: data.title,
            kind: data.kind,
            status: data.status as 'pending' | 'in_progress' | 'completed' | 'failed',
            content: data.content,
            timestamp: new Date().toISOString(),
          });
        }
      }
    });

    // Plan updates
    EventsOn('agent:plan', (data: AgentPlanEvent) => {
      if (
        activeSession &&
        data.connectionID === activeSession.connectionID &&
        data.sessionID === activeSession.sessionID
      ) {
        setPlan(data.entries);
      }
    });

    // Available commands
    EventsOn('agent:commands', (data: { commands: AvailableCommand[] }) => {
      setCommands(data.commands);
    });

    // Permission request
    EventsOn('agent:permission', (data: PermissionRequest) => {
      if (
        activeSession &&
        data.connectionID === activeSession.connectionID &&
        data.sessionID === activeSession.sessionID
      ) {
        addPermissionRequest(data);
      }
    });

    // Prompt done
    EventsOn('agent:prompt-done', (data: PromptDoneEvent) => {
      if (
        activeSession &&
        data.connectionID === activeSession.connectionID &&
        data.sessionID === activeSession.sessionID
      ) {
        setLoading(false);
      }
    });

    // Error
    EventsOn('agent:error', (data: { message: string }) => {
      setError(data.message);
      setLoading(false);
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
