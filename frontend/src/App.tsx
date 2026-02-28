import { useEffect } from 'react';
import { Terminal, Bot, Zap } from 'lucide-react';
import { MainLayout } from './components/layout/MainLayout';
import { ChatPanel } from './components/chat/ChatPanel';
import { useWailsEvents } from './hooks/useWailsEvents';
import { useAppStore } from './stores/appStore';

function WelcomeScreen() {
  return (
    <div className="flex-1 flex items-center justify-center">
      <div className="text-center max-w-md px-6">
        <div className="w-16 h-16 rounded-2xl bg-[var(--accent)] bg-opacity-20 flex items-center justify-center mx-auto mb-6">
          <Terminal className="w-8 h-8 text-[var(--accent)]" />
        </div>
        <h1 className="text-2xl font-bold mb-2">Welcome to ByteSmith</h1>
        <p className="text-sm text-[var(--text-secondary)] mb-8 leading-relaxed">
          A desktop client for AI coding agents. Connect to Claude Code, Codex,
          Gemini CLI, and more through the Agent Client Protocol.
        </p>

        <div className="space-y-3 text-left">
          <StepCard
            icon={<Bot className="w-4 h-4" />}
            title="Select an Agent"
            description="Choose an installed AI agent from the sidebar"
          />
          <StepCard
            icon={<Zap className="w-4 h-4" />}
            title="Pick a Directory"
            description="Set the working directory for your project"
          />
          <StepCard
            icon={<Terminal className="w-4 h-4" />}
            title="Start Coding"
            description="Send prompts and let the agent help you build"
          />
        </div>
      </div>
    </div>
  );
}

function StepCard({
  icon,
  title,
  description,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
}) {
  return (
    <div className="flex items-start gap-3 p-3 rounded-xl bg-[var(--bg-secondary)] border border-[var(--border)]">
      <div className="w-8 h-8 rounded-lg bg-[var(--bg-tertiary)] flex items-center justify-center shrink-0 text-[var(--accent)]">
        {icon}
      </div>
      <div>
        <p className="text-sm font-medium">{title}</p>
        <p className="text-xs text-[var(--text-secondary)]">{description}</p>
      </div>
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
        <div className="absolute top-4 right-4 z-50 max-w-sm bg-[var(--error)] bg-opacity-20 border border-[var(--error)] border-opacity-30 text-[var(--error)] px-4 py-3 rounded-xl text-sm shadow-xl">
          {error}
        </div>
      )}

      {activeSession ? <ChatPanel /> : <WelcomeScreen />}
    </MainLayout>
  );
}

export default App;
