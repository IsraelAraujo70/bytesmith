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

  if (isSystem) {
    return (
      <div className="flex justify-center my-2 px-4">
        <div className="text-xs text-[var(--text-secondary)] bg-[var(--bg-tertiary)] px-3 py-1.5 rounded-full">
          {message.content}
        </div>
      </div>
    );
  }

  return (
    <div
      className={clsx(
        'flex gap-3 px-4 py-3',
        isUser ? 'flex-row-reverse' : 'flex-row'
      )}
    >
      {/* Avatar */}
      <div
        className={clsx(
          'w-7 h-7 rounded-lg flex items-center justify-center shrink-0 mt-0.5',
          isUser
            ? 'bg-[var(--accent)] bg-opacity-20'
            : 'bg-[var(--bg-tertiary)]'
        )}
      >
        {isUser ? (
          <User className="w-4 h-4 text-[var(--accent)]" />
        ) : (
          <Bot className="w-4 h-4 text-[var(--accent)]" />
        )}
      </div>

      {/* Bubble */}
      <div
        className={clsx(
          'max-w-[75%] rounded-xl px-4 py-2.5 text-sm leading-relaxed',
          isUser
            ? 'bg-[var(--accent)] text-white rounded-tr-sm'
            : 'bg-[var(--bg-tertiary)] text-[var(--text-primary)] rounded-tl-sm'
        )}
      >
        {isUser ? (
          <p className="whitespace-pre-wrap">{message.content}</p>
        ) : (
          <div className="markdown-content">
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
                          borderRadius: '0.5rem',
                          fontSize: '0.8rem',
                        }}
                      >
                        {codeString}
                      </SyntaxHighlighter>
                    );
                  }

                  return (
                    <code
                      className="bg-[var(--bg-primary)] text-[var(--accent-hover)] px-1.5 py-0.5 rounded text-xs"
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
                      className="text-[var(--accent-hover)] underline underline-offset-2"
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
        )}

        {/* Timestamp */}
        <div
          className={clsx(
            'text-[10px] mt-1.5 opacity-50',
            isUser ? 'text-right' : 'text-left'
          )}
        >
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
