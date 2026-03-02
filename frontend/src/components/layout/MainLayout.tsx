import { Sidebar } from './Sidebar';
import { StatusBar } from './StatusBar';
import { ModelPickerModal } from '../common/ModelPickerModal';
import { TerminalPanel } from '../terminal/TerminalPanel';
import { useAppStore } from '../../stores/appStore';

interface MainLayoutProps {
  children: React.ReactNode;
}

export function MainLayout({ children }: MainLayoutProps) {
  const { sidebarCollapsed } = useAppStore();

  return (
    <div className="flex flex-col h-full w-full overflow-hidden bg-[var(--bg-primary)]">
      {/* Main content area */}
      <div className="flex flex-1 min-h-0">
        <Sidebar />
        <main
          className="flex-1 flex flex-col min-w-0 transition-all duration-200"
          style={{ marginLeft: sidebarCollapsed ? 0 : undefined }}
        >
          <div className="min-h-0 flex-1">
            {children}
          </div>
          <TerminalPanel />
        </main>
      </div>
      <StatusBar />

      {/* Global modals */}
      <ModelPickerModal />
    </div>
  );
}
