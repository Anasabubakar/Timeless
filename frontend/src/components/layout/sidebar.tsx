"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { Logo } from "@/components/brand/logo";
import { NAVIGATION } from "@/lib/navigation";

// Hidden below md (phones get the bottom dock instead). Collapses to an
// icon-only rail on tablets (md–lg) so it never feels like a stretched
// phone layout, then expands to the full labeled sidebar at lg+.
export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="fixed left-0 top-0 z-40 hidden h-screen w-16 border-r border-border bg-card md:flex md:flex-col lg:w-[240px]">
      <div className="flex h-14 items-center justify-center border-b border-border px-2 lg:justify-start lg:px-5">
        <Logo href="/dashboard" size={28} style="solid" showWordmark={false} priority />
        <span className="ml-2.5 hidden text-sm font-semibold lg:inline">Timeless</span>
      </div>

      <nav className="flex flex-col gap-0.5 p-2 lg:p-3">
        {NAVIGATION.map((item) => {
          const isActive = pathname === item.href || pathname.startsWith(item.href + "/");
          return (
            <Link
              key={item.name}
              href={item.href}
              title={item.name}
              className={cn(
                "flex items-center justify-center gap-2.5 rounded-lg px-3 py-2.5 text-sm transition-colors lg:justify-start lg:py-2",
                isActive
                  ? "bg-accent text-accent-foreground font-medium"
                  : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
              )}
            >
              <item.icon className="h-4 w-4 shrink-0" />
              <span className="hidden lg:inline">{item.name}</span>
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
