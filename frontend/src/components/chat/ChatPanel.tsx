import { useEffect, useRef } from 'react';
import { Loader2 } from 'lucide-react';
import { useAppStore } from '../../stores/appStore';
import { MessageBubble } from './MessageBubble';
import { ToolCallCard } from './ToolCallCard';
import { PlanView } from './PlanView';
import { PromptInput } from './PromptInput';
import { PermissionDialog } from './PermissionDialog';
import type { TimelineItem } from '../../types';

export function ChatPanel() {
  const { messages, toolCalls, plan, loading, getTimeline } = useAppStore();
  const scrollRef = useRef<HTMLDivElement>(null);

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
      <div ref={scrollRef} className="flex-1 overflow-y-auto py-4">
        {timeline.map((item, i) => (
          <TimelineItemRenderer key={itemKey(item, i)} item={item} />
        ))}

        {/* Plan */}
        {plan.length > 0 && <PlanView entries={plan} />}

        {/* Loading indicator */}
        {loading && (
          <div className="flex items-center gap-2 px-7 py-3">
            <div className="w-7 h-7 rounded-lg bg-[var(--bg-tertiary)] flex items-center justify-center">
              <Loader2 className="w-4 h-4 text-[var(--accent)] animate-spin" />
            </div>
            <span className="text-xs text-[var(--text-secondary)]">
              Agent is thinking...
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

function TimelineItemRenderer({ item }: { item: TimelineItem }) {
  if (item.type === 'message') {
    return <MessageBubble message={item.data} />;
  }
  return <ToolCallCard toolCall={item.data} />;
}

function itemKey(item: TimelineItem, index: number): string {
  if (item.type === 'message') {
    return `msg-${index}-${item.data.timestamp}`;
  }
  return `tc-${item.data.id}`;
}
