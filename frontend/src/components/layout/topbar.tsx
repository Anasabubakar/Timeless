"use client";

import { useState } from "react";
import { Search, Plus, Moon, Sun, Megaphone, Users, Link2, Mail, Zap, Building2 } from "lucide-react";
import { useTheme } from "next-themes";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { getInitials } from "@/lib/utils";
import { useMobileNavStore } from "@/stores/mobile-nav";
import { NotificationBell } from "./notification-bell";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

const NEW_ITEMS = [
  { label: "New Campaign", icon: Megaphone, href: "/campaigns", key: "campaign" },
  { label: "New Sponsor", icon: Users, href: "/sponsors", key: "sponsor" },
  { label: "New Company", icon: Building2, href: "/companies", key: "company" },
  { label: "New Contact", icon: Users, href: "/contacts", key: "contact" },
  { label: "New Sequence", icon: Mail, href: "/outreach", key: "sequence" },
  { label: "New Automation", icon: Zap, href: "/automations", key: "automation" },
  { label: "New Integration", icon: Link2, href: "/integrations", key: "integration" },
];

export function Topbar() {
  const { user } = useAuthStore();
  const { theme, setTheme } = useTheme();
  const setMobileNavOpen = useMobileNavStore((state) => state.setOpen);
  const initials = user ? getInitials(`${user.first_name} ${user.last_name}`) : "?";
  const [showNew, setShowNew] = useState(false);
  const router = useRouter();

  return (
    <>
      <header
        className="sticky top-0 z-30 flex h-14 items-center justify-between gap-2 border-b border-border bg-card/80 px-4 backdrop-blur-sm sm:px-6"
        style={{ paddingTop: "env(safe-area-inset-top, 0px)" }}
      >
        <div className="flex min-w-0 flex-1 items-center gap-3">
          <button className="hidden h-8 w-64 items-center gap-2 rounded-lg border border-input bg-muted/50 px-3 text-sm text-muted-foreground transition-colors hover:bg-muted sm:flex">
            <Search className="h-3.5 w-3.5" />
            <span>Search...</span>
            <kbd className="ml-auto rounded border border-border bg-background px-1.5 py-0.5 text-[10px] font-medium">
              ⌘K
            </kbd>
          </button>

          <button
            onClick={() => setMobileNavOpen(true)}
            aria-label="Search"
            className="flex h-11 w-11 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted sm:hidden"
          >
            <Search className="h-5 w-5" />
          </button>
        </div>

        <div className="flex shrink-0 items-center gap-1.5 sm:gap-2">
          <button
            onClick={() => setShowNew(true)}
            className="flex h-9 items-center gap-1.5 rounded-lg bg-primary px-2.5 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90 sm:h-8 sm:px-3"
          >
            <Plus className="h-3.5 w-3.5" />
            <span className="hidden sm:inline">New</span>
          </button>

          <button
            onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
            className="flex h-9 w-9 items-center justify-center rounded-lg border border-input bg-muted/50 text-muted-foreground transition-colors hover:bg-muted sm:h-8 sm:w-8"
          >
            {theme === "dark" ? <Sun className="h-3.5 w-3.5" /> : <Moon className="h-3.5 w-3.5" />}
          </button>

          <NotificationBell />

          <button className="flex h-9 w-9 items-center justify-center rounded-full bg-primary text-xs font-medium text-primary-foreground sm:h-8 sm:w-8">
            {initials}
          </button>
        </div>
      </header>

      <Dialog open={showNew} onOpenChange={setShowNew}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create New</DialogTitle>
          </DialogHeader>
          <div className="mt-3 space-y-1">
            {NEW_ITEMS.map((item) => (
              <button
                key={item.key}
                onClick={() => {
                  setShowNew(false);
                  router.push(`${item.href}?new=1`);
                }}
                className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-foreground transition-colors hover:bg-muted"
              >
                <item.icon className="h-4 w-4 text-muted-foreground" />
                {item.label}
              </button>
            ))}
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
