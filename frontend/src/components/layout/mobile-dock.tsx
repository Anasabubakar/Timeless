"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Menu } from "lucide-react";
import { cn } from "@/lib/utils";
import { DOCK_ITEMS, type NavItem } from "@/lib/navigation";
import { useMobileNavStore } from "@/stores/mobile-nav";
import { NavDialog } from "./nav-dialog";

function isActive(pathname: string, href: string) {
  return pathname === href || pathname.startsWith(href + "/");
}

function DockLink({ item, active }: { item: NavItem; active: boolean }) {
  return (
    <Link
      href={item.href}
      className={cn(
        "flex min-h-[44px] min-w-[44px] flex-col items-center justify-center gap-0.5 rounded-xl px-2 py-1.5 text-[10px] font-medium transition-colors",
        active ? "text-foreground" : "text-muted-foreground active:text-foreground"
      )}
    >
      <item.icon className={cn("h-5 w-5", active && "text-primary")} />
      {item.name}
    </Link>
  );
}

// Replaces the sidebar on phones/small tablets with a floating glass dock.
// The center button isn't a destination — it's the hub that opens the full
// categorized app menu (NavDialog).
export function MobileDock() {
  const pathname = usePathname();
  const { open, setOpen } = useMobileNavStore();

  const left = DOCK_ITEMS.slice(0, 2);
  const right = DOCK_ITEMS.slice(2, 4);

  return (
    <>
      <nav
        className="fixed inset-x-3 z-40 flex items-center justify-between rounded-2xl border border-white/10 bg-background/70 px-2 py-1.5 shadow-[0_8px_30px_rgba(0,0,0,0.15)] backdrop-blur-xl md:hidden"
        style={{ bottom: "max(0.75rem, calc(env(safe-area-inset-bottom, 0px) + 0.5rem))" }}
      >
        {left.map((item) => (
          <DockLink key={item.href} item={item} active={isActive(pathname, item.href)} />
        ))}

        <button
          onClick={() => setOpen(true)}
          aria-label="Open menu"
          className="relative -mt-7 flex h-14 w-14 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg ring-4 ring-background transition-transform active:scale-95"
        >
          <Menu className="h-6 w-6" />
        </button>

        {right.map((item) => (
          <DockLink key={item.href} item={item} active={isActive(pathname, item.href)} />
        ))}
      </nav>

      <NavDialog open={open} onOpenChange={setOpen} />
    </>
  );
}
