import { useEffect, useMemo, useState, type FormEvent } from 'react';
import { clsx } from 'clsx';
import { Check, MessageSquarePlus } from 'lucide-react';
import { useAppStore } from '../../stores/appStore';
import { rejectQuestion, respondQuestion } from '../../lib/api';
import type { QuestionItem } from '../../types';

const USER_NOTE_PREFIX = 'user_note: ';
const OTHER_OPTION_LABEL = 'None of the above';
const SECRET_MASK = '********';

function questionLabel(question: QuestionItem, index: number): string {
  const header = question.header.trim();
  if (header) {
    return header;
  }
  return `Question ${index + 1}`;
}

function renderAnswerValue(values: string[]): string {
  const readable = values
    .map((value) =>
      value.startsWith(USER_NOTE_PREFIX)
        ? value.slice(USER_NOTE_PREFIX.length)
        : value
    )
    .filter((value) => value.trim().length > 0);
  return readable.join(', ');
}

function renderReviewValue(question: QuestionItem, values: string[]): string {
  const resolved = renderAnswerValue(values);
  if (!resolved) {
    return '';
  }
  if (question.isSecret) {
    return SECRET_MASK;
  }
  return resolved;
}

export function QuestionDialog() {
  const {
    questionRequests,
    removeQuestionRequest,
    activeSession,
  } = useAppStore();

  const request = useMemo(
    () =>
      questionRequests.find(
        (r) =>
          activeSession &&
          r.connectionId === activeSession.connectionID &&
          r.sessionId === activeSession.sessionID
      ) ?? questionRequests[0],
    [activeSession, questionRequests]
  );

  const [tab, setTab] = useState(0);
  const [answers, setAnswers] = useState<string[][]>([]);
  const [custom, setCustom] = useState<string[]>([]);
  const [editing, setEditing] = useState(false);
  const [sending, setSending] = useState(false);

  useEffect(() => {
    setTab(0);
    setAnswers([]);
    setCustom([]);
    setEditing(false);
    setSending(false);
  }, [request?.requestId]);

  if (!request) {
    return null;
  }

  const questions = request.questions;
  const single =
    questions.length === 1 &&
    questions[0]?.multiple !== true;
  const confirm = !single && tab === questions.length;
  const question = questions[tab];
  const options = question?.options ?? [];
  const canTypeCustom = Boolean(question?.isOther);
  const multi = question?.multiple === true;

  const answerForQuestion = (index: number): string[] => {
    const explicit = answers[index];
    if (explicit && explicit.length > 0) {
      return explicit;
    }
    return [];
  };

  const setAnswerAt = (index: number, next: string[]) => {
    setAnswers((current) => {
      const updated = [...current];
      updated[index] = next;
      return updated;
    });
  };

  const setCustomAt = (index: number, value: string) => {
    setCustom((current) => {
      const updated = [...current];
      updated[index] = value;
      return updated;
    });
  };

  const buildPayload = (): Record<string, string[]> => {
    const payload: Record<string, string[]> = {};
    for (let i = 0; i < questions.length; i += 1) {
      const q = questions[i];
      if (!q || !q.id) {
        continue;
      }
      payload[q.id] = answerForQuestion(i);
    }
    return payload;
  };

  const submitPayload = async (payload: Record<string, string[]>) => {
    if (sending) {
      return;
    }

    setSending(true);
    try {
      await respondQuestion(request.requestId, payload);
      removeQuestionRequest(request.requestId);
    } catch {
      return;
    } finally {
      setSending(false);
    }
  };

  const dismiss = async () => {
    if (sending) {
      return;
    }

    setSending(true);
    try {
      await rejectQuestion(request.requestId);
      removeQuestionRequest(request.requestId);
    } catch {
      return;
    } finally {
      setSending(false);
    }
  };

  const submitAll = () => {
    void submitPayload(buildPayload());
  };

  const pickOption = (answer: string) => {
    if (!question) {
      return;
    }

    if (single) {
      void submitPayload({ [question.id]: [answer] });
      return;
    }

    setAnswerAt(tab, [answer]);
    setTab((current) => current + 1);
  };

  const toggleOption = (answer: string) => {
    setAnswers((current) => {
      const updated = [...current];
      const existing = (updated[tab] ?? []).filter(
        (item) => item !== OTHER_OPTION_LABEL && !item.startsWith(USER_NOTE_PREFIX)
      );

      if (existing.includes(answer)) {
        updated[tab] = existing.filter((item) => item !== answer);
      } else {
        updated[tab] = [...existing, answer];
      }

      return updated;
    });
  };

  const submitFreeform = (event: FormEvent) => {
    event.preventDefault();
    if (sending || !question) {
      return;
    }

    const value = (custom[tab] ?? '').trim();
    if (!value) {
      return;
    }

    const answerList = [`${USER_NOTE_PREFIX}${value}`];
    if (single) {
      void submitPayload({ [question.id]: answerList });
      return;
    }

    setAnswerAt(tab, answerList);
    setTab((current) => current + 1);
  };

  const submitCustom = (event: FormEvent) => {
    event.preventDefault();
    if (sending || !question) {
      return;
    }

    const value = (custom[tab] ?? '').trim();
    if (!value) {
      setEditing(false);
      return;
    }

    if (multi) {
      setAnswers((current) => {
        const updated = [...current];
        const existing = (updated[tab] ?? []).filter(
          (item) => item !== OTHER_OPTION_LABEL && !item.startsWith(USER_NOTE_PREFIX)
        );
        updated[tab] = [
          ...existing,
          OTHER_OPTION_LABEL,
          `${USER_NOTE_PREFIX}${value}`,
        ];
        return updated;
      });
      setEditing(false);
      return;
    }

    const answerList = [OTHER_OPTION_LABEL, `${USER_NOTE_PREFIX}${value}`];
    if (single) {
      void submitPayload({ [question.id]: answerList });
      return;
    }

    setAnswerAt(tab, answerList);
    setEditing(false);
    setTab((current) => current + 1);
  };

  const selectOption = (optIndex: number) => {
    if (sending || !question) {
      return;
    }

    if (canTypeCustom && optIndex === options.length) {
      setEditing(true);
      return;
    }

    const option = options[optIndex];
    if (!option) {
      return;
    }

    if (multi) {
      toggleOption(option.label);
      return;
    }

    pickOption(option.label);
  };

  const currentAnswers = question ? answerForQuestion(tab) : [];
  const customPicked = currentAnswers.includes(OTHER_OPTION_LABEL);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
      <div className="bg-[var(--bg-elevated)] border border-[var(--border)] rounded-xl shadow-elevated w-full max-w-2xl mx-4 overflow-hidden animate-slide-up max-h-[85vh] flex flex-col">
        <div className="flex items-center gap-3 px-4 py-3 border-b border-[var(--border-subtle)]">
          <div className="w-8 h-8 rounded-lg bg-[var(--accent-muted)] flex items-center justify-center">
            <MessageSquarePlus className="w-4 h-4 text-[var(--accent)]" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-[var(--text-primary)]">Question</h3>
            <p className="text-[10px] text-[var(--text-muted)]">
              The agent needs extra input to continue
            </p>
          </div>
        </div>

        {!single && (
          <div className="px-3 py-2 border-b border-[var(--border-subtle)] bg-[var(--bg-secondary)] flex flex-wrap gap-1.5">
            {questions.map((q, index) => {
              const answered = answerForQuestion(index).length > 0;
              const active = index === tab;
              return (
                <button
                  key={`${q.id || 'q'}-${index}`}
                  onClick={() => {
                    setTab(index);
                    setEditing(false);
                  }}
                  disabled={sending}
                  className={clsx(
                    'px-2 py-1 rounded-md border text-[10px] transition-colors',
                    active
                      ? 'border-[var(--accent)] text-[var(--accent)] bg-[var(--accent-muted)]'
                      : answered
                      ? 'border-[var(--success)] text-[var(--success)] bg-[var(--success-muted)]'
                      : 'border-[var(--border-subtle)] text-[var(--text-muted)] hover:border-[var(--border)] hover:text-[var(--text-secondary)]'
                  )}
                >
                  {questionLabel(q, index)}
                </button>
              );
            })}
            <button
              onClick={() => {
                setTab(questions.length);
                setEditing(false);
              }}
              disabled={sending}
              className={clsx(
                'px-2 py-1 rounded-md border text-[10px] transition-colors',
                confirm
                  ? 'border-[var(--accent)] text-[var(--accent)] bg-[var(--accent-muted)]'
                  : 'border-[var(--border-subtle)] text-[var(--text-muted)] hover:border-[var(--border)] hover:text-[var(--text-secondary)]'
              )}
            >
              Review
            </button>
          </div>
        )}

        <div className="px-4 py-3 overflow-y-auto">
          {!confirm && question && (
            <>
              <div className="text-xs text-[var(--text-primary)] mb-2.5">
                {question.question}
              </div>

              {options.length > 0 ? (
                <div className="space-y-2">
                  {options.map((opt, index) => {
                    const picked = currentAnswers.includes(opt.label);
                    return (
                      <button
                        key={`${opt.label}-${index}`}
                        onClick={() => selectOption(index)}
                        disabled={sending}
                        className={clsx(
                          'w-full text-left px-3 py-2 rounded-md border transition-colors',
                          picked
                            ? 'border-[var(--accent)] bg-[var(--accent-muted)] text-[var(--text-primary)]'
                            : 'border-[var(--border-subtle)] bg-[var(--bg-secondary)] hover:border-[var(--border)] text-[var(--text-secondary)]'
                        )}
                      >
                        <div className="flex items-start gap-2">
                          <div className="flex-1">
                            <div className="text-xs font-medium">{opt.label}</div>
                            {opt.description && (
                              <div className="text-[10px] text-[var(--text-muted)] mt-0.5">
                                {opt.description}
                              </div>
                            )}
                          </div>
                          {picked && <Check className="w-3.5 h-3.5 mt-0.5 text-[var(--accent)]" />}
                        </div>
                      </button>
                    );
                  })}

                  {canTypeCustom && (
                    <button
                      onClick={() => selectOption(options.length)}
                      disabled={sending}
                      className={clsx(
                        'w-full text-left px-3 py-2 rounded-md border transition-colors',
                        customPicked
                          ? 'border-[var(--accent)] bg-[var(--accent-muted)] text-[var(--text-primary)]'
                          : 'border-[var(--border-subtle)] bg-[var(--bg-secondary)] hover:border-[var(--border)] text-[var(--text-secondary)]'
                      )}
                    >
                      <div className="flex items-start gap-2">
                        <div className="flex-1">
                          <div className="text-xs font-medium">Type your own answer</div>
                          {!editing && (custom[tab] ?? '').trim() && (
                            <div className="text-[10px] text-[var(--text-muted)] mt-0.5 break-all">
                              {question?.isSecret ? SECRET_MASK : (custom[tab] ?? '').trim()}
                            </div>
                          )}
                        </div>
                        {customPicked && (
                          <Check className="w-3.5 h-3.5 mt-0.5 text-[var(--accent)]" />
                        )}
                      </div>
                    </button>
                  )}

                  {editing && canTypeCustom && (
                    <form onSubmit={submitCustom} className="flex items-center gap-2 pt-1">
                      <input
                        autoFocus
                        type={question.isSecret ? 'password' : 'text'}
                        value={custom[tab] ?? ''}
                        onChange={(event) => setCustomAt(tab, event.currentTarget.value)}
                        disabled={sending}
                        placeholder={
                          question.isSecret ? 'Type a secret value' : 'Type your answer'
                        }
                        className="flex-1 h-8 px-2.5 text-xs rounded-md border border-[var(--border-subtle)] bg-[var(--bg-secondary)] text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:outline-none focus:border-[var(--accent)]"
                      />
                      <button
                        type="submit"
                        disabled={sending}
                        className="h-8 px-2.5 rounded-md bg-[var(--accent)] text-white text-[11px] font-medium hover:bg-[var(--accent-hover)] transition-colors disabled:opacity-60"
                      >
                        {multi ? 'Add' : 'Submit'}
                      </button>
                      <button
                        type="button"
                        disabled={sending}
                        onClick={() => setEditing(false)}
                        className="h-8 px-2.5 rounded-md border border-[var(--border-subtle)] text-[11px] text-[var(--text-secondary)] hover:border-[var(--border)]"
                      >
                        Cancel
                      </button>
                    </form>
                  )}
                </div>
              ) : (
                <form onSubmit={submitFreeform} className="space-y-2">
                  <input
                    autoFocus
                    type={question.isSecret ? 'password' : 'text'}
                    value={custom[tab] ?? ''}
                    onChange={(event) => setCustomAt(tab, event.currentTarget.value)}
                    disabled={sending}
                    placeholder={
                      question.isSecret ? 'Type a secret value' : 'Type your answer'
                    }
                    className="w-full h-9 px-3 text-xs rounded-md border border-[var(--border-subtle)] bg-[var(--bg-secondary)] text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:outline-none focus:border-[var(--accent)]"
                  />
                  <button
                    type="submit"
                    disabled={sending || !(custom[tab] ?? '').trim()}
                    className="h-8 px-3 rounded-md bg-[var(--accent)] text-white text-[11px] font-medium hover:bg-[var(--accent-hover)] transition-colors disabled:opacity-60"
                  >
                    {single ? 'Submit' : 'Save and continue'}
                  </button>
                </form>
              )}
            </>
          )}

          {confirm && (
            <div className="space-y-2.5">
              <div className="text-xs font-medium text-[var(--text-primary)]">Review answers</div>
              {questions.map((q, index) => {
                const values = answerForQuestion(index);
                const resolved = renderReviewValue(q, values);
                const answered = resolved.length > 0;
                return (
                  <div
                    key={`${q.id || 'review'}-${index}`}
                    className="rounded-md border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 py-2"
                  >
                    <div className="text-[11px] text-[var(--text-secondary)]">{q.question}</div>
                    <div
                      className={clsx(
                        'text-xs mt-1 break-all',
                        answered ? 'text-[var(--text-primary)]' : 'text-[var(--text-muted)]'
                      )}
                    >
                      {answered ? resolved : 'Not answered'}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        <div className="px-4 py-3 border-t border-[var(--border-subtle)] flex items-center justify-end gap-2">
          <button
            onClick={() => {
              void dismiss();
            }}
            disabled={sending}
            className="h-8 px-3 rounded-md border border-[var(--border-subtle)] text-[11px] text-[var(--text-secondary)] hover:border-[var(--border)]"
          >
            Dismiss
          </button>

          {!single && confirm && (
            <button
              onClick={submitAll}
              disabled={sending}
              className="h-8 px-3 rounded-md bg-[var(--accent)] text-white text-[11px] font-medium hover:bg-[var(--accent-hover)] transition-colors disabled:opacity-60"
            >
              Submit
            </button>
          )}

          {!single && !confirm && multi && (
            <button
              onClick={() => {
                setTab((current) => current + 1);
                setEditing(false);
              }}
              disabled={sending || currentAnswers.length === 0}
              className="h-8 px-3 rounded-md bg-[var(--bg-tertiary)] text-[var(--text-primary)] text-[11px] font-medium hover:bg-[var(--border)] transition-colors disabled:opacity-60"
            >
              Next
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
