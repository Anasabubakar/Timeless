"use client";

import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { AnimatePresence, motion } from "motion/react";
import { RefreshCw, X } from "lucide-react";
import { useRealtimeStore } from "@/stores/realtime";

const AUTO_DISMISS_MS = 6000;

// Slides down from the top with a spring "bounce", shows who changed
// what, and gets out of the way on its own — a live-collaboration cue,
// not something that should block or interrupt whatever the user is
// doing. Refresh re-fetches rather than auto-applying the change, since
// silently swapping data out from under someone mid-edit is worse than
// asking.
export function RealtimeActivityBanner() {
  const latest = useRealtimeStore((s) => s.latest);
  const queryClient = useQueryClient();
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (!latest) return;
    setVisible(true);
    const timer = setTimeout(() => setVisible(false), AUTO_DISMISS_MS);
    return () => clearTimeout(timer);
  }, [latest]);

  const handleRefresh = () => {
    queryClient.invalidateQueries();
    setVisible(false);
  };

  return (
    <div className="pointer-events-none fixed inset-x-0 top-4 z-[60] flex justify-center px-4">
      <AnimatePresence>
        {latest && visible && (
          <motion.div
            key={latest.id}
            initial={{ y: -48, opacity: 0, scale: 0.95 }}
            animate={{ y: 0, opacity: 1, scale: 1 }}
            exit={{ y: -24, opacity: 0, scale: 0.97 }}
            transition={{ type: "spring", stiffness: 420, damping: 22, mass: 0.7 }}
            className="pointer-events-auto flex items-center gap-3 rounded-full border border-border bg-card/95 px-4 py-2 shadow-lg backdrop-blur"
          >
            <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-emerald-500" />
            <p className="text-xs font-medium text-foreground">
              <span className="text-muted-foreground">{latest.actorEmail}</span> {latest.subject}
            </p>
            <button
              onClick={handleRefresh}
              className="flex items-center gap-1 rounded-full bg-neutral-900 px-2.5 py-1 text-[11px] font-medium text-white hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-neutral-200"
            >
              <RefreshCw className="h-3 w-3" />
              Refresh
            </button>
            <button
              onClick={() => setVisible(false)}
              className="text-muted-foreground hover:text-foreground"
              aria-label="Dismiss"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
