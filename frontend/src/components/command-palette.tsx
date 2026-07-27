"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Command } from "cmdk";
import { Building2, Users, Target, FolderKanban } from "lucide-react";
import { useSearch, type SearchResult } from "@/queries/search";
import { NAVIGATION } from "@/lib/navigation";

// Reuses the same nav list as the sidebar and mobile nav dialog so "go to X"
// commands never drift out of sync with the real routes.
const navigationCommands = NAVIGATION.map((item) => ({
  id: item.href,
  label: `Go to ${item.name}`,
  group: "Navigation" as const,
  action: item.href,
}));

const actionCommands = [
  { id: "new-campaign", label: "Create Campaign", group: "Actions" as const, action: "/campaigns?new=1" },
  { id: "new-sponsor", label: "Add Sponsor", group: "Actions" as const, action: "/sponsors?new=1" },
  { id: "new-company", label: "Add Company", group: "Actions" as const, action: "/companies?new=1" },
];

const commands = [...navigationCommands, ...actionCommands];

const TYPE_ICONS: Record<string, typeof Building2> = {
  sponsor: FolderKanban,
  company: Building2,
  contact: Users,
  campaign: Target,
};

const TYPE_HREFS: Record<string, (id: string) => string> = {
  sponsor: (id) => `/sponsors/${id}`,
  company: (id) => `/companies/${id}`,
  contact: (id) => `/contacts/${id}`,
  campaign: (id) => `/campaigns/${id}`,
};

export function CommandPalette() {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const router = useRouter();

  const { data: searchData } = useSearch(query);
  const searchResults: SearchResult[] = searchData?.data ?? [];

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setOpen((prev) => !prev);
      }
    }
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, []);

  useEffect(() => {
    if (!open) setQuery("");
  }, [open]);

  const runCommand = useCallback(
    (action: string) => {
      setOpen(false);
      router.push(action);
    },
    [router]
  );

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[55]">
      <div
        className="fixed inset-0 bg-black/50 backdrop-blur-sm"
        onClick={() => setOpen(false)}
      />
      <div className="fixed left-1/2 top-[10%] w-full max-w-lg -translate-x-1/2 px-4 sm:top-[20%] sm:px-0">
        <Command className="overflow-hidden rounded-xl border border-border bg-card shadow-2xl">
          <Command.Input
            placeholder="Type a command or search..."
            value={query}
            onValueChange={setQuery}
            className="h-12 w-full border-b border-border bg-transparent px-4 text-base outline-none placeholder:text-muted-foreground sm:h-11 sm:text-sm"
          />
          <Command.List className="max-h-[min(60vh,300px)] overflow-y-auto p-2">
            <Command.Empty className="py-6 text-center text-sm text-muted-foreground">
              No results found.
            </Command.Empty>

            {searchResults.length > 0 && (
              <Command.Group heading="Search Results" className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-muted-foreground">
                {searchResults.map((result) => {
                  const Icon = TYPE_ICONS[result.type] ?? Building2;
                  const getHref = TYPE_HREFS[result.type];
                  return (
                    <Command.Item
                      key={result.id}
                      value={`${result.title} ${result.subtitle ?? ""}`}
                      onSelect={() => getHref && runCommand(getHref(result.id))}
                      className="flex cursor-pointer items-center gap-2 rounded-lg px-2 py-2 text-sm aria-selected:bg-accent"
                    >
                      <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
                      <span className="flex-1 truncate">{result.title}</span>
                      <span className="rounded bg-muted px-1.5 py-0.5 text-[10px] capitalize text-muted-foreground">{result.type}</span>
                    </Command.Item>
                  );
                })}
              </Command.Group>
            )}

            {["Navigation", "Actions"].map((group) => (
              <Command.Group key={group} heading={group} className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-muted-foreground">
                {commands
                  .filter((c) => c.group === group)
                  .map((command) => (
                    <Command.Item
                      key={command.id}
                      value={command.label}
                      onSelect={() => runCommand(command.action)}
                      className="flex cursor-pointer items-center gap-2 rounded-lg px-2 py-2 text-sm aria-selected:bg-accent"
                    >
                      {command.label}
                    </Command.Item>
                  ))}
              </Command.Group>
            ))}
          </Command.List>
        </Command>
      </div>
    </div>
  );
}
