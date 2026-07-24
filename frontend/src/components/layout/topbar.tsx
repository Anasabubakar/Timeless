"use client";

import { Search, Plus, Moon, Sun } from "lucide-react";
import { useTheme } from "next-themes";
import { useAuthStore } from "@/stores/auth";
import { getInitials } from "@/lib/utils";
import { NotificationBell } from "./notification-bell";

export function Topbar() {
  const { user } = useAuthStore();
  const { theme, setTheme } = useTheme();
  const initials = user ? getInitials(`${user.first_name} ${user.last_name}`) : "?";

  return (
    <header className="sticky top-0 z-30 flex h-14 items-center justify-between border-b border-border bg-card/80 px-6 backdrop-blur-sm">
      <div className="flex items-center gap-3">
        <button className="flex h-8 w-64 items-center gap-2 rounded-lg border border-input bg-muted/50 px-3 text-sm text-muted-foreground transition-colors hover:bg-muted">
          <Search className="h-3.5 w-3.5" />
          <span>Search...</span>
          <kbd className="ml-auto rounded border border-border bg-background px-1.5 py-0.5 text-[10px] font-medium">
            ⌘K
          </kbd>
        </button>
      </div>

      <div className="flex items-center gap-2">
        <button className="flex h-8 items-center gap-1.5 rounded-lg bg-primary px-3 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90">
          <Plus className="h-3.5 w-3.5" />
          New
        </button>

        <button
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
          className="flex h-8 w-8 items-center justify-center rounded-lg border border-input bg-muted/50 text-muted-foreground transition-colors hover:bg-muted"
        >
          {theme === "dark" ? <Sun className="h-3.5 w-3.5" /> : <Moon className="h-3.5 w-3.5" />}
        </button>

        <NotificationBell />

        <button className="flex h-8 w-8 items-center justify-center rounded-full bg-primary text-xs font-medium text-primary-foreground">
          {initials}
        </button>
      </div>
    </header>
  );
}
