import { Sidebar } from "@/components/layout/sidebar";
import { Topbar } from "@/components/layout/topbar";
import { CommandPalette } from "@/components/command-palette";
import { AIAssistant } from "@/components/ai-assistant";
import { AuthGuard } from "@/components/auth-guard";
import { RealtimeProvider } from "@/components/realtime-provider";
import { MobileDock } from "@/components/layout/mobile-dock";

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
          <div className="flex flex-1 flex-col md:pl-16 lg:pl-[240px]">
            <Topbar />
            <main className="flex-1 p-4 pb-28 sm:p-6 md:pb-6">{children}</main>
          </div>
          <CommandPalette />
          <AIAssistant />
          <MobileDock />
        </div>
      </RealtimeProvider>
    </AuthGuard>
  );
}
