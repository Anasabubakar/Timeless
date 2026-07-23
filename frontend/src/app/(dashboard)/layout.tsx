import { Sidebar } from "@/components/layout/sidebar";
import { Topbar } from "@/components/layout/topbar";
import { CommandPalette } from "@/components/command-palette";
import { AIAssistant } from "@/components/ai-assistant";
import { AuthGuard } from "@/components/auth-guard";
import { RealtimeProvider } from "@/components/realtime-provider";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AuthGuard>
      <RealtimeProvider>
        <div className="flex min-h-screen">
          <Sidebar />
          <div className="flex flex-1 flex-col pl-[240px]">
            <Topbar />
            <main className="flex-1 p-6">{children}</main>
          </div>
          <CommandPalette />
          <AIAssistant />
        </div>
      </RealtimeProvider>
    </AuthGuard>
  );
}
