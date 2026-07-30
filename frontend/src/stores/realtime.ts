import { create } from "zustand";

export interface ActivityEvent {
  id: string;
  actorEmail: string;
  action: string;
  entityType: string;
  subject: string;
  receivedAt: number;
}

interface RealtimeState {
  latest: ActivityEvent | null;
  push: (event: Omit<ActivityEvent, "id" | "receivedAt">) => void;
  clear: () => void;
}

// A single-slot store (not a growing list) is enough for a banner that
// only ever shows one notification at a time — see RealtimeActivityBanner.
// Deduping identical back-to-back events (same actor+subject within a
// few seconds) happens here rather than in the banner component so any
// other future subscriber gets the same de-duped stream.
let lastKey = "";
let lastAt = 0;
const DEDUP_WINDOW_MS = 4000;

export const useRealtimeStore = create<RealtimeState>()((set) => ({
  latest: null,
  push: (event) => {
    const key = `${event.actorEmail}:${event.subject}`;
    const now = Date.now();
    if (key === lastKey && now - lastAt < DEDUP_WINDOW_MS) {
      return;
    }
    lastKey = key;
    lastAt = now;
    set({
      latest: {
        ...event,
        id: `${now}-${Math.random().toString(36).slice(2, 8)}`,
        receivedAt: now,
      },
    });
  },
  clear: () => set({ latest: null }),
}));
