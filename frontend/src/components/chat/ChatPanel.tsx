import { useEffect, useRef } from 'react';
import { Loader2, Flame } from 'lucide-react';
import { useAppStore } from '../../stores/appStore';
import { MessageBubble } from './MessageBubble';
import { ToolCallCard } from './ToolCallCard';
import { PlanView } from './PlanView';
import { PromptInput } from './PromptInput';
import { PermissionDialog } from './PermissionDialog';
import type { TimelineItem } from '../../types';

export function ChatPanel() {
  const {
    messages,
    toolCalls,
    plan,
    activeSession,
    isSessionLoading,
    getTimeline,
  } = useAppStore();
  const scrollRef = useRef<HTMLDivElement>(null);

  const loading = activeSession
    ? isSessionLoading(activeSession.connectionID, activeSession.sessionID)
    : false;

  const timeline = getTimeline();

  // Auto-scroll to bottom on new content
  useEffect(() => {
    const el = scrollRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, [messages, toolCalls]);

  return (
    <div className="flex flex-col h-full">
      {/* Messages area */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto py-3">
        {timeline.length === 0 && !loading && (
          <div className="flex flex-col items-center justify-center h-full text-[var(--text-muted)] text-xs">
            <Flame className="w-5 h-5 mb-2 opacity-30" />
            <span>Start a conversation...</span>
          </div>
        )}

        {timeline.map((item, i) => (
          <TimelineItemRenderer
            key={itemKey(item, i)}
            item={item}
            showToolGroupHeader={
              item.type === 'toolcall' &&
              (i === 0 || timeline[i - 1]?.type !== 'toolcall')
            }
          />
        ))}

        {/* Plan */}
        {plan.length > 0 && <PlanView entries={plan} />}

        {/* Loading indicator */}
        {loading && (
          <div className="flex items-center gap-2.5 px-5 py-3 animate-fade-in">
            <div className="w-6 h-6 rounded-md bg-[var(--accent-muted)] flex items-center justify-center">
              <Loader2 className="w-3.5 h-3.5 text-[var(--accent)] animate-spin" />
            </div>
            <span className="text-[11px] text-[var(--text-muted)] animate-pulse-ember">
              Forging response...
            </span>
          </div>
        )}
      </div>

      {/* Input */}
      <PromptInput />

      {/* Permission modal */}
      <PermissionDialog />
    </div>
  );
}

function TimelineItemRenderer({
  item,
  showToolGroupHeader,
}: {
  item: TimelineItem;
  showToolGroupHeader: boolean;
}) {
  if (item.type === 'message') {
    return <MessageBubble message={item.data} />;
  }
  return (
    <>
      {showToolGroupHeader && (
        <div className="px-5 pt-2 pb-0.5">
          <div className="text-[9px] uppercase tracking-wide text-[var(--text-muted)]">
            Tools
          </div>
        </div>
      )}
      <ToolCallCard toolCall={item.data} />
    </>
  );
}

function itemKey(item: TimelineItem, index: number): string {
  if (item.type === 'message') {
    return `msg-${item.data.id || index}-${item.data.timestamp}`;
  }
  return `tc-${item.data.id}`;
}
