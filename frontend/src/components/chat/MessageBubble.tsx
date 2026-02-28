import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { clsx } from 'clsx';
import { Bot, User } from 'lucide-react';
import type { MessageInfo } from '../../types';

interface MessageBubbleProps {
  message: MessageInfo;
}

export function MessageBubble({ message }: MessageBubbleProps) {
  const isUser = message.role === 'user';
  const isSystem = message.role === 'system';

  // System messages — minimal centered pill
  if (isSystem) {
    return (
      <div className="flex justify-center my-2 px-4">
        <div className="text-[10px] text-[var(--text-muted)] bg-[var(--bg-tertiary)] px-3 py-1 rounded-full border border-[var(--border-subtle)]">
          {message.content}
        </div>
      </div>
    );
  }

  // User messages — right-aligned, accent tinted
  if (isUser) {
    return (
      <div className="flex gap-2.5 px-5 py-2 flex-row-reverse animate-fade-in">
        {/* Avatar */}
        <div className="w-6 h-6 rounded-md bg-[var(--accent-muted)] flex items-center justify-center shrink-0 mt-0.5">
          <User className="w-3.5 h-3.5 text-[var(--accent)]" />
        </div>

        {/* Content */}
        <div className="max-w-[70%] rounded-lg rounded-tr-sm px-3.5 py-2 text-sm leading-relaxed bg-[var(--accent)] text-white">
          <p className="whitespace-pre-wrap">{message.content}</p>
          <div className="text-[9px] mt-1 opacity-40 text-right font-mono">
            {formatTime(message.timestamp)}
          </div>
        </div>
      </div>
    );
  }

  // Agent messages — left-aligned, Zed-style (no heavy bubble, clean text)
  return (
    <div className="flex gap-2.5 px-5 py-2 animate-fade-in">
      {/* Avatar */}
      <div className="w-6 h-6 rounded-md bg-[var(--bg-tertiary)] border border-[var(--border-subtle)] flex items-center justify-center shrink-0 mt-0.5">
        <Bot className="w-3.5 h-3.5 text-[var(--accent)]" />
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0 max-w-[80%]">
        <div className="markdown-content text-sm leading-relaxed text-[var(--text-primary)]">
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            components={{
              code({ className, children, ...props }) {
                const match = /language-(\w+)/.exec(className || '');
                const codeString = String(children).replace(/\n$/, '');

                if (match) {
                  return (
                    <SyntaxHighlighter
                      style={oneDark}
                      language={match[1]}
                      PreTag="div"
                      customStyle={{
                        margin: '0.5rem 0',
                        borderRadius: '0.375rem',
                        fontSize: '0.8rem',
                        border: '1px solid var(--border-subtle)',
                        background: 'var(--bg-primary)',
                      }}
                    >
                      {codeString}
                    </SyntaxHighlighter>
                  );
                }

                return (
                  <code
                    className="bg-[var(--bg-primary)] text-[var(--accent-hover)] px-1 py-0.5 rounded text-xs font-mono border border-[var(--border-subtle)]"
                    {...props}
                  >
                    {children}
                  </code>
                );
              },
              a({ href, children }) {
                return (
                  <a
                    href={href}
                    className="text-[var(--accent-hover)] underline underline-offset-2 hover:text-[var(--accent)]"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {children}
                  </a>
                );
              },
            }}
          >
            {message.content}
          </ReactMarkdown>
        </div>
        <div className="text-[9px] mt-1 text-[var(--text-muted)] opacity-40 font-mono">
          {formatTime(message.timestamp)}
        </div>
      </div>
    </div>
  );
}

function formatTime(ts: string): string {
  try {
    const d = new Date(ts);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  } catch {
    return '';
  }
}
