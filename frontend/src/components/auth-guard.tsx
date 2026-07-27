"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useRequireAuth } from "@/hooks/use-require-auth";
import { useAuthStore } from "@/stores/auth";

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { isAuthenticated } = useRequireAuth();
  const user = useAuthStore((state) => state.user);

  useEffect(() => {
    if (isAuthenticated && user && !user.onboarding_completed) {
      router.replace("/onboarding");
    }
  }, [isAuthenticated, user, router]);

  if (!isAuthenticated || (user && !user.onboarding_completed)) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-neutral-900 border-t-transparent" />
      </div>
    );
  }

  return <>{children}</>;
}
