"use client";

import { useWebSocket } from "@/hooks/use-websocket";
import { RealtimeActivityBanner } from "@/components/realtime-activity-banner";

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  useWebSocket();
  return (
    <>
      <RealtimeActivityBanner />
      {children}
    </>
  );
}
