"use client";

import { useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";

// Opens a page's create dialog when the URL carries ?new=1 (used by the
// topbar New menu and the command palette), then strips the param so a
// refresh doesn't re-open the dialog. Reads window.location directly
// instead of useSearchParams to avoid Next 15's Suspense requirement.
export function useNewParam(openCreate: () => void) {
  const pathname = usePathname();
  const router = useRouter();

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get("new") !== "1") return;
    openCreate();
    params.delete("new");
    router.replace(params.size > 0 ? `${pathname}?${params}` : pathname, { scroll: false });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pathname]);
}
