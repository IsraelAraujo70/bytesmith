import { Sidebar } from './Sidebar';
import { StatusBar } from './StatusBar';

interface MainLayoutProps {
  children: React.ReactNode;
}

export function MainLayout({ children }: MainLayoutProps) {
  return (
    <div className="flex flex-col h-screen w-screen overflow-hidden">
      <div className="flex flex-1 min-h-0">
        <Sidebar />
        <main className="flex-1 flex flex-col min-w-0">{children}</main>
      </div>
      <StatusBar />
    </div>
  );
}
