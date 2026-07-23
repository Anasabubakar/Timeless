"use client";

import { useWebSocket } from "@/hooks/use-websocket";

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  useWebSocket();
  return <>{children}</>;
}
