import {
  getSessionHistory,
  getSessionModels,
  getSessionModes,
  listConnections,
  resumeHistoricalSession,
} from './api';
import type { ConnectionInfo, MessageInfo, ToolCallInfo } from '../types';

export interface SessionRef {
  connectionID: string;
  sessionID: string;
}

interface OpenSessionStoreActions {
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  setConnections: (connections: ConnectionInfo[]) => void;
  setActiveSession: (session: SessionRef | null) => void;
  setSessionReadOnly: (readOnly: boolean) => void;
  clearSession: () => void;
  setMessages: (messages: MessageInfo[]) => void;
  setToolCalls: (toolCalls: ToolCallInfo[]) => void;
  setSessionModels: (models: { modelId: string; name: string }[], currentModelId: string) => void;
  setSessionModes: (modes: { modeId: string; name: string }[], currentModeId: string) => void;
  visitSession: (session: SessionRef) => void;
}

interface OpenSessionOptions {
  ensureConnected?: boolean;
  trackHistory?: boolean;
}

export async function openSessionView(
  store: OpenSessionStoreActions,
  target: SessionRef,
  options: OpenSessionOptions = {},
): Promise<SessionRef | null> {
  const ensureConnected = options.ensureConnected === true;
  const trackHistory = options.trackHistory !== false;

  store.setLoading(true);

  try {
    let resolvedConnectionID = target.connectionID;
    let resumed = true;
    let resumeReason = '';

    if (ensureConnected) {
      const connections = await listConnections();
      store.setConnections(connections);

      const stillConnected = connections.some((c) => c.id === target.connectionID);
      if (!stillConnected) {
        const resume = await resumeHistoricalSession(target.sessionID);
        resumed = resume.resumed;
        resumeReason = resume.reason || '';
        if (resume.connectionId) {
          resolvedConnectionID = resume.connectionId;
        }

        const refreshed = await listConnections();
        store.setConnections(refreshed);
      }
    }

    const resolved: SessionRef = {
      connectionID: resolvedConnectionID,
      sessionID: target.sessionID,
    };

    store.setActiveSession(resolved);
    store.clearSession();

    const [history, modelsInfo, modesInfo] = await Promise.all([
      getSessionHistory(target.sessionID),
      getSessionModels(target.sessionID),
      getSessionModes(target.sessionID),
    ]);

    if (history) {
      store.setMessages(history.messages || []);
      store.setToolCalls(history.toolCalls || []);
    }

    if (modelsInfo) {
      store.setSessionModels(modelsInfo.models, modelsInfo.currentModelId);
    }

    if (modesInfo) {
      store.setSessionModes(modesInfo.modes, modesInfo.currentModeId);
    }

    store.setSessionReadOnly(!resumed);

    if (!resumed && resumeReason) {
      store.setError(`Sessao aberta em somente leitura: ${resumeReason}`);
    }

    if (trackHistory) {
      store.visitSession(resolved);
    }

    return resolved;
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    store.setError(`Falha ao carregar sessao: ${message}`);
    return null;
  } finally {
    store.setLoading(false);
  }
}
