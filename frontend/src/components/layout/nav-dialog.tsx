"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { AnimatePresence, motion } from "motion/react";
import { Search, X } from "lucide-react";
import { groupedNavigation } from "@/lib/navigation";
import { useReducedMotion } from "@/hooks/use-media-query";

interface NavDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

// The full application menu, opened from the center dock button. A
// lightweight, mobile-optimized command palette: search up top, every
// destination grouped underneath, large touch targets throughout.
export function NavDialog({ open, onOpenChange }: NavDialogProps) {
  const [query, setQuery] = useState("");
  const prefersReducedMotion = useReducedMotion();

  const groups = useMemo(() => {
    const all = groupedNavigation();
    const q = query.trim().toLowerCase();
    if (!q) return all;
    return all
      .map((g) => ({ ...g, items: g.items.filter((item) => item.name.toLowerCase().includes(q)) }))
      .filter((g) => g.items.length > 0);
  }, [query]);

  function handleNavigate() {
    setQuery("");
    onOpenChange(false);
  }

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          className="fixed inset-0 z-[60] flex flex-col bg-background md:hidden"
          initial={{ opacity: 0, y: 24 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 24 }}
          transition={prefersReducedMotion ? { duration: 0 } : { type: "spring", damping: 30, stiffness: 300 }}
          drag="y"
          dragConstraints={{ top: 0, bottom: 0 }}
          dragElastic={{ top: 0, bottom: 0.6 }}
          onDragEnd={(_, info) => {
            if (info.offset.y > 140 || info.velocity.y > 600) onOpenChange(false);
          }}
        >
          <div
            className="flex items-center gap-2 border-b border-border px-4 pb-3"
            style={{ paddingTop: "max(1rem, env(safe-area-inset-top, 0px))" }}
          >
            <div className="flex h-11 flex-1 items-center gap-2 rounded-xl border border-input bg-muted/50 px-3">
              <Search className="h-4 w-4 shrink-0 text-muted-foreground" />
              <input
                autoFocus
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Search anywhere..."
                className="h-full flex-1 bg-transparent text-base outline-none placeholder:text-muted-foreground"
              />
            </div>
            <button
              onClick={() => onOpenChange(false)}
              aria-label="Close menu"
              className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl text-muted-foreground transition-colors hover:bg-accent"
            >
              <X className="h-5 w-5" />
            </button>
          </div>

          <div
            className="flex-1 overflow-y-auto overscroll-contain px-4 pt-2"
            style={{ paddingBottom: "max(1.5rem, env(safe-area-inset-bottom, 0px))" }}
          >
            {groups.length === 0 && (
              <p className="py-12 text-center text-sm text-muted-foreground">
                No matches for &ldquo;{query}&rdquo;
              </p>
            )}
            {groups.map((group) => (
              <div key={group.group} className="mb-6">
                <h3 className="mb-2 px-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                  {group.group}
                </h3>
                <div className="flex flex-col gap-1">
                  {group.items.map((item) => (
                    <Link
                      key={item.href}
                      href={item.href}
                      onClick={handleNavigate}
                      className="flex min-h-[52px] items-center gap-3 rounded-xl px-3 text-base font-medium text-foreground transition-colors active:bg-accent"
                    >
                      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-muted">
                        <item.icon className="h-4 w-4" />
                      </span>
                      {item.name}
                    </Link>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
