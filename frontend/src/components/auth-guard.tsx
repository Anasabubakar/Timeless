"use client";

import { useRequireAuth } from "@/hooks/use-require-auth";

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useRequireAuth();

  if (!isAuthenticated) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-neutral-900 border-t-transparent" />
      </div>
    );
  }

  return <>{children}</>;
}
