import { useEffect } from 'react';
import { Hammer, Bot, Zap, Flame } from 'lucide-react';
import { MainLayout } from './components/layout/MainLayout';
import { ChatPanel } from './components/chat/ChatPanel';
import { useWailsEvents } from './hooks/useWailsEvents';
import { useAppStore } from './stores/appStore';

function WelcomeScreen() {
  return (
    <div className="flex-1 flex items-center justify-center">
      <div className="text-center max-w-md px-6 animate-fade-in">
        {/* Forge icon with ember glow */}
        <div className="relative w-20 h-20 mx-auto mb-8">
          <div className="absolute inset-0 rounded-2xl bg-[var(--accent-muted)] animate-glow-pulse" />
          <div className="relative w-full h-full rounded-2xl bg-[var(--bg-tertiary)] border border-[var(--border)] flex items-center justify-center">
            <Hammer className="w-10 h-10 text-[var(--accent)]" />
          </div>
        </div>

        <h1 className="text-2xl font-bold mb-1 text-[var(--text-primary)]">
          ByteSmith
        </h1>
        <p className="text-xs text-[var(--text-muted)] font-mono tracking-widest uppercase mb-6">
          forge your code
        </p>
        <p className="text-sm text-[var(--text-secondary)] mb-10 leading-relaxed max-w-sm mx-auto">
          A desktop client for AI coding agents. Connect to Codex, OpenCode,
          and more through the Agent Client Protocol.
        </p>

        <div className="space-y-2 text-left">
          <StepCard
            step={1}
            icon={<Bot className="w-4 h-4" />}
            title="Select an Agent"
            description="Choose an installed AI agent from the sidebar"
          />
          <StepCard
            step={2}
            icon={<Zap className="w-4 h-4" />}
            title="Pick a Directory"
            description="Set the working directory for your project"
          />
          <StepCard
            step={3}
            icon={<Flame className="w-4 h-4" />}
            title="Start Forging"
            description="Send prompts and let the agent help you build"
          />
        </div>
      </div>
    </div>
  );
}

function StepCard({
  step,
  icon,
  title,
  description,
}: {
  step: number;
  icon: React.ReactNode;
  title: string;
  description: string;
}) {
  return (
    <div
      className="flex items-center gap-3 p-3 rounded-lg bg-[var(--bg-secondary)] border border-[var(--border-subtle)] hover:border-[var(--border)] transition-all duration-200 group"
      style={{ animationDelay: `${step * 100}ms` }}
    >
      <div className="w-8 h-8 rounded-md bg-[var(--bg-tertiary)] flex items-center justify-center shrink-0 text-[var(--accent)] group-hover:bg-[var(--accent-muted)] transition-colors">
        {icon}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-xs font-medium text-[var(--text-primary)]">{title}</p>
        <p className="text-[11px] text-[var(--text-muted)]">{description}</p>
      </div>
      <span className="text-[10px] font-mono text-[var(--text-muted)] opacity-40">{step}</span>
    </div>
  );
}

function App() {
  const { activeSession, error, setError } = useAppStore();

  // Subscribe to Wails backend events
  useWailsEvents();

  // Auto-dismiss errors after 5 seconds
  useEffect(() => {
    if (error) {
      const timer = setTimeout(() => setError(null), 5000);
      return () => clearTimeout(timer);
    }
  }, [error, setError]);

  return (
    <MainLayout>
      {/* Error toast */}
      {error && (
        <div className="absolute top-3 right-3 z-50 max-w-sm bg-[var(--error-muted)] border border-[var(--error)] text-[var(--error)] px-4 py-2.5 rounded-lg text-xs shadow-elevated animate-slide-up backdrop-blur-sm">
          {error}
        </div>
      )}

      {activeSession ? <ChatPanel /> : <WelcomeScreen />}
    </MainLayout>
  );
}

export default App;
